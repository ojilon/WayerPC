package session

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"wayerpc/internal/telemetry"
	"wayerpc/internal/wire"
)

// serveData runs one short-lived data connection: hello(role=data) then
// exactly one push (phone → PC) or pull (PC → phone).
func (m *Manager) serveData(wc *wire.Conn, hello wire.Hello, ip string) {
	if !m.dataSlots() {
		_ = wc.WriteJSON(wire.Error{T: wire.TError, Code: wire.CodeBusy, Msg: "too many concurrent transfers"})
		return
	}
	defer m.releaseDataSlot()

	if err := wc.WriteJSON(m.helloOK()); err != nil {
		return
	}
	d, err := m.session(hello.DID)
	if err != nil {
		_ = wc.WriteJSON(wire.Error{T: wire.TError, Code: wire.CodeBadRequest, Msg: "no control link for this did"})
		return
	}

	for {
		_ = wc.Net().SetReadDeadline(m.deps.now().Add(5 * time.Minute))
		typ, payload, rerr := wc.ReadMessage()
		if rerr != nil {
			return
		}
		switch typ {
		case wire.TPing:
			_ = wc.WriteJSON(wire.Pong{T: wire.TPong})
			continue
		case wire.TPush:
			var msg wire.Push
			if derr := wire.Decode(payload, &msg); derr != nil {
				_ = wc.WriteJSON(wire.Error{T: wire.TError, Code: wire.CodeBadRequest, Msg: "invalid push"})
				return
			}
			m.handlePush(wc, d, msg)
			return
		case wire.TPull:
			var msg wire.Pull
			if derr := wire.Decode(payload, &msg); derr != nil {
				_ = wc.WriteJSON(wire.Error{T: wire.TError, Code: wire.CodeBadRequest, Msg: "invalid pull"})
				return
			}
			m.handlePull(wc, d, msg)
			return
		default:
			_ = wc.WriteJSON(wire.Error{T: wire.TError, Code: wire.CodeUnsupported, Msg: "data connections carry one push or pull"})
			return
		}
	}
}

