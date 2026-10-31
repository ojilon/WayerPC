package session

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"wayerpc/internal/telemetry"
	"wayerpc/internal/wire"
)

// deviceSession is one phone's control link plus its in-flight requests.
type deviceSession struct {
	mgr  *Manager
	did  string
	name string
	ip   string
	ctrl *wire.Conn

	writeMu  sync.Mutex
	lastPong time.Time
	lastSeen time.Time

	mu      sync.Mutex
	pending map[string]*pending
	queries map[string]*serverQuery // phone-initiated numbered lists (flow C)
	closed  bool
}

// pending is one outstanding request started by the PC.
type pending struct {
	id      string
	kind    string
	created time.Time
	file    FileRef
	offset  int64

	matches  chan []wire.Item
	accepted chan bool
	sent     chan doneInfo
	received chan FileRef
	manifest chan wire.Manifest
	errs     chan *wire.Error
}

type doneInfo struct {
	bytes int64
	sha   string
}

func newPending(id, kind string) *pending {
	return &pending{
		id: id, kind: kind, created: time.Now(),
		matches:  make(chan []wire.Item, 1),
		accepted: make(chan bool, 1),
		sent:     make(chan doneInfo, 1),
		received: make(chan FileRef, 1),
		manifest: make(chan wire.Manifest, 32),
		errs:     make(chan *wire.Error, 1),
	}
}

func (d *deviceSession) addPending(p *pending) {
	d.mu.Lock()
	d.pending[p.id] = p
	d.mu.Unlock()
}

func (d *deviceSession) takePending(id string) *pending {
	d.mu.Lock()
	p := d.pending[id]
	delete(d.pending, id)
	d.mu.Unlock()
	return p
}

func (d *deviceSession) pendingByID(id string) *pending {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.pending[id]
}

// send serialises one frame on the control link.
func (d *deviceSession) send(v any) error {
	d.writeMu.Lock()
	defer d.writeMu.Unlock()
	if d.ctrl == nil {
		return errors.New("session: control link closed")
	}
	_ = d.ctrl.Net().SetWriteDeadline(time.Now().Add(30 * time.Second))
	return d.ctrl.WriteJSON(v)
}

// sendErr reports a protocol error for id.
func (d *deviceSession) sendErr(id, code, msg string) {
	_ = d.send(wire.Error{T: wire.TError, ID: id, Code: code, Msg: msg})
}

func (d *deviceSession) info(connected bool) DeviceInfo {
	d.mu.Lock()
	defer d.mu.Unlock()
	return DeviceInfo{
		DID: d.did, Name: d.name, LastSeen: d.lastSeen.Unix(), LastIP: d.ip, Connected: connected,
	}
}

// close tears the control link down (replacement or shutdown).
func (d *deviceSession) close() {
	d.mu.Lock()
	if d.closed {
		d.mu.Unlock()
		return
	}
	d.closed = true
	d.mu.Unlock()
	if d.ctrl != nil {
		_ = d.ctrl.Net().Close()
	}
}

// serverQuery is a numbered list the PC offered to the phone (flow C).
type serverQuery struct {
	id      string
	query   string
	files   []FileRef
	items   []Item
	created time.Time
}

