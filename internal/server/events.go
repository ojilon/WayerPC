package server

// Transfer progress + structured activity, shared by the v1 handlers and the
// v2 session layer. Both are opt-in side channels: the wire protocol is
// untouched, and a Server without an Observer behaves exactly as before
// (this is what keeps every pre-existing test green).

import (
	"net"
	"sync"
	"sync/atomic"
	"time"

	"wayerpc/internal/telemetry"
)

// The transfer/activity payloads live in internal/telemetry so the v2
// session layer can emit the same shapes; these aliases keep the v1 call
// sites (and the existing tests) unchanged.
type Transfer = telemetry.Transfer

// Activity is one structured row for the Console activity table.
type Activity = telemetry.Activity

// Observer receives transfer + activity pushes. Implementations must be
// cheap and must not block: they are called from connection goroutines.
type Observer = telemetry.Observer

// Transfer states.
const (
	TransferActive    = telemetry.TransferActive
	TransferDone      = telemetry.TransferDone
	TransferError     = telemetry.TransferError
	TransferCancelled = telemetry.TransferCancelled
)

// Transfer directions.
const (
	DirIn  = telemetry.DirIn
	DirOut = telemetry.DirOut
)

// Activity kinds.
const (
	KindServerStarted   = telemetry.KindServerStarted
	KindServerStopped   = telemetry.KindServerStopped
	KindPhoneConnected  = telemetry.KindPhoneConnected
	KindPhoneDisconnect = telemetry.KindPhoneDisconnect
	KindFileSent        = telemetry.KindFileSent
	KindFileReceived    = telemetry.KindFileReceived
	KindMatchesSent     = telemetry.KindMatchesSent
	KindError           = telemetry.KindError
	KindImport          = telemetry.KindImport
)

// SetObserver installs (or clears, with nil) the UI observer.
func (s *Server) SetObserver(o Observer) {
	s.obsMu.Lock()
	s.obs = o
	s.obsMu.Unlock()
}

func (s *Server) observer() Observer {
	s.obsMu.RLock()
	defer s.obsMu.RUnlock()
	return s.obs
}

// emitActivity pushes one structured row (no-op without an observer).
func (s *Server) emitActivity(kind, detail, status, device string) {
	o := s.observer()
	if o == nil {
		return
	}
	o.Activity(Activity{TS: time.Now().Unix(), Kind: kind, Detail: detail, Status: status, Device: device})
}

// transferSeq numbers v1 transfers ("v1-7"); the session layer uses its own
// request ids so the two never collide.
var transferSeq int64

func nextTransferID() string {
	n := atomic.AddInt64(&transferSeq, 1)
	return "v1-" + itoa64(n)
}

func itoa64(n int64) string {
	if n == 0 {
		return "0"
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 && i > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	return string(buf[i:])
}

// transferProgress tracks one transfer: it throttles progress to ~4/s and
// parks the connection in the cancel registry so the UI can abort it.
type transferProgress struct {
	srv     *Server
	id      string
	dir     string
	name    string
	total   int64
	device  string
	conn    net.Conn
	start   time.Time
	lastAt  time.Time
	lastPos int64
	bps     int64

	mu      sync.Mutex
	bytes   int64
	settled bool
	cancel  bool
}

// startTransfer registers a transfer + its cancel handle and emits the
// initial `active` frame.
func (s *Server) startTransfer(conn net.Conn, dir, name string, total int64, device string) *transferProgress {
	t := &transferProgress{
		srv: s, id: nextTransferID(), dir: dir, name: name, total: total,
		device: device, conn: conn, start: time.Now(),
	}
	s.registerCancel(t.id, conn, func() { t.markCancelled() })
	t.emit(TransferActive)
	return t
}

func (t *transferProgress) markCancelled() {
	t.mu.Lock()
	t.cancel = true
	t.mu.Unlock()
}

func (t *transferProgress) isCancelled() bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.cancel
}

