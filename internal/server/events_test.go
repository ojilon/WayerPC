package server

import (
	"bufio"
	"bytes"
	"net"
	"sync"
	"testing"
	"time"
)

// recorder captures the observer pushes for assertions.
type recorder struct {
	mu        sync.Mutex
	transfers []Transfer
	activity  []Activity
}

func (r *recorder) Transfer(t Transfer) {
	r.mu.Lock()
	r.transfers = append(r.transfers, t)
	r.mu.Unlock()
}

func (r *recorder) Activity(a Activity) {
	r.mu.Lock()
	r.activity = append(r.activity, a)
	r.mu.Unlock()
}

func (r *recorder) transferStates() []string {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]string, 0, len(r.transfers))
	for _, t := range r.transfers {
		out = append(out, t.State)
	}
	return out
}

func TestUploadEmitsTransferProgressAndActivity(t *testing.T) {
	srv, _, _ := testServer(t)
	rec := &recorder{}
	srv.SetObserver(rec)

	body := bytes.Repeat([]byte("x"), 4096)
	out, status, detail, fatal := serveRaw(t, srv, func(client net.Conn) {
		if _, err := client.Write([]byte("/upload 4096 prog.bin\n")); err != nil {
			return
		}
		buf := make([]byte, 5) // bare READY, no newline (Android behaviour)
		_, _ = client.Read(buf)
		_, _ = client.Write(body)
		_, _ = client.Read(make([]byte, 64)) // DONE
	})
	if fatal || status != StatusUploadOK {
		t.Fatalf("status=%d detail=%q fatal=%v out=%q", status, detail, fatal, out)
	}

	states := rec.transferStates()
	if len(states) < 2 || states[0] != TransferActive || states[len(states)-1] != TransferDone {
		t.Fatalf("want active..done, got %v", states)
	}
	rec.mu.Lock()
	first := rec.transfers[0]
	last := rec.transfers[len(rec.transfers)-1]
	rec.mu.Unlock()
	if first.ID == "" || first.Dir != DirIn || first.Name != "prog.bin" || first.Total != 4096 {
		t.Fatalf("bad initial transfer frame: %+v", first)
	}
	if last.Bytes != 4096 {
		t.Fatalf("final bytes = %d, want 4096", last.Bytes)
	}

	var sawReceived bool
	for _, a := range rec.activity {
		if a.Kind == KindFileReceived && a.Status == "Complete" {
			sawReceived = true
		}
	}
	if !sawReceived {
		t.Fatalf("missing file_received activity: %+v", rec.activity)
	}
}

func TestAskEmitsFileSentActivity(t *testing.T) {
	srv, _, _ := testServer(t)
	rec := &recorder{}
	srv.SetObserver(rec)
	p := seed(t, t.TempDir(), "sent.pdf", "0123456789")
	if _, err := srv.cat.AddPath(p); err != nil {
		t.Fatal(err)
	}
	if _, status, _, _ := serveRaw(t, srv, func(client net.Conn) {
		_, _ = client.Write([]byte("/ask sent.pdf\n"))
	}); status != StatusAskOK {
		t.Fatalf("status=%d", status)
	}
	rec.mu.Lock()
	defer rec.mu.Unlock()
	var sawSent bool
	for _, a := range rec.activity {
		if a.Kind == KindFileSent {
			sawSent = true
		}
	}
	if !sawSent {
		t.Fatalf("missing file_sent activity: %+v", rec.activity)
	}
}

func TestCancelStopsUpload(t *testing.T) {
	srv, store, _ := testServer(t)
	rec := &recorder{}
	srv.SetObserver(rec)

	client, server := net.Pipe()
	defer client.Close()
	defer server.Close()
	go srv.serveConn(server)

	go func() {
		_, _ = client.Write([]byte("/upload 8192 cancel.bin\n"))
		buf := make([]byte, 5)
		_, _ = client.Read(buf) // READY
		_, _ = client.Write(bytes.Repeat([]byte("y"), 1024))
	}()

	// Wait for the transfer to be registered, then cancel it.
	var id string
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		ids := srv.ActiveTransfers()
		if len(ids) > 0 {
			id = ids[0]
			break
		}
		time.Sleep(5 * time.Millisecond)
	}
	if id == "" {
		t.Fatal("transfer was never registered for cancellation")
	}
	if !srv.Cancel(id) {
		t.Fatal("Cancel reported no such transfer")
	}

	// The server closes the connection; the reader sees EOF quickly.
	_ = client.SetReadDeadline(time.Now().Add(3 * time.Second))
	_, _ = bufio.NewReader(client).ReadByte()

	deadline = time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		states := rec.transferStates()
		if len(states) > 0 && states[len(states)-1] == TransferCancelled {
			break
		}
		time.Sleep(5 * time.Millisecond)
	}
	states := rec.transferStates()
	if len(states) == 0 || states[len(states)-1] != TransferCancelled {
		t.Fatalf("want a cancelled transfer, got %v", states)
	}
	if _, err := store.ListFiles(store.ReceivedDir()); err != nil {
		t.Fatalf("received dir unreadable: %v", err)
	}
	files, _ := store.ListFiles(store.ReceivedDir())
	for _, f := range files {
		if f.Name == "cancel.bin" || f.Name == "cancel.bin.part" {
			t.Fatalf("cancelled upload left %q behind", f.Name)
		}
	}
}

func TestCancelUnknownTransfer(t *testing.T) {
	srv, _, _ := testServer(t)
	if srv.Cancel("does-not-exist") {
		t.Fatal("Cancel must report false for unknown ids")
	}
}

func TestHumanBytes(t *testing.T) {
	cases := map[int64]string{
		512:         "512 B",
		1024:        "1 KB",
		1536:        "1.5 KB",
		1024 * 1024: "1 MB",
		1843200:     "1.75 MB",
	}
	for in, want := range cases {
		if got := humanBytes(in); got != want {
			t.Errorf("humanBytes(%d) = %q, want %q", in, got, want)
		}
	}
}