// serveControl runs one control connection to completion.
func (m *Manager) serveControl(wc *wire.Conn, hello wire.Hello, ip string) {
	if hello.DID == "" {
		_ = wc.WriteJSON(wire.Error{T: wire.TError, Code: wire.CodeBadRequest, Msg: "hello needs a did"})
		return
	}
	d := &deviceSession{
		mgr: m, did: hello.DID, name: hello.Device, ip: ip, ctrl: wc,
		lastPong: m.deps.now(), lastSeen: m.deps.now(),
		pending: map[string]*pending{}, queries: map[string]*serverQuery{},
	}

	helloOK := m.helloOK()
	_ = wc.WriteJSON(helloOK)

	m.mu.Lock()
	if old := m.devices[d.did]; old != nil {
		// A second control connection from the same did replaces the first.
		old.close()
		m.deps.logf("[V2] Replacing the control link for %s", deviceLabel(hello))
	}
	m.devices[d.did] = d
	m.mu.Unlock()

	if m.deps.DeviceSeen != nil {
		m.deps.DeviceSeen(d.did, hello.Device, ip)
	}
	m.deps.logf("[V2] Control link up: %s", deviceLabel(hello))
	m.deps.activity(telemetry.KindPhoneConnected, describeDevice(hello, ip), "Connected", deviceLabel(hello))

	defer func() {
		d.close()
		m.mu.Lock()
		if m.devices[d.did] == d {
			delete(m.devices, d.did)
		}
		m.mu.Unlock()
		// Wake every waiter so nobody hangs on a dead link.
		d.mu.Lock()
		for id, p := range d.pending {
			select {
			case p.errs <- &wire.Error{T: wire.TError, ID: id, Code: wire.CodeAborted, Msg: "control link closed"}:
			default:
			}
		}
		d.pending = map[string]*pending{}
		d.mu.Unlock()
		m.deps.logf("[V2] Control link closed: %s", deviceLabel(hello))
		m.deps.activity(telemetry.KindPhoneDisconnect, describeDevice(hello, ip), "Disconnected", deviceLabel(hello))
	}()

	go d.keepalive()

	for {
		_ = wc.Net().SetReadDeadline(m.deps.now().Add(3 * m.deps.pongTimeout()))
		typ, payload, err := wc.ReadMessage()
		if err != nil {
			return
		}
		d.mu.Lock()
		d.lastSeen = m.deps.now()
		d.mu.Unlock()
		if err := d.dispatch(typ, payload); err != nil {
			d.sendErr("", wire.CodeBadRequest, err.Error())
		}
	}
}

// keepalive pings the phone and drops the link after 60 s without a pong.
func (d *deviceSession) keepalive() {
	interval := wire.DefaultPing * time.Second
	if d.mgr.deps.PingInterval > 0 {
		interval = d.mgr.deps.PingInterval
	}
	tick := time.NewTicker(interval)
	defer tick.Stop()
	for range tick.C {
		d.mu.Lock()
		closed := d.closed
		last := d.lastPong
		d.mu.Unlock()
		if closed {
			return
		}
		if d.mgr.deps.now().Sub(last) > d.mgr.deps.pongTimeout() {
			d.mgr.deps.logf("[V2] No pong for %s — dropping the control link", d.name)
			d.close()
			return
		}
		if err := d.send(wire.Ping{T: wire.TPing}); err != nil {
			return
		}
	}
}

// dispatch routes one inbound control message.
func (d *deviceSession) dispatch(typ string, payload []byte) error {
	m := d.mgr
	switch typ {
	case wire.TPing:
		return d.send(wire.Pong{T: wire.TPong})
	case wire.TPong:
		d.mu.Lock()
		d.lastPong = m.deps.now()
		d.mu.Unlock()
		return nil
	case wire.TMatches:
		var msg wire.Matches
		if err := wire.Decode(payload, &msg); err != nil {
			return err
		}
		if p := d.pendingByID(msg.ID); p != nil && p.kind == "query" {
			select {
			case p.matches <- msg.Items:
			default:
			}
		}
		return nil
	case wire.TError:
		var msg wire.Error
		if err := wire.Decode(payload, &msg); err != nil {
			return err
		}
		if p := d.pendingByID(msg.ID); p != nil {
			select {
			case p.errs <- &msg:
			default:
			}
		}
		return nil
	case wire.TAccept:
		var msg wire.Accept
		if err := wire.Decode(payload, &msg); err != nil {
			return err
		}
		if p := d.pendingByID(msg.ID); p != nil {
			select {
			case p.accepted <- true:
			default:
			}
		}
		return nil
	case wire.TReject:
		var msg wire.Reject
		if err := wire.Decode(payload, &msg); err != nil {
			return err
		}
		if p := d.pendingByID(msg.ID); p != nil {
			code := msg.Code
			if code == "" {
				code = wire.CodeDenied
			}
			select {
			case p.errs <- &wire.Error{T: wire.TError, ID: msg.ID, Code: code, Msg: "phone rejected the offer"}:
			default:
			}
		}
		return nil
	case wire.TDone:
		var msg wire.Done
		if err := wire.Decode(payload, &msg); err != nil {
			return err
		}
		if p := d.pendingByID(msg.ID); p != nil {
			select {
			case p.sent <- doneInfo{bytes: msg.Bytes, sha: msg.SHA}:
			default:
			}
		}
		return nil
	case wire.TManifest:
		var msg wire.Manifest
		if err := wire.Decode(payload, &msg); err != nil {
			return err
		}
		if p := d.pendingByID(msg.ID); p != nil && p.kind == "manifest" {
			select {
			case p.manifest <- msg:
			default:
				// The waiter is gone or the phone floods parts; fail the
				// request rather than silently dropping a manifest part.
				select {
				case p.errs <- &wire.Error{T: wire.TError, ID: msg.ID, Code: wire.CodeBusy, Msg: "too many manifest parts"}:
				default:
				}
			}
		}
		return nil
	case wire.TQuery:
		var msg wire.Query
		if err := wire.Decode(payload, &msg); err != nil {
			return err
		}
		d.handlePhoneQuery(msg)
		return nil
	case wire.TPick:
		var msg wire.Pick
		if err := wire.Decode(payload, &msg); err != nil {
			return err
		}
		d.handlePhonePick(msg)
		return nil
	case wire.TAbort:
		var msg wire.Abort
		if err := wire.Decode(payload, &msg); err != nil {
			return err
		}
		d.handlePhoneAbort(msg)
		return nil
	case wire.TPush, wire.TPull:
		return fmt.Errorf("push/pull must go on a data connection")
	default:
		return fmt.Errorf("unsupported message %q", typ)
	}
}