// add records progress and emits a throttled update (≥250 ms apart).
func (t *transferProgress) add(n int64) {
	t.mu.Lock()
	t.bytes += n
	pos := t.bytes
	t.mu.Unlock()
	now := time.Now()
	if now.Sub(t.lastAt) < 250*time.Millisecond {
		return
	}
	if elapsed := now.Sub(t.lastAt); elapsed > 0 {
		t.bps = int64(float64(pos-t.lastPos) / elapsed.Seconds())
	}
	t.lastAt = now
	t.lastPos = pos
	t.emit(TransferActive)
}

// finish emits the terminal frame exactly once and releases the cancel slot.
func (t *transferProgress) finish(state string) {
	t.mu.Lock()
	if t.settled {
		t.mu.Unlock()
		return
	}
	t.settled = true
	pos := t.bytes
	if pos > t.total && t.total > 0 {
		pos = t.total
	}
	if t.cancel && state == TransferError {
		state = TransferCancelled
	}
	t.mu.Unlock()
	t.srv.unregisterCancel(t.id)
	now := time.Now()
	if d := now.Sub(t.start).Seconds(); d > 0 {
		t.bps = int64(float64(pos) / d)
	}
	o := t.srv.observer()
	if o != nil {
		o.Transfer(Transfer{
			ID: t.id, Dir: t.dir, Name: t.name, Bytes: pos, Total: t.total,
			BPS: t.bps, State: state, Device: t.device,
		})
	}
}

func (t *transferProgress) emit(state string) {
	o := t.srv.observer()
	if o == nil {
		return
	}
	t.mu.Lock()
	pos := t.bytes
	t.mu.Unlock()
	o.Transfer(Transfer{
		ID: t.id, Dir: t.dir, Name: t.name, Bytes: pos, Total: t.total,
		BPS: t.bps, State: state, Device: t.device,
	})
}

// ---------- cancel registry ----------

// registerCancel records how to abort one transfer.
func (s *Server) registerCancel(id string, conn net.Conn, onCancel func()) {
	s.cancelMu.Lock()
	if s.cancels == nil {
		s.cancels = make(map[string]cancelEntry)
	}
	s.cancels[id] = cancelEntry{conn: conn, onCancel: onCancel}
	s.cancelMu.Unlock()
}

func (s *Server) unregisterCancel(id string) {
	s.cancelMu.Lock()
	delete(s.cancels, id)
	s.cancelMu.Unlock()
}

type cancelEntry struct {
	conn     net.Conn
	onCancel func()
}

// Cancel aborts one in-flight transfer by closing its connection. It
// reports whether such a transfer existed. Safe to call from the UI thread.
func (s *Server) Cancel(id string) bool {
	s.cancelMu.Lock()
	entry, ok := s.cancels[id]
	if ok {
		delete(s.cancels, id)
	}
	s.cancelMu.Unlock()
	if !ok {
		return false
	}
	if entry.onCancel != nil {
		entry.onCancel()
	}
	if entry.conn != nil {
		_ = entry.conn.Close()
	}
	return true
}

// ActiveTransfers lists the in-flight transfer ids (tests + UI).
func (s *Server) ActiveTransfers() []string {
	s.cancelMu.Lock()
	defer s.cancelMu.Unlock()
	out := make([]string, 0, len(s.cancels))
	for id := range s.cancels {
		out = append(out, id)
	}
	return out
}

// hostOf returns the peer IP ("" for net.Pipe connections in tests).
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

// humanBytes renders a short size for activity details ("1.73 MB").
func humanBytes(n int64) string {
	const unit = 1024
	if n < unit {
		return itoa64(n) + " B"
	}
	div, exp := int64(unit), 0
	for v := n / unit; v >= unit && exp < 3; v /= unit {
		div *= unit
		exp++
	}
	val := float64(n) / float64(div)
	suffix := []string{"KB", "MB", "GB", "TB"}[exp]
	// one decimal, trimmed
	whole := int64(val)
	frac := int64((val - float64(whole)) * 100)
	if frac < 0 {
		frac = -frac
	}
	if frac == 0 {
		return itoa64(whole) + " " + suffix
	}
	if frac%10 == 0 {
		return itoa64(whole) + "." + itoa64(frac/10) + " " + suffix
	}
	return itoa64(whole) + "." + itoa64(frac) + " " + suffix
}