// handlePush receives one file from the phone: ready/offset negotiation,
// streaming with a running sha256, resume, collision-safe naming.
func (m *Manager) handlePush(wc *wire.Conn, d *deviceSession, msg wire.Push) {
	pend := d.pendingByID(msg.ID)
	done := func(ref FileRef) {
		if pend == nil || pend.kind != "push" {
			return
		}
		select {
		case pend.received <- ref:
		default:
		}
	}
	failed := func(code, detail string) {
		if pend == nil || pend.kind != "push" {
			return
		}
		select {
		case pend.errs <- &wire.Error{T: wire.TError, ID: msg.ID, Code: code, Msg: detail}:
		default:
		}
	}

	name, err := wire.SanitizeName(msg.Name)
	if err != nil {
		_ = wc.WriteJSON(wire.Error{T: wire.TError, ID: msg.ID, Code: wire.CodeBadRequest, Msg: err.Error()})
		failed(wire.CodeBadRequest, err.Error())
		return
	}
	if msg.Size < 0 || msg.Size > wire.MaxUploadBytes {
		_ = wc.WriteJSON(wire.Error{T: wire.TError, ID: msg.ID, Code: wire.CodeTooLarge, Msg: "upload exceeds 2 GiB"})
		failed(wire.CodeTooLarge, "upload exceeds 2 GiB")
		return
	}
	dir := ""
	if m.deps.ReceiveDir != nil {
		dir = m.deps.ReceiveDir()
	}
	if dir == "" {
		_ = wc.WriteJSON(wire.Error{T: wire.TError, ID: msg.ID, Code: wire.CodeInternal, Msg: "received folder unavailable"})
		failed(wire.CodeInternal, "received folder unavailable")
		return
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		_ = wc.WriteJSON(wire.Error{T: wire.TError, ID: msg.ID, Code: wire.CodeInternal, Msg: "received folder unavailable"})
		failed(wire.CodeInternal, "received folder unavailable")
		return
	}
	m.cleanPartials(dir)

	partPath, have := m.partialState(dir, name, msg.SHA, msg.Size)
	// Never claim more than the sender offered to send.
	if have > msg.Size {
		have = 0
	}
	if err := wc.WriteJSON(wire.Ready{T: wire.TReady, ID: msg.ID, Offset: have}); err != nil {
		return
	}

	// Resuming needs read access (to hash what is already on disk) *and* an
	// explicit seek, so O_RDWR + seek beats O_APPEND here.
	flags := os.O_CREATE | os.O_RDWR
	if have == 0 {
		flags |= os.O_TRUNC
	}
	f, err := os.OpenFile(partPath, flags, 0o644)
	if err != nil {
		_ = wc.WriteJSON(wire.Error{T: wire.TError, ID: msg.ID, Code: wire.CodeInternal, Msg: "cannot write the received file"})
		failed(wire.CodeInternal, "cannot write the received file")
		return
	}

	hasher := sha256.New()
	if have > 0 {
		// Hash the bytes already on disk, then continue writing after them.
		if _, err := io.CopyN(hasher, f, have); err != nil {
			_ = f.Close()
			_ = wc.WriteJSON(wire.Error{T: wire.TError, ID: msg.ID, Code: wire.CodeInternal, Msg: "resume state unreadable"})
			_ = os.Remove(partPath)
			failed(wire.CodeInternal, "resume state unreadable")
			return
		}
		if _, err := f.Seek(have, io.SeekStart); err != nil {
			_ = f.Close()
			_ = os.Remove(partPath)
			failed(wire.CodeInternal, "resume state unreadable")
			return
		}
	}

	tp := m.newTransfer(msg.ID, telemetry.DirIn, name, msg.Size, d.name)
	tp.bytes = have
	tp.emit(telemetry.TransferActive)

	writer := io.MultiWriter(f, hasher)
	read, rerr := wc.ReadRaw(writer, msg.Size-have, tp.add)
	closeErr := f.Close()
	if rerr != nil || closeErr != nil || read != msg.Size-have {
		// Keep the partial around when we know its hash (resume later).
		if msg.SHA == "" {
			_ = os.Remove(partPath)
		}
		tp.finish(telemetry.TransferError)
		_ = wc.WriteJSON(wire.Error{T: wire.TError, ID: msg.ID, Code: wire.CodeAborted, Msg: "transfer interrupted"})
		failed(wire.CodeAborted, "transfer interrupted")
		m.deps.activity(telemetry.KindError, name+" · interrupted", "Failed", d.name)
		return
	}

	sum := hex.EncodeToString(hasher.Sum(nil))
	if msg.SHA != "" && !strings.EqualFold(sum, msg.SHA) {
		_ = os.Remove(partPath)
		tp.finish(telemetry.TransferError)
		_ = wc.WriteJSON(wire.Error{T: wire.TError, ID: msg.ID, Code: wire.CodeHashMismatch, Msg: "sha256 mismatch"})
		failed(wire.CodeHashMismatch, "sha256 mismatch")
		m.deps.logf("[V2] Hash mismatch receiving %s — partial deleted", name)
		m.deps.activity(telemetry.KindError, name+" · hash mismatch", "Failed", d.name)
		return
	}

	final := uniquePath(dir, name)
	if err := os.MkdirAll(filepath.Dir(final), 0o755); err != nil {
		tp.finish(telemetry.TransferError)
		return
	}
	if err := os.Rename(partPath, final); err != nil {
		tp.finish(telemetry.TransferError)
		_ = wc.WriteJSON(wire.Error{T: wire.TError, ID: msg.ID, Code: wire.CodeInternal, Msg: "cannot store the file"})
		failed(wire.CodeInternal, "cannot store the file")
		return
	}
	_ = wc.WriteJSON(wire.Done{T: wire.TDone, ID: msg.ID, Bytes: msg.Size, SHA: sum})
	tp.finish(telemetry.TransferDone)
	done(FileRef{Name: filepath.Base(final), Path: final, Size: msg.Size, MTime: m.deps.now().Unix()})

	m.deps.logf("[V2] Received %s (%d bytes) from %s", filepath.Base(final), msg.Size, d.name)
	m.deps.activity(telemetry.KindFileReceived, filepath.Base(final), "Complete", d.name)
	if m.deps.Uploaded != nil {
		m.deps.Uploaded(final)
	}
}