// describeDevice renders "Pixel 8 · 192.168.43.21".
func describeDevice(hello wire.Hello, ip string) string {
	if hello.Device == "" {
		if ip != "" {
			return ip
		}
		return "phone"
	}
	if ip == "" {
		return hello.Device
	}
	return hello.Device + " · " + ip
}

func deviceLabel(hello wire.Hello) string {
	if hello.Device != "" {
		return hello.Device
	}
	if hello.DID != "" {
		return hello.DID
	}
	return "phone"
}

// ---------- PC-initiated operations ----------

// errCode maps a wire error to an error value callers can inspect.
type wireError struct{ werr *wire.Error }

func (e wireError) Error() string {
	if e.werr == nil {
		return "wire error"
	}
	if e.werr.Msg != "" {
		return fmt.Sprintf("%s: %s", e.werr.Code, e.werr.Msg)
	}
	return e.werr.Code
}

// Code exposes the protocol error code ("" for a plain error).
func (e wireError) Code() string {
	if e.werr == nil {
		return ""
	}
	return e.werr.Code
}

// ErrNoDevice means no phone currently holds a control link.
var ErrNoDevice = errors.New("session: no phone connected")

// waitFor blocks on the pending's error channel (or the caller's context /
// the request TTL) and returns whichever arrives first.
func (d *deviceSession) waitFor(ctx context.Context, p *pending, ttl time.Duration) *wire.Error {
	timer := time.NewTimer(ttl)
	defer timer.Stop()
	select {
	case werr := <-p.errs:
		return werr
	case <-ctx.Done():
		return &wire.Error{T: wire.TError, ID: p.id, Code: wire.CodeAborted, Msg: "cancelled"}
	case <-timer.C:
		return &wire.Error{T: wire.TError, ID: p.id, Code: wire.CodeExpired, Msg: "request expired"}
	}
}

// Ask sends a query to the phone and returns its numbered matches.
func (m *Manager) Ask(ctx context.Context, did, query string, max int) (string, []Item, error) {
	d, err := m.session(did)
	if err != nil {
		return "", nil, err
	}
	if max <= 0 || max > wire.MaxMatches {
		max = wire.MaxMatches
	}
	id := m.nextID()
	p := newPending(id, "query")
	d.addPending(p)
	defer d.takePending(id)

	if err := d.send(wire.Query{T: wire.TQuery, ID: id, Q: query, Max: max}); err != nil {
		return id, nil, err
	}
	timer := time.NewTimer(wire.QueryTTL)
	defer timer.Stop()
	select {
	case items := <-p.matches:
		out := make([]Item, 0, len(items))
		for _, it := range items {
			out = append(out, Item{N: it.N, Name: it.Name, Size: it.Size, MTime: it.MTime})
		}
		m.deps.logf("[V2] Phone returned %d match(es) for %q", len(out), query)
		return id, out, nil
	case werr := <-p.errs:
		return id, nil, wireError{werr}
	case <-timer.C:
		return id, nil, wireError{&wire.Error{ID: id, Code: wire.CodeExpired, Msg: "phone did not answer within 120 s"}}
	case <-ctx.Done():
		return id, nil, ctx.Err()
	}
}

