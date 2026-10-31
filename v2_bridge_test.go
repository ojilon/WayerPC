package main

// End-to-end bridge test: a real TCP server, the real session manager, and a
// fake phone driving flow B (search the PC, pick a match, push the file).
// This is the seam the unit tests cannot see — Deps wiring in v2.go.

import (
	"bufio"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"wayerpc/internal/catalog"
	"wayerpc/internal/config"
	"wayerpc/internal/index"
	"wayerpc/internal/server"
	"wayerpc/internal/storage"
	"wayerpc/internal/wire"
)

func sha256Hex(b []byte) string {
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

// newBridgedApp wires an App to a temp data root without Wails.
func newBridgedApp(t *testing.T) (*App, *server.Server, string) {
	t.Helper()
	dir := t.TempDir()
	store, err := storage.Init(dir)
	if err != nil {
		t.Fatal(err)
	}
	cat, err := catalog.Open(store.DbPath(), "test")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { cat.Close() })

	app := NewApp()
	app.cfg = config.Defaults()
	app.cfg.DataRoot = dir
	app.store = store
	app.cat = cat
	app.idx = index.New(cat.SQL())

	srv := server.New("127.0.0.1", 0, store, cat, app.logf)
	app.srv = srv
	srv.SetObserver(app)
	srv.SetV2(app.sessionManager())

	go func() { _ = srv.Start() }()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if srv.Snapshot().Running {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if !srv.Snapshot().Running {
		t.Fatal("server never started")
	}
	t.Cleanup(func() { _ = srv.Stop() })
	return app, srv, dir
}

type fakePhone struct {
	conn net.Conn
	wc   *wire.Conn
}

func dialPhone(t *testing.T, srv *server.Server) *fakePhone {
	t.Helper()
	c, err := net.Dial("tcp", srv.ListenAddr())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { c.Close() })
	p := &fakePhone{conn: c, wc: wire.NewConn(c, bufio.NewReaderSize(c, 64*1024))}
	if err := p.wc.WriteJSON(wire.Hello{T: wire.THello, Proto: 2, Role: "control", Device: "Pixel 8", DID: "did-e2e"}); err != nil {
		t.Fatal(err)
	}
	typ, _, err := p.wc.ReadMessage()
	if err != nil || typ != wire.THelloOK {
		t.Fatalf("hello_ok: %q %v", typ, err)
	}
	return p
}