// handlePull serves an accepted offer: `file` frame then raw bytes, then the
// phone's `done` (verified against the hash we announced).
func (m *Manager) handlePull(wc *wire.Conn, d *deviceSession, msg wire.Pull) {
	p := d.pendingByID(msg.ID)
	if p == nil || p.kind != "offer" {
		_ = wc.WriteJSON(wire.Error{T: wire.TError, ID: msg.ID, Code: wire.CodeExpired, Msg: "no accepted offer with that id"})
		return
	}
	f, size, sha, err := openForSend(p.file, m.deps.Hash)
	if err != nil {
		_ = wc.WriteJSON(wire.Error{T: wire.TError, ID: msg.ID, Code: wire.CodeNotFound, Msg: err.Error()})
		select {
		case p.errs <- &wire.Error{T: wire.TError, ID: msg.ID, Code: wire.CodeNotFound, Msg: err.Error()}:
		default:
		}
		return
	}
	defer f.Close()

	offset := msg.Offset
	if offset < 0 || offset > size {
		offset = 0
	}
	if offset > 0 {
		if _, err := f.Seek(offset, io.SeekStart); err != nil {
			offset = 0
			_, _ = f.Seek(0, io.SeekStart)
		}
	}

	tp := m.newTransfer(msg.ID, telemetry.DirOut, p.file.Name, size, d.name)
	tp.bytes = offset
	tp.emit(telemetry.TransferActive)

	if err := wc.WriteJSON(wire.File{T: wire.TFile, ID: msg.ID, Name: p.file.Name, Size: size, MTime: p.file.MTime, SHA: sha}); err != nil {
		tp.finish(telemetry.TransferError)
		return
	}
	sent, werr := wc.WriteRaw(f, tp.add)
	if werr != nil || sent != size-offset {
		tp.finish(telemetry.TransferError)
		return
	}
	tp.finish(telemetry.TransferDone)

	// The phone confirms on the same connection.
	_ = wc.Net().SetReadDeadline(time.Now().Add(30 * time.Second))
	typ, payload, rerr := wc.ReadMessage()
	var done wire.Done
	if rerr == nil && typ == wire.TDone {
		_ = wire.Decode(payload, &done)
	}
	if rerr == nil && sha != "" && done.SHA != "" && !strings.EqualFold(done.SHA, sha) {
		select {
		case p.errs <- &wire.Error{T: wire.TError, ID: msg.ID, Code: wire.CodeHashMismatch, Msg: "phone reported a different hash"}:
		default:
		}
		m.deps.activity(telemetry.KindError, p.file.Name+" · hash mismatch", "Failed", d.name)
		return
	}
	select {
	case p.sent <- doneInfo{bytes: done.Bytes, sha: done.SHA}:
	default:
	}
	m.deps.logf("[V2] Sent %s to %s", p.file.Name, d.name)
	m.deps.activity(telemetry.KindFileSent, p.file.Name, "Complete", d.name)
}

// ---------- filesystem helpers ----------

// openForSend opens a PC file and returns it with its size + sha256.
func openForSend(ref FileRef, hash func(string) (string, error)) (*os.File, int64, string, error) {
	f, err := os.Open(ref.Path)
	if err != nil {
		return nil, 0, "", err
	}
	st, err := f.Stat()
	if err != nil {
		f.Close()
		return nil, 0, "", err
	}
	if st.IsDir() {
		f.Close()
		return nil, 0, "", fmt.Errorf("not a file: %s", ref.Path)
	}
	sha := ""
	if hash != nil {
		if sum, herr := hash(ref.Path); herr == nil {
			sha = sum
		}
	}
	return f, st.Size(), sha, nil
}