// Pick chooses one number from the phone's match list; the phone then opens a
// data connection and pushes that file, which this call waits for.
func (m *Manager) Pick(ctx context.Context, did, id string, n int) (FileRef, error) {
	d, err := m.session(did)
	if err != nil {
		return FileRef{}, err
	}
	p := newPending(id, "push")
	d.addPending(p)
	defer d.takePending(id)

	if err := d.send(wire.Pick{T: wire.TPick, ID: id, N: n}); err != nil {
		return FileRef{}, err
	}
	select {
	case ref := <-p.received:
		return ref, nil
	case werr := <-p.errs:
		return FileRef{}, wireError{werr}
	case <-time.After(wire.QueryTTL):
		return FileRef{}, wireError{&wire.Error{ID: id, Code: wire.CodeExpired, Msg: "phone did not send the file"}}
	case <-ctx.Done():
		return FileRef{}, ctx.Err()
	}
}

// Abort cancels a request on the phone side.
func (m *Manager) Abort(did, id string) error {
	d, err := m.session(did)
	if err != nil {
		return err
	}
	if p := d.takePending(id); p != nil {
		select {
		case p.errs <- &wire.Error{T: wire.TError, ID: id, Code: wire.CodeAborted, Msg: "aborted by PC"}:
		default:
		}
	}
	return d.send(wire.Abort{T: wire.TAbort, ID: id})
}

// Get asks the phone for one known file (sync "pull").
func (m *Manager) Get(ctx context.Context, did, name string) (FileRef, error) {
	d, err := m.session(did)
	if err != nil {
		return FileRef{}, err
	}
	id := m.nextID()
	p := newPending(id, "push")
	d.addPending(p)
	defer d.takePending(id)
	if err := d.send(wire.Get{T: wire.TGet, ID: id, Path: name}); err != nil {
		return FileRef{}, err
	}
	select {
	case ref := <-p.received:
		return ref, nil
	case werr := <-p.errs:
		return FileRef{}, wireError{werr}
	case <-time.After(wire.QueryTTL):
		return FileRef{}, wireError{&wire.Error{ID: id, Code: wire.CodeExpired, Msg: "phone did not send the file"}}
	case <-ctx.Done():
		return FileRef{}, ctx.Err()
	}
}

// Offer proposes a PC file to the phone and waits for the phone to accept and
// pull it (sync "push" and the Files → Send action).
func (m *Manager) Offer(ctx context.Context, did string, ref FileRef) error {
	d, err := m.session(did)
	if err != nil {
		return err
	}
	sha := ""
	if m.deps.Hash != nil {
		if sum, herr := m.deps.Hash(ref.Path); herr == nil {
			sha = sum
		}
	}
	id := m.nextID()
	p := newPending(id, "offer")
	p.file = ref
	d.addPending(p)
	defer d.takePending(id)

	if err := d.send(wire.Offer{T: wire.TOffer, ID: id, Name: ref.Name, Size: ref.Size, MTime: ref.MTime, SHA: sha}); err != nil {
		return err
	}
	timer := time.NewTimer(wire.QueryTTL)
	defer timer.Stop()
	select {
	case <-p.accepted:
	case werr := <-p.errs:
		return wireError{werr}
	case <-timer.C:
		return wireError{&wire.Error{ID: id, Code: wire.CodeExpired, Msg: "phone did not accept the offer"}}
	case <-ctx.Done():
		return ctx.Err()
	}
	// The phone now opens a data connection and pulls.
	select {
	case done := <-p.sent:
		if sha != "" && done.sha != "" && done.sha != sha {
			return wireError{&wire.Error{ID: id, Code: wire.CodeHashMismatch, Msg: "phone reported a different hash"}}
		}
		return nil
	case werr := <-p.errs:
		return wireError{werr}
	case <-timer.C:
		return wireError{&wire.Error{ID: id, Code: wire.CodeExpired, Msg: "phone never pulled the file"}}
	case <-ctx.Done():
		return ctx.Err()
	}
}

