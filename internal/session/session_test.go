package session

import (
	"bufio"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"net"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"wayerpc/internal/telemetry"
	"wayerpc/internal/wire"
)

// ---------- harness ----------

type harness struct {
	m      *Manager
	pcDir  string // received/
	shared []FileRef
	logs   []string
	mu     sync.Mutex
	acts   []telemetry.Activity
	xfer   []telemetry.Transfer

	// phone side of the control connection
	ctrl  net.Conn
	hello wire.Hello
}

func shaOf(b []byte) string {
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

// newHarness starts a manager and completes the control handshake.
func newHarness(t *testing.T, tweak func(*Deps)) *harness {
	t.Helper()
	dir := t.TempDir()
	received := filepath.Join(dir, "received")
	if err := os.MkdirAll(received, 0o755); err != nil {
		t.Fatal(err)
	}
	h := &harness{pcDir: received}
	deps := Deps{
		Logf: func(line string) {
			h.mu.Lock()
			h.logs = append(h.logs, line)
			h.mu.Unlock()
		},
		Activity: func(kind, detail, status, device string) {
			h.mu.Lock()
			h.acts = append(h.acts, telemetry.Activity{Kind: kind, Detail: detail, Status: status, Device: device})
			h.mu.Unlock()
		},
		Transfer: func(t telemetry.Transfer) {
			h.mu.Lock()
			h.xfer = append(h.xfer, t)
			h.mu.Unlock()
		},
		LocalMatches: func(query string, max int) []FileRef {
			var out []FileRef
			for _, f := range h.shared {
				if strings.Contains(strings.ToLower(f.Name), strings.ToLower(query)) {
					out = append(out, f)
					if len(out) >= max {
						break
					}
				}
			}
			return out
		},
		ResolveName: func(name string) (FileRef, bool) {
			for _, f := range h.shared {
				if strings.EqualFold(f.Name, name) {
					return f, true
				}
			}
			return FileRef{}, false
		},
		ReceiveDir: func() string { return received },
		Uploaded:   func(string) {},
		Hash: func(path string) (string, error) {
			data, err := os.ReadFile(path)
			if err != nil {
				return "", err
			}
			return shaOf(data), nil
		},
		PCName:       "test-pc",
		PingInterval: time.Hour, // keep pings out of the way
	}
	if tweak != nil {
		tweak(&deps)
	}
	h.m = New(deps)

	pc, ph := net.Pipe()
	t.Cleanup(func() { ph.Close(); pc.Close() })
	go h.m.Serve(pc, bufio.NewReaderSize(pc, 64*1024))

	h.ctrl = ph
	wc := wire.NewConn(ph, bufio.NewReaderSize(ph, 64*1024))
	h.hello = wire.Hello{T: wire.THello, Proto: 2, Role: "control", Device: "Pixel 8", DID: "did-1"}
	if err := wc.WriteJSON(h.hello); err != nil {
		t.Fatal(err)
	}
	typ, _, err := wc.ReadMessage()
	if err != nil {
		t.Fatalf("hello_ok: %v", err)
	}
	if typ != wire.THelloOK {
		t.Fatalf("first reply = %q, want hello_ok", typ)
	}
	waitFor(t, func() bool { return h.m.ControlLinks() == 1 })
	return h
}

// phoneClient is the PC side view of one data connection opened by the phone.
type phoneClient struct {
	conn net.Conn
	wc   *wire.Conn
}

// openData dials a new data connection (the phone always initiates).
func (h *harness) openData(t *testing.T, did string) *phoneClient {
	t.Helper()
	pc, ph := net.Pipe()
	t.Cleanup(func() { ph.Close(); pc.Close() })
	go h.m.Serve(pc, bufio.NewReaderSize(pc, 64*1024))
	wc := wire.NewConn(ph, bufio.NewReaderSize(ph, 64*1024))
	if err := wc.WriteJSON(wire.Hello{T: wire.THello, Proto: 2, Role: "data", DID: did}); err != nil {
		t.Fatal(err)
	}
	typ, _, err := wc.ReadMessage()
	if err != nil || typ != wire.THelloOK {
		t.Fatalf("data hello_ok: %q %v", typ, err)
	}
	return &phoneClient{conn: ph, wc: wc}
}

func waitFor(t *testing.T, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatal("condition not met in time")
}

// controlReader gives the test a persistent buffered reader on the control
// connection (net.Pipe is unbuffered, so one reader must own it).
type controlReader struct {
	conn net.Conn
	wc   *wire.Conn
}

func (h *harness) reader() *controlReader {
	return &controlReader{conn: h.ctrl, wc: wire.NewConn(h.ctrl, bufio.NewReaderSize(h.ctrl, 64*1024))}
}

func (r *controlReader) next(t *testing.T) (string, []byte) {
	t.Helper()
	_ = r.conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	typ, payload, err := r.wc.ReadMessage()
	if err != nil {
		t.Fatalf("control read: %v", err)
	}
	return typ, payload
}

func (r *controlReader) send(t *testing.T, v any) {
	t.Helper()
	if err := r.wc.WriteJSON(v); err != nil {
		t.Fatalf("control write: %v", err)
	}
}

// ---------- tests ----------

func TestHandshakeRejectsBadFirstFrame(t *testing.T) {
	pc, ph := net.Pipe()
	defer ph.Close()
	defer pc.Close()
	m := New(Deps{})
	go m.Serve(pc, bufio.NewReaderSize(pc, 4096))

	wc := wire.NewConn(ph, bufio.NewReaderSize(ph, 4096))
	if err := wc.WriteJSON(wire.Ping{T: wire.TPing}); err != nil {
		t.Fatal(err)
	}
	typ, payload, err := wc.ReadMessage()
	if err != nil {
		t.Fatal(err)
	}
	if typ != wire.TError {
		t.Fatalf("reply = %q, want error", typ)
	}
	var werr wire.Error
	_ = wire.Decode(payload, &werr)
	if werr.Code != wire.CodeBadRequest {
		t.Fatalf("code = %q", werr.Code)
	}
}

func TestHandshakeRejectsWrongProto(t *testing.T) {
	pc, ph := net.Pipe()
	defer ph.Close()
	defer pc.Close()
	m := New(Deps{})
	go m.Serve(pc, bufio.NewReaderSize(pc, 4096))
	wc := wire.NewConn(ph, bufio.NewReaderSize(ph, 4096))
	if err := wc.WriteJSON(wire.Hello{T: wire.THello, Proto: 1, Role: "control", DID: "x"}); err != nil {
		t.Fatal(err)
	}
	typ, payload, err := wc.ReadMessage()
	if err != nil || typ != wire.TError {
		t.Fatalf("reply = %q %v", typ, err)
	}
	var werr wire.Error
	_ = wire.Decode(payload, &werr)
	if werr.Code != wire.CodeUnsupported {
		t.Fatalf("code = %q", werr.Code)
	}
}

func TestFlowB_AskAndPick(t *testing.T) {
	h := newHarness(t, nil)
	r := h.reader()
	body := []byte("PHONE-ECOLOGY-BYTES")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	type askResult struct {
		id    string
		items []Item
		err   error
	}
	ask := make(chan askResult, 1)
	go func() {
		id, items, err := h.m.Ask(ctx, "did-1", "ecology", 50)
		ask <- askResult{id, items, err}
	}()

	typ, payload := r.next(t)
	if typ != wire.TQuery {
		t.Fatalf("PC sent %q, want query", typ)
	}
	var q wire.Query
	_ = wire.Decode(payload, &q)
	if q.Q != "ecology" || q.ID == "" {
		t.Fatalf("query = %+v", q)
	}
	r.send(t, wire.Matches{T: wire.TMatches, ID: q.ID, Items: []wire.Item{
		{N: 1, Name: "Ecology Report.pdf", Size: 999, MTime: 1},
		{N: 2, Name: "ecology-final.docx", Size: int64(len(body)), MTime: 2},
	}})

	res := <-ask
	if res.err != nil {
		t.Fatalf("Ask: %v", res.err)
	}
	if len(res.items) != 2 || res.items[1].Name != "ecology-final.docx" {
		t.Fatalf("items = %+v", res.items)
	}

	// Pick number 2: the phone now opens a data connection and pushes.
	pickDone := make(chan struct {
		ref FileRef
		err error
	}, 1)
	go func() {
		ref, err := h.m.Pick(ctx, "did-1", res.id, 2)
		pickDone <- struct {
			ref FileRef
			err error
		}{ref, err}
	}()

	typ, payload = r.next(t)
	if typ != wire.TPick {
		t.Fatalf("PC sent %q, want pick", typ)
	}
	var pick wire.Pick
	_ = wire.Decode(payload, &pick)
	if pick.N != 2 || pick.ID != res.id {
		t.Fatalf("pick = %+v", pick)
	}

	data := h.openData(t, "did-1")
	if err := data.wc.WriteJSON(wire.Push{T: wire.TPush, ID: res.id, Name: "ecology-final.docx", Size: int64(len(body)), SHA: shaOf(body), Offset: 0}); err != nil {
		t.Fatal(err)
	}
	typ, payload, err := data.wc.ReadMessage()
	if err != nil || typ != wire.TReady {
		t.Fatalf("ready: %q %v", typ, err)
	}
	var ready wire.Ready
	_ = wire.Decode(payload, &ready)
	if ready.Offset != 0 {
		t.Fatalf("ready offset = %d, want 0", ready.Offset)
	}
	if _, err := data.wc.WriteRaw(bytes.NewReader(body), nil); err != nil {
		t.Fatal(err)
	}
	typ, payload, err = data.wc.ReadMessage()
	if err != nil || typ != wire.TDone {
		t.Fatalf("done: %q %v", typ, err)
	}
	var done wire.Done
	_ = wire.Decode(payload, &done)
	if done.Bytes != int64(len(body)) || done.SHA != shaOf(body) {
		t.Fatalf("done = %+v", done)
	}

	got := <-pickDone
	if got.err != nil {
		t.Fatalf("Pick: %v", got.err)
	}
	written, err := os.ReadFile(got.ref.Path)
	if err != nil || !bytes.Equal(written, body) {
		t.Fatalf("received file mismatch: %v", err)
	}
	if filepath.Dir(got.ref.Path) != h.pcDir {
		t.Fatalf("file stored in %s, want %s", got.ref.Path, h.pcDir)
	}

	// Progress events must have been emitted (active … done).
	waitFor(t, func() bool { return len(h.xfer) > 0 })
	h.mu.Lock()
	last := h.xfer[len(h.xfer)-1]
	h.mu.Unlock()
	if last.State != telemetry.TransferDone || last.Dir != telemetry.DirIn {
		t.Fatalf("last transfer = %+v", last)
	}
}

func TestFlowC_PhoneQueriesPC_SingleMatchInline(t *testing.T) {
	h := newHarness(t, nil)
	r := h.reader()
	body := []byte("BUDGET-SHEET")
	path := filepath.Join(t.TempDir(), "budget.xlsx")
	if err := os.WriteFile(path, body, 0o644); err != nil {
		t.Fatal(err)
	}
	h.shared = []FileRef{{Name: "budget.xlsx", Path: path, Size: int64(len(body)), MTime: 10}}

	r.send(t, wire.Query{T: wire.TQuery, ID: "ph-7", Q: "budget"})
	typ, payload := r.next(t)
	if typ != wire.TFile {
		t.Fatalf("PC replied %q, want file", typ)
	}
	var file wire.File
	_ = wire.Decode(payload, &file)
	if file.Name != "budget.xlsx" || file.Size != int64(len(body)) || file.SHA != shaOf(body) {
		t.Fatalf("file frame = %+v", file)
	}
	var buf bytes.Buffer
	n, err := r.wc.ReadRaw(&buf, file.Size, nil)
	if err != nil || n != file.Size {
		t.Fatalf("raw read: %d %v", n, err)
	}
	if !bytes.Equal(buf.Bytes(), body) {
		t.Fatal("bytes mismatch")
	}
	r.send(t, wire.Done{T: wire.TDone, ID: "ph-7", Bytes: file.Size, SHA: shaOf(body)})

	waitFor(t, func() bool {
		reqs := h.m.Requests()
		return len(reqs) == 1 && reqs[0].Outcome == "sent"
	})
	if got := h.m.Requests()[0]; got.Picked != 1 || got.PickedName != "budget.xlsx" {
		t.Fatalf("request log = %+v", got)
	}
}

func TestFlowC_PhoneQueriesPC_MultipleMatchesAndPick(t *testing.T) {
	h := newHarness(t, nil)
	r := h.reader()
	dir := t.TempDir()
	bodyA, bodyB := []byte("AAAA"), []byte("BBBBBB")
	write := func(name string, b []byte) FileRef {
		p := filepath.Join(dir, name)
		if err := os.WriteFile(p, b, 0o644); err != nil {
			t.Fatal(err)
		}
		return FileRef{Name: name, Path: p, Size: int64(len(b)), MTime: 1}
	}
	h.shared = []FileRef{write("report-a.pdf", bodyA), write("report-b.pdf", bodyB)}

	r.send(t, wire.Query{T: wire.TQuery, ID: "ph-9", Q: "report"})
	typ, payload := r.next(t)
	if typ != wire.TMatches {
		t.Fatalf("PC replied %q, want matches", typ)
	}
	var matches wire.Matches
	_ = wire.Decode(payload, &matches)
	if len(matches.Items) != 2 || matches.Items[0].N != 1 || matches.TTL != 120 {
		t.Fatalf("matches = %+v", matches)
	}

	r.send(t, wire.Pick{T: wire.TPick, ID: "ph-9", N: 2})
	typ, payload = r.next(t)
	if typ != wire.TFile {
		t.Fatalf("after pick PC replied %q, want file", typ)
	}
	var file wire.File
	_ = wire.Decode(payload, &file)
	if file.Name != "report-b.pdf" {
		t.Fatalf("picked = %q", file.Name)
	}
	var buf bytes.Buffer
	if _, err := r.wc.ReadRaw(&buf, file.Size, nil); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(buf.Bytes(), bodyB) {
		t.Fatal("picked bytes mismatch")
	}
	r.send(t, wire.Done{T: wire.TDone, ID: "ph-9", Bytes: file.Size, SHA: shaOf(bodyB)})

	waitFor(t, func() bool {
		reqs := h.m.Requests()
		return len(reqs) == 1 && reqs[0].Picked == 2
	})
}

func TestFlowC_NoMatches(t *testing.T) {
	h := newHarness(t, nil)
	r := h.reader()
	r.send(t, wire.Query{T: wire.TQuery, ID: "ph-1", Q: "nothing"})
	typ, payload := r.next(t)
	if typ != wire.TError {
		t.Fatalf("reply = %q, want error", typ)
	}
	var werr wire.Error
	_ = wire.Decode(payload, &werr)
	if werr.ID != "ph-1" || werr.Code != wire.CodeNotFound {
		t.Fatalf("error = %+v", werr)
	}
	waitFor(t, func() bool {
		reqs := h.m.Requests()
		return len(reqs) == 1 && reqs[0].Outcome == "no-match"
	})
}

func TestFlowC_AbortAndExpiredPick(t *testing.T) {
	h := newHarness(t, nil)
	r := h.reader()
	dir := t.TempDir()
	for _, name := range []string{"x-1.txt", "x-2.txt"} {
		p := filepath.Join(dir, name)
		_ = os.WriteFile(p, []byte(name), 0o644)
		h.shared = append(h.shared, FileRef{Name: name, Path: p, Size: int64(len(name))})
	}

	r.send(t, wire.Query{T: wire.TQuery, ID: "ph-2", Q: "x-"})
	if typ, _ := r.next(t); typ != wire.TMatches {
		t.Fatalf("want matches, got %q", typ)
	}
	r.send(t, wire.Abort{T: wire.TAbort, ID: "ph-2"})
	typ, payload := r.next(t)
	if typ != wire.TOk {
		t.Fatalf("abort reply = %q", typ)
	}
	var ok wire.Ok
	_ = wire.Decode(payload, &ok)
	if ok.ID != "ph-2" {
		t.Fatalf("ok id = %q", ok.ID)
	}

	// Picking after an abort (or with an unknown id) is "expired".
	r.send(t, wire.Pick{T: wire.TPick, ID: "ph-2", N: 1})
	typ, payload = r.next(t)
	if typ != wire.TError {
		t.Fatalf("late pick reply = %q", typ)
	}
	var werr wire.Error
	_ = wire.Decode(payload, &werr)
	if werr.Code != wire.CodeExpired {
		t.Fatalf("code = %q, want expired", werr.Code)
	}
}

func TestFlowD_OfferAcceptPush(t *testing.T) {
	h := newHarness(t, nil)
	r := h.reader()
	body := []byte("PROPOSAL-BYTES")
	path := filepath.Join(t.TempDir(), "proposal-v3.docx")
	if err := os.WriteFile(path, body, 0o644); err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	errCh := make(chan error, 1)
	go func() {
		errCh <- h.m.Offer(ctx, "did-1", FileRef{Name: "proposal-v3.docx", Path: path, Size: int64(len(body))})
	}()

	typ, payload := r.next(t)
	if typ != wire.TOffer {
		t.Fatalf("PC sent %q, want offer", typ)
	}
	var offer wire.Offer
	_ = wire.Decode(payload, &offer)
	if offer.Name != "proposal-v3.docx" || offer.Size != int64(len(body)) || offer.SHA != shaOf(body) {
		t.Fatalf("offer = %+v", offer)
	}
	r.send(t, wire.Accept{T: wire.TAccept, ID: offer.ID})

	data := h.openData(t, "did-1")
	if err := data.wc.WriteJSON(wire.Pull{T: wire.TPull, ID: offer.ID, Offset: 0}); err != nil {
		t.Fatal(err)
	}
	typ, payload, err := data.wc.ReadMessage()
	if err != nil || typ != wire.TFile {
		t.Fatalf("pull answer = %q %v", typ, err)
	}
	var file wire.File
	_ = wire.Decode(payload, &file)
	var buf bytes.Buffer
	if _, err := data.wc.ReadRaw(&buf, file.Size, nil); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(buf.Bytes(), body) {
		t.Fatal("streamed bytes mismatch")
	}
	if err := data.wc.WriteJSON(wire.Done{T: wire.TDone, ID: offer.ID, Bytes: file.Size, SHA: shaOf(body)}); err != nil {
		t.Fatal(err)
	}
	if err := <-errCh; err != nil {
		t.Fatalf("Offer: %v", err)
	}
}

func TestFlowD_Reject(t *testing.T) {
	h := newHarness(t, nil)
	r := h.reader()
	path := filepath.Join(t.TempDir(), "denied.bin")
	_ = os.WriteFile(path, []byte("x"), 0o644)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	errCh := make(chan error, 1)
	go func() { errCh <- h.m.Offer(ctx, "did-1", FileRef{Name: "denied.bin", Path: path, Size: 1}) }()

	typ, payload := r.next(t)
	if typ != wire.TOffer {
		t.Fatalf("want offer, got %q", typ)
	}
	var offer wire.Offer
	_ = wire.Decode(payload, &offer)
	r.send(t, wire.Reject{T: wire.TReject, ID: offer.ID, Code: wire.CodeDenied})

	err := <-errCh
	if err == nil || !strings.Contains(err.Error(), wire.CodeDenied) {
		t.Fatalf("Offer error = %v, want denied", err)
	}
}

func TestFlowE_GetSpecificFile(t *testing.T) {
	h := newHarness(t, nil)
	r := h.reader()
	body := []byte("TRIP-PLAN")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	type res struct {
		ref FileRef
		err error
	}
	done := make(chan res, 1)
	go func() {
		ref, err := h.m.Get(ctx, "did-1", "trip-plan.pdf")
		done <- res{ref, err}
	}()

	typ, payload := r.next(t)
	if typ != wire.TGet {
		t.Fatalf("PC sent %q, want get", typ)
	}
	var get wire.Get
	_ = wire.Decode(payload, &get)
	if get.Path != "trip-plan.pdf" {
		t.Fatalf("get = %+v", get)
	}

	data := h.openData(t, "did-1")
	_ = data.wc.WriteJSON(wire.Push{T: wire.TPush, ID: get.ID, Name: "trip-plan.pdf", Size: int64(len(body)), SHA: shaOf(body)})
	if typ, _, err := data.wc.ReadMessage(); err != nil || typ != wire.TReady {
		t.Fatalf("ready: %q %v", typ, err)
	}
	if _, err := data.wc.WriteRaw(bytes.NewReader(body), nil); err != nil {
		t.Fatal(err)
	}
	if typ, _, err := data.wc.ReadMessage(); err != nil || typ != wire.TDone {
		t.Fatalf("done: %q %v", typ, err)
	}
	got := <-done
	if got.err != nil {
		t.Fatalf("Get: %v", got.err)
	}
	if data, err := os.ReadFile(got.ref.Path); err != nil || !bytes.Equal(data, body) {
		t.Fatal("pull destination mismatch")
	}
}

func TestFlowF_MultiPartManifest(t *testing.T) {
	h := newHarness(t, nil)
	r := h.reader()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	type res struct {
		items []wire.ManifestItem
		err   error
	}
	done := make(chan res, 1)
	go func() {
		items, err := h.m.Manifest(ctx, "did-1", "sync")
		done <- res{items, err}
	}()

	typ, payload := r.next(t)
	if typ != wire.TManifestReq {
		t.Fatalf("PC sent %q, want manifest_req", typ)
	}
	var req wire.ManifestReq
	_ = wire.Decode(payload, &req)
	if req.Scope != "sync" {
		t.Fatalf("scope = %q", req.Scope)
	}
	r.send(t, wire.Manifest{T: wire.TManifest, ID: req.ID, Part: 1, Last: false, Items: []wire.ManifestItem{
		{P: "b.pdf", Size: 2, MTime: 2},
	}})
	r.send(t, wire.Manifest{T: wire.TManifest, ID: req.ID, Part: 2, Last: true, Items: []wire.ManifestItem{
		{P: "a.pdf", Size: 1, MTime: 1, SHA: "abc"},
	}})

	got := <-done
	if got.err != nil {
		t.Fatalf("Manifest: %v", got.err)
	}
	if len(got.items) != 2 || got.items[0].P != "a.pdf" {
		t.Fatalf("manifest items = %+v", got.items)
	}
}

func TestResumeFromPartial(t *testing.T) {
	h := newHarness(t, nil)
	full := bytes.Repeat([]byte("0123456789"), 200) // 2000 bytes
	sum := shaOf(full)
	half := len(full) / 2

	// The phone starts a push that dies halfway…
	data := h.openData(t, "did-1")
	_ = data.wc.WriteJSON(wire.Push{T: wire.TPush, ID: "pc-1", Name: "big.bin", Size: int64(len(full)), SHA: sum, Offset: 0})
	if typ, _, err := data.wc.ReadMessage(); err != nil || typ != wire.TReady {
		t.Fatalf("ready: %q %v", typ, err)
	}
	if _, err := data.wc.WriteRaw(bytes.NewReader(full[:half]), nil); err != nil {
		t.Fatal(err)
	}
	data.conn.Close()
	// The partial is created empty when `ready` is written, so wait for the
	// half that actually landed, not just for the file to exist.
	waitFor(t, func() bool {
		st, err := os.Stat(filepath.Join(h.pcDir, ".partial", sum+".part"))
		return err == nil && st.Size() == int64(half)
	})

	// …and comes back with the same hash: the PC reports what it already has.
	data2 := h.openData(t, "did-1")
	_ = data2.wc.WriteJSON(wire.Push{T: wire.TPush, ID: "pc-2", Name: "big.bin", Size: int64(len(full)), SHA: sum, Offset: 0})
	typ, payload, err := data2.wc.ReadMessage()
	if err != nil || typ != wire.TReady {
		t.Fatalf("second ready: %q %v", typ, err)
	}
	var ready wire.Ready
	if derr := wire.Decode(payload, &ready); derr != nil {
		t.Fatalf("decode ready %q: %v", payload, derr)
	}
	if ready.Offset != int64(half) {
		t.Fatalf("resume offset = %d, want %d", ready.Offset, half)
	}
	if _, err := data2.wc.WriteRaw(bytes.NewReader(full[ready.Offset:]), nil); err != nil {
		t.Fatal(err)
	}
	if typ, _, err := data2.wc.ReadMessage(); err != nil || typ != wire.TDone {
		t.Fatalf("done: %q %v", typ, err)
	}
	written, err := os.ReadFile(filepath.Join(h.pcDir, "big.bin"))
	if err != nil || !bytes.Equal(written, full) {
		t.Fatalf("resumed file mismatch (%d bytes, err %v)", len(written), err)
	}
	// The partial is gone once the file is complete.
	if _, err := os.Stat(filepath.Join(h.pcDir, ".partial", sum+".part")); err == nil {
		t.Fatal("partial left behind after a completed resume")
	}
}

func TestHashMismatchDeletesPartial(t *testing.T) {
	h := newHarness(t, nil)
	data := h.openData(t, "did-1")
	body := []byte("SOME-BYTES")
	wrong := shaOf([]byte("something else"))
	_ = data.wc.WriteJSON(wire.Push{T: wire.TPush, ID: "pc-3", Name: "evil.bin", Size: int64(len(body)), SHA: wrong})
	if typ, _, err := data.wc.ReadMessage(); err != nil || typ != wire.TReady {
		t.Fatalf("ready: %q %v", typ, err)
	}
	if _, err := data.wc.WriteRaw(bytes.NewReader(body), nil); err != nil {
		t.Fatal(err)
	}
	typ, payload, err := data.wc.ReadMessage()
	if err != nil || typ != wire.TError {
		t.Fatalf("reply = %q %v, want error", typ, err)
	}
	var werr wire.Error
	_ = wire.Decode(payload, &werr)
	if werr.Code != wire.CodeHashMismatch {
		t.Fatalf("code = %q", werr.Code)
	}
	if _, err := os.Stat(filepath.Join(h.pcDir, "evil.bin")); err == nil {
		t.Fatal("file with a bad hash was stored")
	}
}

func TestPushNameSanitising(t *testing.T) {
	h := newHarness(t, nil)
	data := h.openData(t, "did-1")
	_ = data.wc.WriteJSON(wire.Push{T: wire.TPush, ID: "pc-4", Name: "../escape.bin", Size: 3})
	typ, payload, err := data.wc.ReadMessage()
	if err != nil || typ != wire.TError {
		t.Fatalf("reply = %q %v", typ, err)
	}
	var werr wire.Error
	_ = wire.Decode(payload, &werr)
	if werr.Code != wire.CodeBadRequest {
		t.Fatalf("code = %q", werr.Code)
	}
	if _, err := os.Stat(filepath.Join(filepath.Dir(h.pcDir), "escape.bin")); err == nil {
		t.Fatal("traversal escaped the received folder")
	}
}

func TestCollisionKeepsBothFiles(t *testing.T) {
	h := newHarness(t, nil)
	existing := filepath.Join(h.pcDir, "notes.txt")
	if err := os.WriteFile(existing, []byte("old"), 0o644); err != nil {
		t.Fatal(err)
	}
	body := []byte("new content")
	data := h.openData(t, "did-1")
	_ = data.wc.WriteJSON(wire.Push{T: wire.TPush, ID: "pc-5", Name: "notes.txt", Size: int64(len(body)), SHA: shaOf(body)})
	if typ, _, err := data.wc.ReadMessage(); err != nil || typ != wire.TReady {
		t.Fatalf("ready: %q %v", typ, err)
	}
	_, _ = data.wc.WriteRaw(bytes.NewReader(body), nil)
	if typ, _, err := data.wc.ReadMessage(); err != nil || typ != wire.TDone {
		t.Fatalf("done: %q %v", typ, err)
	}
	// The original file is untouched and the new one is "notes (1).txt".
	if old, _ := os.ReadFile(existing); string(old) != "old" {
		t.Fatalf("existing file was overwritten: %q", old)
	}
	if _, err := os.Stat(filepath.Join(h.pcDir, "notes (1).txt")); err != nil {
		t.Fatal("collision produced no second copy")
	}
}

func TestControlReplacement(t *testing.T) {
	h := newHarness(t, nil)
	if got := h.m.ControlLinks(); got != 1 {
		t.Fatalf("links = %d", got)
	}
	pc2, ph2 := net.Pipe()
	t.Cleanup(func() { ph2.Close(); pc2.Close() })
	go h.m.Serve(pc2, bufio.NewReaderSize(pc2, 4096))
	wc2 := wire.NewConn(ph2, bufio.NewReaderSize(ph2, 4096))
	_ = wc2.WriteJSON(wire.Hello{T: wire.THello, Proto: 2, Role: "control", Device: "Pixel 8", DID: "did-1"})
	if typ, _, err := wc2.ReadMessage(); err != nil || typ != wire.THelloOK {
		t.Fatalf("second handshake: %q %v", typ, err)
	}
	waitFor(t, func() bool { return h.m.ControlLinks() == 1 })

	// The first link must have been closed by the PC.
	_ = h.ctrl.SetReadDeadline(time.Now().Add(2 * time.Second))
	buf := make([]byte, 1)
	if _, err := h.ctrl.Read(buf); err == nil {
		t.Fatal("old control link still open")
	}
}

func TestPingTimeoutDropsControlLink(t *testing.T) {
	h := newHarness(t, func(d *Deps) {
		d.PingInterval = 20 * time.Millisecond
		d.PongTimeout = 60 * time.Millisecond
	})
	waitFor(t, func() bool { return h.m.ControlLinks() == 0 })
}

func TestPongKeepsControlLinkAlive(t *testing.T) {
	h := newHarness(t, func(d *Deps) {
		d.PingInterval = 20 * time.Millisecond
		d.PongTimeout = 200 * time.Millisecond
	})
	r := h.reader()
	deadline := time.Now().Add(300 * time.Millisecond)
	for time.Now().Before(deadline) {
		_ = h.ctrl.SetReadDeadline(time.Now().Add(100 * time.Millisecond))
		typ, _, err := r.wc.ReadMessage()
		if err != nil {
			break
		}
		if typ == wire.TPing {
			r.send(t, wire.Pong{T: wire.TPong})
		}
	}
	if h.m.ControlLinks() != 1 {
		t.Fatal("a pong-answering phone must keep its control link")
	}
}

func TestNoDeviceError(t *testing.T) {
	m := New(Deps{})
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if _, _, err := m.Ask(ctx, "", "q", 10); err != ErrNoDevice {
		t.Fatalf("err = %v, want ErrNoDevice", err)
	}
}

func TestDataConnectionLimit(t *testing.T) {
	h := newHarness(t, nil)
	// Consume the 4 slots with connections that never send a transfer.
	for i := 0; i < wire.MaxDataConns; i++ {
		_ = h.openData(t, "did-1")
	}
	pc, ph := net.Pipe()
	t.Cleanup(func() { ph.Close(); pc.Close() })
	go h.m.Serve(pc, bufio.NewReaderSize(pc, 4096))
	wc := wire.NewConn(ph, bufio.NewReaderSize(ph, 4096))
	_ = wc.WriteJSON(wire.Hello{T: wire.THello, Proto: 2, Role: "data", DID: "did-1"})
	typ, payload, err := wc.ReadMessage()
	if err != nil || typ != wire.TError {
		t.Fatalf("fifth data connection = %q %v, want busy", typ, err)
	}
	var werr wire.Error
	_ = wire.Decode(payload, &werr)
	if werr.Code != wire.CodeBusy {
		t.Fatalf("code = %q, want busy", werr.Code)
	}
}

func TestOversizeUploadRejected(t *testing.T) {
	h := newHarness(t, nil)
	data := h.openData(t, "did-1")
	_ = data.wc.WriteJSON(wire.Push{T: wire.TPush, ID: "x", Name: "huge.bin", Size: wire.MaxUploadBytes + 1})
	typ, payload, err := data.wc.ReadMessage()
	if err != nil || typ != wire.TError {
		t.Fatalf("reply = %q %v", typ, err)
	}
	var werr wire.Error
	_ = wire.Decode(payload, &werr)
	if werr.Code != wire.CodeTooLarge {
		t.Fatalf("code = %q", werr.Code)
	}
}