// partialState decides where a push is written and how much is already there.
// Resume only ever happens when the sender supplied a sha256 (the partial file
// is keyed by that hash).
func (m *Manager) partialState(dir, name, sha string, size int64) (string, int64) {
	if sha == "" {
		// No hash → no resume; a fresh .part file (same as v1).
		path := filepath.Join(dir, name+".part")
		_ = os.Remove(path)
		return path, 0
	}
	partDir := filepath.Join(dir, ".partial")
	if err := os.MkdirAll(partDir, 0o755); err != nil {
		return filepath.Join(dir, name+".part"), 0
	}
	path := filepath.Join(partDir, strings.ToLower(sha)+".part")
	st, err := os.Stat(path)
	if err != nil {
		return path, 0
	}
	if st.Size() > size {
		_ = os.Remove(path)
		return path, 0
	}
	return path, st.Size()
}

// cleanPartials drops resume state older than 24 h (called on each receive).
func (m *Manager) cleanPartials(dir string) {
	partDir := filepath.Join(dir, ".partial")
	entries, err := os.ReadDir(partDir)
	if err != nil {
		return
	}
	cutoff := m.deps.now().Add(-24 * time.Hour)
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		info, ierr := e.Info()
		if ierr != nil {
			continue
		}
		if info.ModTime().Before(cutoff) {
			_ = os.Remove(filepath.Join(partDir, e.Name()))
		}
	}
}

// uniquePath implements the keep-both collision policy: "name (1).ext".
func uniquePath(dir, name string) string {
	path := filepath.Join(dir, name)
	if _, err := os.Stat(path); err != nil {
		return path
	}
	ext := filepath.Ext(name)
	base := strings.TrimSuffix(name, ext)
	for i := 1; i < 10000; i++ {
		cand := filepath.Join(dir, fmt.Sprintf("%s (%d)%s", base, i, ext))
		if _, err := os.Stat(cand); err != nil {
			return cand
		}
	}
	return path
}

// ---------- transfer progress ----------

// xferProgress throttles per-transfer UI updates to ~4/s.
type xferProgress struct {
	mgr    *Manager
	id     string
	dir    string
	name   string
	device string
	total  int64

	start   time.Time
	lastAt  time.Time
	lastPos int64
	bps     int64

	mu      sync.Mutex
	bytes   int64
	settled bool
}

func (m *Manager) newTransfer(id, dir, name string, total int64, device string) *xferProgress {
	return &xferProgress{
		mgr: m, id: id, dir: dir, name: name, device: device, total: total,
		start: m.deps.now(), lastAt: m.deps.now(),
	}
}

func (t *xferProgress) add(n int64) {
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
	t.emit(telemetry.TransferActive)
}

func (t *xferProgress) emit(state string) {
	t.mu.Lock()
	pos := t.bytes
	t.mu.Unlock()
	t.mgr.deps.transfer(telemetry.Transfer{
		ID: t.id, Dir: t.dir, Name: t.name, Bytes: pos, Total: t.total,
		BPS: t.bps, State: state, Device: t.device,
	})
}

func (t *xferProgress) finish(state string) {
	t.mu.Lock()
	if t.settled {
		t.mu.Unlock()
		return
	}
	t.settled = true
	pos := t.bytes
	t.mu.Unlock()
	if d := time.Since(t.start).Seconds(); d > 0 && t.bps == 0 {
		t.bps = int64(float64(pos) / d)
	}
	t.emit(state)
}

// sortManifest keeps manifests deterministic (tests + UI).
func sortManifest(items []wire.ManifestItem) {
	sort.Slice(items, func(i, j int) bool { return items[i].P < items[j].P })
}