// Manifest requests the phone's sync manifest, following multi-part replies.
func (m *Manager) Manifest(ctx context.Context, did, scope string) ([]wire.ManifestItem, error) {
	d, err := m.session(did)
	if err != nil {
		return nil, err
	}
	id := m.nextID()
	p := newPending(id, "manifest")
	d.addPending(p)
	defer d.takePending(id)
	if err := d.send(wire.ManifestReq{T: wire.TManifestReq, ID: id, Scope: scope}); err != nil {
		return nil, err
	}
	var out []wire.ManifestItem
	for {
		select {
		case msg := <-p.manifest:
			out = append(out, msg.Items...)
			if msg.Last {
				sortManifest(out)
				return out, nil
			}
		case werr := <-p.errs:
			return nil, wireError{werr}
		case <-time.After(wire.QueryTTL):
			return nil, wireError{&wire.Error{ID: id, Code: wire.CodeExpired, Msg: "manifest timed out"}}
		case <-ctx.Done():
			return out, ctx.Err()
		}
	}
}

// session looks up a control link by did (empty did = first connected phone).
func (m *Manager) session(did string) (*deviceSession, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if did == "" {
		for _, d := range m.devices {
			if d.ctrl != nil {
				return d, nil
			}
		}
		return nil, ErrNoDevice
	}
	d, ok := m.devices[did]
	if !ok || d.ctrl == nil {
		return nil, ErrNoDevice
	}
	return d, nil
}

// ---------- phone-initiated queries (flow C) ----------

func (d *deviceSession) handlePhoneQuery(msg wire.Query) {
	max := msg.Max
	if max <= 0 || max > wire.MaxMatches {
		max = wire.MaxMatches
	}
	var files []FileRef
	if d.mgr.deps.LocalMatches != nil {
		files = d.mgr.deps.LocalMatches(msg.Q, max)
	}
	entry := RequestEntry{
		ID: msg.ID, TS: d.mgr.deps.now().Unix(), Device: d.name,
		Query: msg.Q, MatchCount: len(files),
	}

	if len(files) == 0 {
		d.sendErr(msg.ID, wire.CodeNotFound, "no matching file")
		entry.Outcome = "no-match"
		d.mgr.recordRequest(entry, nil)
		d.mgr.deps.activity(telemetry.KindError, "phone query · "+msg.Q+" · no matches", "Not found", d.name)
		return
	}

	if len(files) == 1 {
		entry.Outcome = "sent"
		entry.Picked = 1
		entry.PickedName = files[0].Name
		entry.Bundle = []Item{{N: 1, Name: files[0].Name, Size: files[0].Size, MTime: files[0].MTime}}
		d.mgr.recordRequest(entry, nil)
		d.startInlineSend(msg.ID, files[0])
		return
	}

	items := make([]Item, 0, len(files))
	for i, f := range files {
		items = append(items, Item{N: i + 1, Name: f.Name, Size: f.Size, MTime: f.MTime})
	}
	entry.Outcome = "pending"
	entry.Bundle = items
	d.mu.Lock()
	d.queries[msg.ID] = &serverQuery{id: msg.ID, query: msg.Q, files: files, items: items, created: d.mgr.deps.now()}
	d.mu.Unlock()
	d.mgr.recordRequest(entry, nil)

	wireItems := make([]wire.Item, 0, len(items))
	for _, it := range items {
		wireItems = append(wireItems, wire.Item{N: it.N, Name: it.Name, Size: it.Size, MTime: it.MTime})
	}
	if err := d.send(wire.Matches{T: wire.TMatches, ID: msg.ID, TTL: int(wire.QueryTTL / time.Second), Items: wireItems}); err != nil {
		return
	}
	d.mgr.deps.activity(telemetry.KindMatchesSent, fmt.Sprintf("%d match(es) · %s", len(items), msg.Q), "Complete", d.name)
}

