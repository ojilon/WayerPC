// Package session implements protocol v2: the PC side of the "two-way
// cooperation" link.
//
// Design rules from the spec (doc/protocol-v2.md):
//   - the PHONE always opens TCP connections;
//   - the PC sends commands over a long-lived control connection per device;
//   - every request carries an `id` chosen by the requester;
//   - data connections are short-lived and carry exactly one transfer.
//
// The package is deliberately free of storage policy: callbacks in Deps answer
// "what does the PC have", "where do uploads go" and "tell the UI". That keeps
// it testable over net.Pipe (see session_test.go).
package session

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"sync"
	"time"

	"wayerpc/internal/telemetry"
	"wayerpc/internal/wire"
)

// FileRef is one PC-side file the session can send.
type FileRef struct {
	Name  string
	Path  string
	Size  int64
	MTime int64
}

// DeviceInfo is one known phone.
type DeviceInfo struct {
	DID       string `json:"did"`
	Name      string `json:"name"`
	LastSeen  int64  `json:"lastSeen"`
	LastIP    string `json:"lastIp"`
	Connected bool   `json:"connected"`
}

// Deps wires the session layer to the rest of the app.
type Deps struct {
	// Logf receives human-readable lines (the UI log).
	Logf func(string)
	// Activity/Transfer push UI updates (both nil-safe).
	Activity func(kind, detail, status, device string)
	Transfer func(telemetry.Transfer)

	// LocalMatches answers a phone query with PC files (already ranked).
	LocalMatches func(query string, max int) []FileRef
	// ResolveName maps a phone-supplied bare name to a PC file.
	ResolveName func(name string) (FileRef, bool)
	// ReceiveDir is where phone pushes land ("" disables receiving).
	ReceiveDir func() string
	// Uploaded is called after a successful push (catalog + index + UI).
	Uploaded func(path string)
	// Hash returns the sha256 of a file, using the lazy cache.
	Hash func(path string) (string, error)
	// DeviceSeen persists device information.
	DeviceSeen func(did, name, ip string)
	// PCName is advertised in hello_ok.
	PCName string
	// Now is injectable for TTL tests (defaults to time.Now).
	Now func() time.Time
	// PingInterval overrides the advertised keepalive (tests use ~50ms).
	PingInterval time.Duration
	// PongTimeout overrides how long a silent control link survives.
	PongTimeout time.Duration
}

// pongTimeout is the control-link silence budget.
func (d Deps) pongTimeout() time.Duration {
	if d.PongTimeout > 0 {
		return d.PongTimeout
	}
	return wire.PongTimeout
}

func (d Deps) now() time.Time {
	if d.Now != nil {
		return d.Now()
	}
	return time.Now()
}

func (d Deps) logf(format string, args ...any) {
	if d.Logf != nil {
		d.Logf(fmt.Sprintf(format, args...))
	}
}

func (d Deps) activity(kind, detail, status, device string) {
	if d.Activity != nil {
		d.Activity(kind, detail, status, device)
	}
}

func (d Deps) transfer(t telemetry.Transfer) {
	if d.Transfer != nil {
		d.Transfer(t)
	}
}

// Manager owns every v2 connection and the per-device control links.
type Manager struct {
	deps Deps

	// OnRequest is called whenever a request-log row changes (bridge hook).
	OnRequest func(RequestEntry)

	mu       sync.Mutex
	devices  map[string]*deviceSession
	seq      int64
	dataLive int
	// serverQueries tracks phone-initiated numbered lists (flow C).
	serverQueries map[string]*serverQuery
	requests      []RequestEntry
}

// New builds a manager.
func New(deps Deps) *Manager {
	if deps.Now == nil {
		deps.Now = time.Now
	}
	if deps.PCName == "" {
		deps.PCName, _ = os.Hostname()
	}
	return &Manager{
		deps:          deps,
		devices:       map[string]*deviceSession{},
		serverQueries: map[string]*serverQuery{},
	}
}

// RequestEntry is one row of the "Incoming request" log (phone-initiated
// queries only — PC-initiated ones are shown by the Ask phone stepper).
type RequestEntry struct {
	ID         string `json:"id"`
	TS         int64  `json:"ts"`
	Device     string `json:"device"`
	Query      string `json:"query"`
	MatchCount int    `json:"matchCount"`
	Picked     int    `json:"picked,omitempty"`
	PickedName string `json:"pickedName,omitempty"`
	Outcome    string `json:"outcome"` // "sent" | "no-match" | "aborted" | "error"
	Bundle     []Item `json:"bundle,omitempty"`
}

// Item is a numbered match as shown to the phone (and in the request log).
type Item struct {
	N     int    `json:"n"`
	Name  string `json:"name"`
	Size  int64  `json:"size"`
	MTime int64  `json:"mtime"`
}

// nextID returns a request id from the PC's namespace ("pc-24").
func (m *Manager) nextID() string {
	m.mu.Lock()
	m.seq++
	n := m.seq
	m.mu.Unlock()
	return "pc-" + itoa64(n)
}

// ServeV2 is the server.V2Handler entry point (kept as a separate name so the
// server package never has to import session).
func (m *Manager) ServeV2(conn net.Conn, br *bufio.Reader) { m.Serve(conn, br) }

