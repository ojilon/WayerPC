package server

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"wayerpc/internal/catalog"
	"wayerpc/internal/storage"
)

func testServer(t *testing.T) (*Server, *storage.Store, []string) {
	t.Helper()
	var logs []string
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
	srv := New("127.0.0.1", 0, store, cat, func(l string) { logs = append(logs, l) })
	srv.cmdSettle = 50 * time.Millisecond // fast framing tests; prod uses 500ms
	return srv, store, logs
}

func seed(t *testing.T, dir, name, body string) string {
	t.Helper()
	p := filepath.Join(dir, name)
	if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

// runCommand executes one request/response line over net.Pipe.
// net.Pipe is unbuffered, so the client writes, then reads (the server
// goroutine alternates the same way) — never wait for completion first.
func runCommand(t *testing.T, srv *Server, line string) (string, int, string) {
	t.Helper()
	client, server := net.Pipe()
	defer client.Close()
	defer server.Close()
	type result struct {
		status int
		detail string
		fatal  bool
	}
	done := make(chan result, 1)
	go func() {
		r := bufio.NewReader(server)
		cmd, err := readCommand(r, server, srv.settle())
		if err != nil {
			done <- result{-99, err.Error(), false}
			return
		}
		st, detail, fatal := srv.handleCommand(cmd, server, r)
		done <- result{st, detail, fatal}
	}()
	if _, err := fmt.Fprint(client, line + "\n"); err != nil {
		t.Fatal(err)
	}
	out := readAll(client)
	select {
	case res := <-done:
		if res.fatal {
			t.Fatalf("unexpected fatal for %q: %s", line, res.detail)
		}
		return out, res.status, res.detail
	case <-time.After(5 * time.Second):
		t.Fatal("server did not answer")
		return "", 0, ""
	}
}

func readAll(c net.Conn) string {
	_ = c.SetReadDeadline(time.Now().Add(300 * time.Millisecond))
	var b strings.Builder
	buf := make([]byte, 64*1024)
	for {
		n, err := c.Read(buf)
		if n > 0 {
			b.Write(buf[:n])
		}
		if err != nil {
			break
		}
	}
	return b.String()
}

func TestAskStreamsExactFile(t *testing.T) {
	srv, _, _ := testServer(t)
	dir := t.TempDir()
	p := seed(t, dir, "report.pdf", "PDF-BYTES-12345")
	if _, err := srv.cat.AddPath(p); err != nil {
		t.Fatal(err)
	}
	out, status, _ := runCommand(t, srv, "/ask report.pdf")
	if status != StatusAskOK {
		t.Fatalf("status=%d out=%q", status, out)
	}
	if !strings.HasPrefix(out, "FOUND 15\n") {
		t.Fatalf("bad FOUND header: %q", out)
	}
	if !strings.HasSuffix(out, "PDF-BYTES-12345") {
		t.Fatalf("bad body: %q", out)
	}
	if got := srv.Snapshot().BytesSent; got != 15 {
		t.Fatalf("bytesSent=%d", got)
	}
}

func TestAskDuplicatePrefixTolerated(t *testing.T) {
	srv, _, _ := testServer(t)
	p := seed(t, t.TempDir(), "dup.txt", "DUP")
	if _, err := srv.cat.AddPath(p); err != nil {
		t.Fatal(err)
	}
	out, status, _ := runCommand(t, srv, "/ask /ask dup.txt")
	if status != StatusAskOK || !strings.HasPrefix(out, "FOUND 3\n") {
		t.Fatalf("status=%d out=%q", status, out)
	}
}

// serveRaw runs one command against exactly the bytes writeFn puts on the
// wire (newline optional — mirrors the Android client, which sends the
// header with no line ending and then waits for the reply).
func serveRaw(t *testing.T, srv *Server, writeFn func(client net.Conn)) (string, int, string, bool) {
	t.Helper()
	client, server := net.Pipe()
	defer client.Close()
	defer server.Close()
	type result struct {
		status int
		detail string
		fatal  bool
	}
	done := make(chan result, 1)
	go func() {
		r := bufio.NewReader(server)
		cmd, err := readCommand(r, server, srv.settle())
		if err != nil {
			done <- result{-99, err.Error(), false}
			return
		}
		st, detail, fatal := srv.handleCommand(cmd, server, r)
		done <- result{st, detail, fatal}
	}()
	writeFn(client)
	out := readAll(client)
	select {
	case res := <-done:
		return out, res.status, res.detail, res.fatal
	case <-time.After(5 * time.Second):
		t.Fatal("server did not answer")
		return "", 0, "", false
	}
}

func TestAskWithoutNewline(t *testing.T) {
	// Exact phone behavior: header bytes, no "\n", then wait for reply.
	// The server must answer after the settle window, not hang for minutes.
	srv, _, _ := testServer(t)
	p := seed(t, t.TempDir(), "report.pdf", "PDF-BYTES-12345")
	if _, err := srv.cat.AddPath(p); err != nil {
		t.Fatal(err)
	}
	out, status, _, fatal := serveRaw(t, srv, func(client net.Conn) {
		if _, err := client.Write([]byte("/ask report.pdf")); err != nil {
			t.Errorf("write: %v", err)
		}
	})
	if fatal {
		t.Fatal("newline-less command must not be fatal")
	}
	if status != StatusAskOK || !strings.HasPrefix(out, "FOUND 15\n") {
		t.Fatalf("status=%d out=%q", status, out)
	}
}

func TestSplitHeaderJoinsWithinSettle(t *testing.T) {
	// One flush split across TCP segments must still parse as one command.
	srv, _, _ := testServer(t)
	p := seed(t, t.TempDir(), "report.pdf", "PDF-BYTES-12345")
	if _, err := srv.cat.AddPath(p); err != nil {
		t.Fatal(err)
	}
	out, status, _, _ := serveRaw(t, srv, func(client net.Conn) {
		if _, err := client.Write([]byte("/ask rep")); err != nil {
			t.Errorf("write: %v", err)
			return
		}
		time.Sleep(10 * time.Millisecond) // well inside the 50ms test settle
		if _, err := client.Write([]byte("ort.pdf")); err != nil {
			t.Errorf("write: %v", err)
		}
	})
	if status != StatusAskOK || !strings.HasPrefix(out, "FOUND 15\n") {
		t.Fatalf("status=%d out=%q", status, out)
	}
}

func TestEofWithDataProcessesCommand(t *testing.T) {
	// Client sends a newline-less header then half-closes: the bytes must
	// still be processed (not discarded as a bare EOF).
	srv, _, _ := testServer(t)
	p := seed(t, t.TempDir(), "report.pdf", "PDF-BYTES-12345")
	if _, err := srv.cat.AddPath(p); err != nil {
		t.Fatal(err)
	}
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	type result struct {
		status int
		detail string
		fatal  bool
	}
	done := make(chan result, 1)
	go func() {
		c, err := ln.Accept()
		if err != nil {
			return
		}
		defer c.Close()
		r := bufio.NewReader(c)
		cmd, rerr := readCommand(r, c, srv.settle())
		if rerr != nil {
			done <- result{-99, rerr.Error(), false}
			return
		}
		st, detail, fatal := srv.handleCommand(cmd, c, r)
		done <- result{st, detail, fatal}
	}()
	client, err := net.Dial("tcp", ln.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()
	if _, err := client.Write([]byte("/ask report.pdf")); err != nil {
		t.Fatal(err)
	}
	if tc, ok := client.(*net.TCPConn); ok {
		_ = tc.CloseWrite() // EOF with data pending
	}
	out := readAll(client)
	select {
	case res := <-done:
		if res.status != StatusAskOK || res.fatal {
			t.Fatalf("status=%d fatal=%v detail=%q", res.status, res.fatal, res.detail)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("server did not answer")
	}
	if !strings.HasPrefix(out, "FOUND 15\n") || !strings.HasSuffix(out, "PDF-BYTES-12345") {
		t.Fatalf("bad reply: %q", out)
	}
}

func TestAskMatchesList(t *testing.T) {
	srv, _, _ := testServer(t)
	dir := t.TempDir()
	for _, n := range []string{"summer-photo.jpg", "summer-photo-2.jpg", "winter-photo.jpg"} {
		p := seed(t, dir, n, "x")
		if _, err := srv.cat.AddPath(p); err != nil {
			t.Fatal(err)
		}
	}
	out, status, _ := runCommand(t, srv, "/ask summer")
	if status != StatusMatches {
		t.Fatalf("status=%d out=%q", status, out)
	}
	if !strings.HasPrefix(out, "MATCHES ") || !strings.Contains(out, "summer-photo.jpg") {
		t.Fatalf("bad MATCHES: %q", out)
	}
}

func TestAskNotFound(t *testing.T) {
	srv, _, _ := testServer(t)
	out, status, _ := runCommand(t, srv, "/ask nope-never-exists.xyz")
	if status != StatusFail || !strings.HasPrefix(out, "ERROR file_not_found") {
		t.Fatalf("status=%d out=%q", status, out)
	}
}

func TestAskFallsBackToShared(t *testing.T) {
	srv, store, _ := testServer(t)
	seed(t, store.SharedDir(), "dropped.txt", "DROP")
	out, status, _ := runCommand(t, srv, "/ask dropped.txt")
	if status != StatusAskOK || !strings.HasPrefix(out, "FOUND 4\n") {
		t.Fatalf("status=%d out=%q", status, out)
	}
}

func TestUnknownCommand(t *testing.T) {
	srv, _, _ := testServer(t)
	out, status, _ := runCommand(t, srv, "/dance")
	if status != StatusProtocol || !strings.HasPrefix(out, "ERROR unknown_protocol_command.") {
		t.Fatalf("status=%d out=%q", status, out)
	}
}

func TestUploadQuerySearchesCatalog(t *testing.T) {
	srv, _, _ := testServer(t)
	p := seed(t, t.TempDir(), "holiday.mp4", "x")
	if _, err := srv.cat.AddPath(p); err != nil {
		t.Fatal(err)
	}
	out, status, _ := runCommand(t, srv, "/upload holiday")
	if status != StatusMatches || !strings.Contains(out, "holiday.mp4") {
		t.Fatalf("status=%d out=%q", status, out)
	}
}

func TestUploadBinaryRoundTrip(t *testing.T) {
	srv, store, _ := testServer(t)
	body := "HELLO-PHONE-BYTES"
	client, server := net.Pipe()
	defer client.Close()
	defer server.Close()
	type result struct {
		status int
		detail string
		fatal  bool
	}
	done := make(chan result, 1)
	go func() {
		r := bufio.NewReader(server)
		cmd, err := readCommand(r, server, srv.settle())
		if err != nil {
			done <- result{-99, err.Error(), false}
			return
		}
		st, detail, fatal := srv.handleCommand(cmd, server, r)
		done <- result{st, detail, fatal}
	}()
	if _, err := fmt.Fprintf(client, "/upload %d greeting.txt\n", len(body)); err != nil {
		t.Fatal(err)
	}
	// Server answers bare "READY" (no newline), then we stream the bytes.
	ready := make([]byte, 5)
	if _, err := io.ReadFull(client, ready); err != nil || string(ready) != "READY" {
		t.Fatalf("no READY: %q %v", ready, err)
	}
	if _, err := client.Write([]byte(body)); err != nil {
		t.Fatal(err)
	}
	out := readAll(client) // "DONE\n"
	var res result
	select {
	case res = <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("server did not answer")
	}
	if res.status != StatusUploadOK {
		t.Fatalf("status=%d out=%q", res.status, out)
	}
	if res.fatal {
		t.Fatalf("happy-path upload must not be fatal: %s", res.detail)
	}
	if !strings.Contains(out, "DONE") {
		t.Fatalf("no DONE: %q", out)
	}
	saved, err := os.ReadFile(filepath.Join(store.ReceivedDir(), "greeting.txt"))
	if err != nil || string(saved) != body {
		t.Fatalf("saved file mismatch: %q %v", saved, err)
	}
	hits, _ := srv.cat.FindForRequest("greeting", catalog.DefaultCutoff)
	if len(hits) != 1 {
		t.Fatalf("uploaded file not cataloged: %+v", hits)
	}
	if got := srv.Snapshot().BytesReceived; got != int64(len(body)) {
		t.Fatalf("bytesReceived=%d", got)
	}
}

func TestReadySendFailureIsFatal(t *testing.T) {
	// Regression: phone idles past the deadline (or vanishes), then the
	// /upload header arrives but READY cannot be delivered
	// ("write tcp …: i/o timeout"). The session must die immediately
	// instead of lingering as a ghost connection.
	srv, _, _ := testServer(t)
	client, server := net.Pipe()
	_ = client.Close() // peer gone — server writes now fail
	defer server.Close()
	r := bufio.NewReader(server)
	status, detail, fatal := srv.handleCommand("/upload 10 gone.pptx", server, r)
	if status != StatusProtocol {
		t.Fatalf("status=%d detail=%q", status, detail)
	}
	if !fatal {
		t.Fatal("undeliverable READY must be fatal")
	}
	if !strings.Contains(detail, "failed to send READY") {
		t.Fatalf("detail should name READY: %q", detail)
	}
}

func TestUploadIncomplete(t *testing.T) {
	srv, store, _ := testServer(t)
	// Real loopback socket: net.Pipe has no half-close, and the server only
	// notices the short upload once the client stops sending.
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	go func() {
		c, err := ln.Accept()
		if err != nil {
			return
		}
		srv.serveConn(c)
	}()
	client, err := net.Dial("tcp", ln.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()
	if _, err := fmt.Fprint(client, "/upload 100 short.bin\n"); err != nil {
		t.Fatal(err)
	}
	_ = client.SetReadDeadline(time.Now().Add(5 * time.Second))
	r := bufio.NewReader(client)
	ready, err := r.ReadString('Y') // tail of "READY"
	if err != nil || !strings.HasSuffix(ready, "READY") {
		t.Fatalf("no READY: %q %v", ready, err)
	}
	if _, err := client.Write([]byte("only-a-fragment")); err != nil {
		t.Fatal(err)
	}
	if tc, ok := client.(*net.TCPConn); ok {
		_ = tc.CloseWrite()
	}
	resp, err := r.ReadString('\n')
	if err != nil {
		t.Fatalf("no error reply: %v", err)
	}
	if !strings.Contains(resp, "ERROR upload_incomplete") {
		t.Fatalf("expected upload_incomplete, got %q", resp)
	}
	if _, err := os.Stat(filepath.Join(store.ReceivedDir(), "short.bin")); !os.IsNotExist(err) {
		t.Fatal("partial file must be removed")
	}
}

func TestStartStopLiveSocket(t *testing.T) {
	srv, _, _ := testServer(t)
	srv.host = "127.0.0.1"
	srv.port = 0
	lnErr := make(chan error, 1)
	go func() { lnErr <- srv.Start() }()
	deadline := time.Now().Add(5 * time.Second)
	for {
		snap := srv.Snapshot()
		if snap.Running {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("server never became ready")
		}
		time.Sleep(20 * time.Millisecond)
	}
	// Find the real port via a live connection through Snapshot? Addr is
	// host:0; instead connect via the listener port is internal — so just
	// verify stats + stop behavior here (protocol covered over net.Pipe).
	if err := srv.Stop(); err != nil {
		t.Fatal(err)
	}
	select {
	case <-lnErr:
	case <-time.After(5 * time.Second):
		t.Fatal("Start did not return after Stop")
	}
	if srv.Snapshot().Running {
		t.Fatal("still running after Stop")
	}
}