func (d *deviceSession) handlePhonePick(msg wire.Pick) {
	d.mu.Lock()
	sq := d.queries[msg.ID]
	if sq != nil {
		delete(d.queries, msg.ID)
	}
	d.mu.Unlock()
	if sq == nil {
		d.sendErr(msg.ID, wire.CodeExpired, "match list expired")
		return
	}
	if msg.N < 1 || msg.N > len(sq.files) {
		d.sendErr(msg.ID, wire.CodeBadRequest, "number out of range")
		return
	}
	ref := sq.files[msg.N-1]
	d.mgr.recordRequest(RequestEntry{
		ID: sq.id, TS: d.mgr.deps.now().Unix(), Device: d.name, Query: sq.query,
		MatchCount: len(sq.files), Picked: msg.N, PickedName: ref.Name,
		Outcome: "sent", Bundle: sq.items,
	}, func(e RequestEntry) { d.mgr.emitRequest(e) })
	d.startInlineSend(msg.ID, ref)
}

func (d *deviceSession) handlePhoneAbort(msg wire.Abort) {
	d.mu.Lock()
	sq := d.queries[msg.ID]
	if sq != nil {
		delete(d.queries, msg.ID)
	}
	d.mu.Unlock()
	_ = d.send(wire.Ok{T: wire.TOk, ID: msg.ID})
	if sq != nil {
		d.mgr.recordRequest(RequestEntry{
			ID: sq.id, TS: d.mgr.deps.now().Unix(), Device: d.name, Query: sq.query,
			MatchCount: len(sq.files), Outcome: "aborted", Bundle: sq.items,
		}, func(e RequestEntry) { d.mgr.emitRequest(e) })
	}
}

// startInlineSend streams one file over the control connection (flow C) in a
// goroutine, because the control loop must keep reading to receive `done`.
func (d *deviceSession) startInlineSend(id string, ref FileRef) {
	p := newPending(id, "inline")
	d.addPending(p)
	go func() {
		defer d.takePending(id)
		if err := d.sendInline(id, ref); err != nil {
			d.mgr.deps.logf("[V2] Sending %s failed: %v", ref.Name, err)
			d.mgr.deps.activity(telemetry.KindError, ref.Name, "Failed", d.name)
			return
		}
		d.mgr.deps.logf("[V2] Sent %s to %s", ref.Name, d.name)
		d.mgr.deps.activity(telemetry.KindFileSent, ref.Name, "Complete", d.name)
	}()
}

// sendInline writes the `file` frame and the raw bytes, holding the write
// lock for the whole stream so keepalive pings can never land mid-file.
func (d *deviceSession) sendInline(id string, ref FileRef) error {
	f, size, sha, err := openForSend(ref, d.mgr.deps.Hash)
	if err != nil {
		d.sendErr(id, wire.CodeNotFound, err.Error())
		return err
	}
	defer f.Close()

	tp := d.mgr.newTransfer(id, telemetry.DirOut, ref.Name, size, d.name)
	d.writeMu.Lock()
	defer d.writeMu.Unlock()
	_ = d.ctrl.Net().SetWriteDeadline(time.Now().Add(5 * time.Minute))
	if err := d.ctrl.WriteJSON(wire.File{T: wire.TFile, ID: id, Name: ref.Name, Size: size, MTime: ref.MTime, SHA: sha}); err != nil {
		tp.finish(telemetry.TransferError)
		return err
	}
	sent, err := d.ctrl.WriteRaw(f, tp.add)
	if err != nil {
		tp.finish(telemetry.TransferError)
		return err
	}
	if sent != size {
		tp.finish(telemetry.TransferError)
		return fmt.Errorf("short send: %d/%d", sent, size)
	}
	tp.finish(telemetry.TransferDone)
	// Wait for the phone's `done` so a hash mismatch is visible here too.
	select {
	case done := <-pendingDone(d, id):
		if sha != "" && done.sha != "" && done.sha != sha {
			return fmt.Errorf("hash mismatch from phone")
		}
	case <-time.After(30 * time.Second):
		return nil // the bytes are gone; do not fail the send over a late ack
	}
	return nil
}

// pendingDone exposes the `sent` channel of a pending by id (nil-safe).
func pendingDone(d *deviceSession, id string) <-chan doneInfo {
	if p := d.pendingByID(id); p != nil {
		return p.sent
	}
	ch := make(chan doneInfo)
	return ch
}

// emitRequest forwards a request-log change to the bridge (nil-safe).
func (m *Manager) emitRequest(e RequestEntry) {
	if m.OnRequest != nil {
		m.OnRequest(e)
	}
}