// Serve handles one accepted v2 connection (the caller has already peeked
// the first byte and seen '@').
func (m *Manager) Serve(conn net.Conn, br *bufio.Reader) {
	wc := wire.NewConn(conn, br)
	// A v2 connection must start with hello; bound the wait so a stalled
	// socket cannot pin a goroutine forever.
	_ = conn.SetReadDeadline(m.deps.now().Add(30 * time.Second))
	typ, payload, err := wc.ReadMessage()
	if err != nil {
		m.deps.logf("[V2] handshake failed from %s: %v", hostOf(conn), err)
		return
	}
	if typ != wire.THello {
		_ = wc.WriteJSON(wire.Error{T: wire.TError, Code: wire.CodeBadRequest, Msg: "expected hello"})
		m.deps.logf("[V2] first frame was %q, not hello", typ)
		return
	}
	var hello wire.Hello
	if err := wire.Decode(payload, &hello); err != nil {
		_ = wc.WriteJSON(wire.Error{T: wire.TError, Code: wire.CodeBadRequest, Msg: "invalid hello"})
		return
	}
	if hello.Proto != wire.ProtoVersion {
		_ = wc.WriteJSON(wire.Error{T: wire.TError, Code: wire.CodeUnsupported, Msg: "unsupported proto"})
		m.deps.logf("[V2] phone asked for proto %d (PC speaks %d)", hello.Proto, wire.ProtoVersion)
		return
	}
	_ = conn.SetReadDeadline(time.Time{})

	switch hello.Role {
	case "control":
		m.serveControl(wc, hello, hostOf(conn))
	case "data":
		m.serveData(wc, hello, hostOf(conn))
	default:
		_ = wc.WriteJSON(wire.Error{T: wire.TError, Code: wire.CodeBadRequest, Msg: "unknown role"})
	}
}

// helloOK builds the handshake answer.
func (m *Manager) helloOK() wire.HelloOK {
	ping := int(wire.DefaultPing)
	if m.deps.PingInterval > 0 {
		ping = int(m.deps.PingInterval / time.Second)
		if ping <= 0 {
			ping = 1
		}
	}
	return wire.HelloOK{
		T: wire.THelloOK, Proto: wire.ProtoVersion, PC: m.deps.PCName,
		Caps:     []string{"query", "manifest", "offer", "get", "sha256"},
		MaxFrame: wire.MaxFrame, Ping: ping,
	}
}

// ControlLinks reports how many devices currently hold a control link.
func (m *Manager) ControlLinks() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	n := 0
	for _, d := range m.devices {
		if d.ctrl != nil {
			n++
		}
	}
	return n
}

// ConnectedDevice returns the first device with a live control link.
func (m *Manager) ConnectedDevice() (DeviceInfo, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, d := range m.devices {
		if d.ctrl != nil {
			return d.info(true), true
		}
	}
	return DeviceInfo{}, false
}

// Devices lists every known phone (connected first, then most recent).
func (m *Manager) Devices() []DeviceInfo {
	m.mu.Lock()
	out := make([]DeviceInfo, 0, len(m.devices))
	for _, d := range m.devices {
		out = append(out, d.info(d.ctrl != nil))
	}
	m.mu.Unlock()
	// Connected first, then newest lastSeen.
	for i := 1; i < len(out); i++ {
		for j := i; j > 0; j-- {
			if out[j].Connected && !out[j-1].Connected ||
				out[j].Connected == out[j-1].Connected && out[j].LastSeen > out[j-1].LastSeen {
				out[j], out[j-1] = out[j-1], out[j]
			}
		}
	}
	return out
}

// Requests returns the phone-initiated request log (newest first).
func (m *Manager) Requests() []RequestEntry {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]RequestEntry, 0, len(m.requests))
	for i := len(m.requests) - 1; i >= 0; i-- {
		out = append(out, m.requests[i])
	}
	return out
}

// recordRequest prepends one request-log row (ring of 200).
func (m *Manager) recordRequest(e RequestEntry, update func(RequestEntry)) {
	m.mu.Lock()
	replaced := false
	for i := range m.requests {
		if m.requests[i].ID == e.ID {
			m.requests[i] = e
			replaced = true
			break
		}
	}
	if !replaced {
		m.requests = append(m.requests, e)
		if len(m.requests) > 200 {
			m.requests = m.requests[len(m.requests)-200:]
		}
	}
	m.mu.Unlock()
	if update != nil {
		update(e)
	}
}

// dataSlots reserves one of the 4 concurrent data connections.
func (m *Manager) dataSlots() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.dataLive >= wire.MaxDataConns {
		return false
	}
	m.dataLive++
	return true
}

func (m *Manager) releaseDataSlot() {
	m.mu.Lock()
	if m.dataLive > 0 {
		m.dataLive--
	}
	m.mu.Unlock()
}

// hostOf returns the peer IP ("" for net.Pipe).
func hostOf(conn net.Conn) string {
	if conn == nil || conn.RemoteAddr() == nil {
		return ""
	}
	addr := conn.RemoteAddr().String()
	if h, _, err := net.SplitHostPort(addr); err == nil {
		return h
	}
	return addr
}

func itoa64(n int64) string {
	if n == 0 {
		return "0"
	}
	var buf [20]byte
	i := len(buf)
	neg := n < 0
	if neg {
		n = -n
	}
	for n > 0 && i > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	if neg && i > 0 {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}
