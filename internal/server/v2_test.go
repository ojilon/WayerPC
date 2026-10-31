package server

// Protocol-v2 opt-in: a connection whose first byte is '@' is handed to the
// V2Handler; anything else takes the v1 command path unchanged. These tests
// only exercise the dispatch seam (the session package owns the wire itself).

import (
	"bufio"
	"bytes"
	"encoding/json"
	"strconv"
	"sync"

	"net"
	"strings"
	"testing"
	"time"
	"wayerpc/internal/catalog"
	"wayerpc/internal/storage"

	"wayerpc/internal/wire"
)

type fakeV2 struct {
	called chan string
	reply  string
}

func (f *fakeV2) ServeV2(conn net.Conn, br *bufio.Reader) {
	first, _ := br.Peek(1)
	f.called <- string(first)
	if f.reply != "" {
		_, _ = conn.Write([]byte(f.reply))
	}
}

// dialV2 runs one connection through srv.serveConn over a real socket.
func dialV2(t *testing.T, srv *Server) (net.Conn, func()) {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	go func() {
		c, aerr := ln.Accept()
		if aerr != nil {
			return
		}
		srv.serveConn(c)
	}()
	client, err := net.Dial("tcp", ln.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	return client, func() { client.Close(); ln.Close() }
}

func TestV2DispatchByFirstByte(t *testing.T) {
	srv, _, _ := testServer(t)
	fake := &fakeV2{called: make(chan string, 1)}
	srv.SetV2(fake)

	client, done := dialV2(t, srv)
	defer done()

	// Build the frame by hand: "@" + byte length + "\n" + JSON.
	payload, err := json.Marshal(wire.Hello{T: wire.THello, Proto: 2, Role: "control", DID: "d1"})
	if err != nil {
		t.Fatal(err)
	}
	frame := []byte{wire.Marker}
	frame = append(frame, []byte(strconv.Itoa(len(payload))+"\n")...)
	frame = append(frame, payload...)
	if _, err := client.Write(frame); err != nil {
		t.Fatal(err)
	}
	select {
	case first := <-fake.called:
		if first != string([]byte{wire.Marker}) {
			t.Fatalf("handler saw %q, want '@'", first)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("V2Handler was never called")
	}
}

type logSink struct {
	mu    sync.Mutex
	lines []string
}

func (l *logSink) add(line string) {
	l.mu.Lock()
	l.lines = append(l.lines, line)
	l.mu.Unlock()
}

func (l *logSink) joined() string {
	l.mu.Lock()
	defer l.mu.Unlock()
	return strings.Join(l.lines, "\n")
}

func TestV2DisabledClosesConnection(t *testing.T) {
	root := t.TempDir()
	store, err := storage.Init(root)
	if err != nil {
		t.Fatal(err)
	}
	cat, err := catalog.Open(store.DbPath(), "test")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { cat.Close() })
	sink := &logSink{}
	srv := New("127.0.0.1", 0, store, cat, sink.add)
	srv.cmdSettle = 50 * time.Millisecond
	srv.SetV2(nil)
	client, done := dialV2(t, srv)
	defer done()
	if _, err := client.Write([]byte("@5\nhello")); err != nil {
		t.Fatal(err)
	}
	_ = client.SetReadDeadline(time.Now().Add(3 * time.Second))
	buf := make([]byte, 16)
	if _, err := client.Read(buf); err == nil {
		t.Fatal("connection stayed open with v2 disabled")
	}
	// The log slice is appended from the connection goroutine; give it a
	// moment to land before asserting on it.
	var joined string
	for i := 0; i < 40; i++ {
		joined = sink.joined()
		if strings.Contains(joined, "protocol v2") {
			break
		}
		time.Sleep(25 * time.Millisecond)
	}
	if !strings.Contains(joined, "protocol v2") {
		t.Fatalf("no log line explaining the refusal: %q", joined)
	}
}

func TestV1StillServedWhenV2Installed(t *testing.T) {
	srv, store, _ := testServer(t)
	srv.SetV2(&fakeV2{called: make(chan string, 1)})
	path := seed(t, store.SharedDir(), "notes.txt", "hello")

	client, done := dialV2(t, srv)
	defer done()
	if _, err := client.Write([]byte("/ask notes.txt\n")); err != nil {
		t.Fatal(err)
	}
	_ = client.SetReadDeadline(time.Now().Add(5 * time.Second))
	var got bytes.Buffer
	buf := make([]byte, 512)
	for {
		n, err := client.Read(buf)
		got.Write(buf[:n])
		if err != nil || bytes.Contains(got.Bytes(), []byte("hello")) {
			break
		}
	}
	if !strings.HasPrefix(got.String(), "FOUND 5") {
		t.Fatalf("v1 reply = %q (file %s)", got.String(), path)
	}
}
