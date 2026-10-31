package wire

import (
	"bytes"
	"encoding/json"
	"net"
	"strings"
	"testing"
	"time"
)

func pipePair(t *testing.T) (*Conn, *Conn) {
	t.Helper()
	a, b := net.Pipe()
	t.Cleanup(func() { a.Close(); b.Close() })
	return NewConn(a, nil), NewConn(b, nil)
}

func TestFrameRoundTrip(t *testing.T) {
	c1, c2 := pipePair(t)
	msg := Hello{T: THello, Proto: 2, Role: "control", Device: "Pixel 8", DID: "uuid-1", Caps: []string{"query"}}

	go func() {
		if err := c1.WriteJSON(msg); err != nil {
			t.Errorf("write: %v", err)
		}
	}()
	typ, payload, err := c2.ReadMessage()
	if err != nil {
		t.Fatal(err)
	}
	if typ != THello {
		t.Fatalf("type = %q", typ)
	}
	var got Hello
	if err := Decode(payload, &got); err != nil {
		t.Fatal(err)
	}
	if got.Proto != 2 || got.Role != "control" || got.DID != "uuid-1" {
		t.Fatalf("decoded = %+v", got)
	}
}

func TestFrameEncodingIsByteExact(t *testing.T) {
	// "@<len>\n" + JSON, nothing else.
	var buf bytes.Buffer
	c := NewConn(&fakeConn{w: &buf}, nil)
	if err := c.WriteJSON(Pong{T: TPong}); err != nil {
		t.Fatal(err)
	}
	body := `{"t":"pong"}`
	want := "@" + itoa(len(body)) + "\n" + body
	if buf.String() != want {
		t.Fatalf("framing = %q, want %q", buf.String(), want)
	}
}

func TestRejectsBadMarker(t *testing.T) {
	c := NewConn(&fakeConn{r: strings.NewReader("/ask file.pdf")}, nil)
	if _, _, err := c.ReadMessage(); err == nil {
		t.Fatal("a v1 line must not parse as a v2 frame")
	}
}

func TestRejectsOversizeFrame(t *testing.T) {
	c := NewConn(&fakeConn{r: strings.NewReader("@5000000\n")}, nil)
	if _, _, err := c.ReadMessage(); err != ErrFrameTooLarge {
		t.Fatalf("err = %v, want ErrFrameTooLarge", err)
	}
	c2 := NewConn(&fakeConn{r: strings.NewReader("@x\n")}, nil)
	if _, _, err := c2.ReadMessage(); err == nil {
		t.Fatal("non-numeric length must fail")
	}
}

func TestRejectsFrameWithoutType(t *testing.T) {
	body := `{"x":1}`
	c := NewConn(&fakeConn{r: strings.NewReader("@" + itoa(len(body)) + "\n" + body)}, nil)
	if _, _, err := c.ReadMessage(); err == nil {
		t.Fatal("frame without \"t\" must fail")
	}
}

func TestIsV2Sniff(t *testing.T) {
	v2 := NewConn(&fakeConn{r: strings.NewReader("@12\n{}")}, nil)
	if ok, err := IsV2(v2.Reader()); err != nil || !ok {
		t.Fatalf("IsV2(@) = %v, %v", ok, err)
	}
	v1 := NewConn(&fakeConn{r: strings.NewReader("/ask report.pdf")}, nil)
	if ok, err := IsV2(v1.Reader()); err != nil || ok {
		t.Fatalf("IsV2(/) = %v, %v", ok, err)
	}
}

func TestRawBytesAfterFrame(t *testing.T) {
	c1, c2 := pipePair(t)
	payload := bytes.Repeat([]byte("z"), 40_000)
	go func() {
		_ = c1.WriteJSON(Push{T: TPush, ID: "pc-1", Name: "a.bin", Size: int64(len(payload))})
		_, _ = c1.WriteRaw(bytes.NewReader(payload), nil)
	}()
	typ, _, err := c2.ReadMessage()
	if err != nil || typ != TPush {
		t.Fatalf("type=%q err=%v", typ, err)
	}
	var out bytes.Buffer
	var chunks int
	n, err := c2.ReadRaw(&out, int64(len(payload)), func(int64) { chunks++ })
	if err != nil {
		t.Fatal(err)
	}
	if n != int64(len(payload)) || !bytes.Equal(out.Bytes(), payload) {
		t.Fatalf("raw copy mismatch (%d bytes)", n)
	}
	if chunks == 0 {
		t.Fatal("progress callback never fired")
	}
}

func TestSanitizeName(t *testing.T) {
	good := map[string]string{
		"report.pdf":              "report.pdf",
		"  spaced.docx  ":         "spaced.docx",
		`C:\Users\me\photo.jpg`:   "photo.jpg",
		"/tmp/notes.txt":          "notes.txt",
		"folder/sub/file.txt":     "file.txt",
		"Meeting notes 2026.docx": "Meeting notes 2026.docx",
		"console.txt":             "console.txt", // only exact device names are reserved
	}
	for in, want := range good {
		got, err := SanitizeName(in)
		if err != nil {
			t.Errorf("SanitizeName(%q) error: %v", in, err)
			continue
		}
		if got != want {
			t.Errorf("SanitizeName(%q) = %q, want %q", in, got, want)
		}
	}

	bad := []string{
		"", "   ", ".", "..", "../secret", "..\\secret", "a/../b",
		"CON", "con.txt", "PRN.log", "LPT1", "nul.bin",
		"bad\x00name", "ctrl\x01char", "trailing.",
		".hidden", "a:b", "what?.txt", "star*.txt", "pipe|.txt", "quote\".txt",
		strings.Repeat("x", 300),
	}
	for _, in := range bad {
		if got, err := SanitizeName(in); err == nil {
			t.Errorf("SanitizeName(%q) = %q, want an error", in, got)
		}
	}
}

// ---- helpers ----

type fakeConn struct {
	r *strings.Reader
	w *bytes.Buffer
}

func (f *fakeConn) Read(p []byte) (int, error) {
	if f.r == nil {
		return 0, net.ErrClosed
	}
	return f.r.Read(p)
}

func (f *fakeConn) Write(p []byte) (int, error) {
	if f.w == nil {
		return len(p), nil
	}
	return f.w.Write(p)
}

func (f *fakeConn) Close() error                     { return nil }
func (f *fakeConn) LocalAddr() net.Addr              { return &net.TCPAddr{} }
func (f *fakeConn) RemoteAddr() net.Addr             { return &net.TCPAddr{} }
func (f *fakeConn) SetDeadline(time.Time) error      { return nil }
func (f *fakeConn) SetReadDeadline(time.Time) error  { return nil }
func (f *fakeConn) SetWriteDeadline(time.Time) error { return nil }

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var b []byte
	for n > 0 {
		b = append([]byte{byte('0' + n%10)}, b...)
		n /= 10
	}
	return string(b)
}

func TestManifestJSONShape(t *testing.T) {
	// The documented example must round-trip unchanged.
	raw := `{"t":"manifest","id":"pc-50","part":1,"last":true,"items":[{"p":"trip-plan.pdf","size":10,"mtime":5,"sha256":"ab"}]}`
	var m Manifest
	if err := json.Unmarshal([]byte(raw), &m); err != nil {
		t.Fatal(err)
	}
	out, err := json.Marshal(m)
	if err != nil {
		t.Fatal(err)
	}
	if string(out) != raw {
		t.Fatalf("manifest re-encode = %s", out)
	}
}