func TestEndToEndAskAndPick(t *testing.T) {
	app, srv, dir := newBridgedApp(t)
	body := []byte("REPORT-BYTES-FROM-THE-PHONE")
	if err := os.WriteFile(filepath.Join(dir, "shared", "Q3-report.pdf"), []byte("pc copy"), 0o644); err != nil {
		t.Fatal(err)
	}

	phone := dialPhone(t, srv)
	// The PC must see the link before AskPhone will talk to it.
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) && app.GetStatus().ControlLinks == 0 {
		time.Sleep(10 * time.Millisecond)
	}
	if got := app.GetStatus().Device; got != "Pixel 8" {
		t.Fatalf("status.device = %q", got)
	}

	type askOut struct{ res AskPhoneResult }
	askCh := make(chan askOut, 1)
	go func() { askCh <- askOut{app.AskPhone("report")} }()

	// Phone side of the ask flow.
	typ, payload, err := phone.wc.ReadMessage()
	if err != nil || typ != wire.TQuery {
		t.Fatalf("PC sent %q %v, want query", typ, err)
	}
	var query wire.Query
	_ = wire.Decode(payload, &query)
	if !strings.Contains(strings.ToLower(query.Q), "report") {
		t.Fatalf("query = %+v", query)
	}
	_ = phone.wc.WriteJSON(wire.Matches{T: wire.TMatches, ID: query.ID, Items: []wire.Item{
		{N: 1, Name: "Q3-report.pdf", Size: int64(len(body)), MTime: 3},
	}})

	res := (<-askCh).res
	if res.State != "matches" || len(res.Matches) != 1 || res.ID != query.ID {
		t.Fatalf("ask result = %+v", res)
	}

	// Pick: the phone opens a data connection and pushes the bytes.
	pickCh := make(chan struct {
		path string
		err  error
	}, 1)
	go func() {
		p, perr := app.PickAskMatch(res.ID, 1)
		pickCh <- struct {
			path string
			err  error
		}{p, perr}
	}()

	typ, payload, err = phone.wc.ReadMessage()
	if err != nil || typ != wire.TPick {
		t.Fatalf("PC sent %q %v, want pick", typ, err)
	}
	var pick wire.Pick
	_ = wire.Decode(payload, &pick)
	if pick.N != 1 {
		t.Fatalf("pick = %+v", pick)
	}

	data, err := net.Dial("tcp", srv.ListenAddr())
	if err != nil {
		t.Fatal(err)
	}
	defer data.Close()
	dwc := wire.NewConn(data, bufio.NewReaderSize(data, 64*1024))
	_ = dwc.WriteJSON(wire.Hello{T: wire.THello, Proto: 2, Role: "data", DID: "did-e2e"})
	if typ, _, err := dwc.ReadMessage(); err != nil || typ != wire.THelloOK {
		t.Fatalf("data hello_ok: %q %v", typ, err)
	}
	_ = dwc.WriteJSON(wire.Push{T: wire.TPush, ID: res.ID, Name: "Q3-report.pdf", Size: int64(len(body)), SHA: sha256Hex(body)})
	if typ, _, err := dwc.ReadMessage(); err != nil || typ != wire.TReady {
		t.Fatalf("ready: %q %v", typ, err)
	}
	if _, err := dwc.WriteRaw(bytes.NewReader(body), nil); err != nil {
		t.Fatal(err)
	}
	if typ, _, err := dwc.ReadMessage(); err != nil || typ != wire.TDone {
		t.Fatalf("done: %q %v", typ, err)
	}

	got := <-pickCh
	if got.err != nil {
		t.Fatalf("PickAskMatch: %v", got.err)
	}
	stored, err := os.ReadFile(got.path)
	if err != nil || !bytes.Equal(stored, body) {
		t.Fatalf("stored file mismatch: %v", err)
	}
	if filepath.Dir(got.path) != filepath.Join(dir, "received") {
		t.Fatalf("saved outside received/: %s", got.path)
	}
	// The bridge must index + catalog the new file (Files view depends on it).
	if entries, err := app.cat.List(); err == nil {
		found := false
		for _, e := range entries {
			if strings.EqualFold(e.Name, "Q3-report.pdf") {
				found = true
			}
		}
		if !found {
			t.Fatalf("uploaded file missing from the catalog: %+v", entries)
		}
	}
	// Flow B is PC-initiated, so the *phone-initiated* request log stays empty;
	// the link state is what must have updated.
	if st := app.GetV2State(); st.ControlLinks != 1 || st.ConnectedName != "Pixel 8" {
		t.Fatalf("v2 state after the flow = %+v", st)
	}
	if reqs := app.GetRequestLog(); len(reqs) != 0 {
		t.Fatalf("flow B should not log a phone request: %+v", reqs)
	}
	// The device registry remembers the phone for the next session.
	devices := app.GetDevices()
	if len(devices) != 1 || devices[0].DID != "did-e2e" {
		t.Fatalf("devices = %+v", devices)
	}
}

func TestEndToEndSendToPhoneRequiresLink(t *testing.T) {
	app, _, dir := newBridgedApp(t)
	path := filepath.Join(dir, "shared", "handout.pdf")
	if err := os.WriteFile(path, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	err := app.SendToPhone(path)
	if err == nil || !strings.Contains(err.Error(), "No phone connected") {
		t.Fatalf("SendToPhone without a link = %v", err)
	}
	if res := app.AskPhone("handout"); res.State != "error" || res.Error == "" {
		t.Fatalf("AskPhone without a link = %+v", res)
	}
}

func TestEndToEndV2Toggle(t *testing.T) {
	app, srv, _ := newBridgedApp(t)
	if st := app.GetV2State(); !st.Enabled || st.Protocol != 2 {
		t.Fatalf("v2 state = %+v", st)
	}
	app.SetProtocolV2(false)
	if app.GetV2State().Enabled {
		t.Fatal("SetProtocolV2(false) did not stick")
	}
	// With v2 off, an '@' connection is closed rather than served.
	c, err := net.Dial("tcp", srv.ListenAddr())
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	if _, err := c.Write([]byte("@2\n{}")); err != nil {
		t.Fatal(err)
	}
	_ = c.SetReadDeadline(time.Now().Add(3 * time.Second))
	buf := make([]byte, 8)
	if _, err := c.Read(buf); err == nil {
		t.Fatal("v2 connection stayed open while disabled")
	}
	app.SetProtocolV2(true)
}

func waitForCond(t *testing.T, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("condition not met in time")
}
