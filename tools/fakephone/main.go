// Command fakephone is a protocol test client for WayerPC.
//
// It speaks both protocols: v1 (/ask, /upload) and v2 (the '@'-framed link
// described in docs/protocol-v2.md). It is a developer/QA tool — the app never
// needs it — and it is the quickest way to prove the PC side works without an
// Android build.
//
// Usage:
//
//	fakephone -help
package main

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"flag"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"wayerpc/internal/wire"
)

const connectTimeout = 5 * time.Second

// queries remembers each query id's text so a later `pick` can resolve the
// same list (the protocol only sends the id + number back).
var queries sync.Map

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}
	cmd, args := os.Args[1], os.Args[2:]
	var err error
	switch cmd {
	case "ask":
		err = runAsk(args)
	case "upload":
		err = runUpload(args)
	case "probe":
		err = runProbe(args)
	case "phone":
		err = runPhone(args)
	case "-help", "--help", "help":
		usage()
		return
	default:
		usage()
		os.Exit(2)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "fakephone:", err)
		os.Exit(1)
	}
}

func usage() {
	fmt.Fprint(os.Stderr, `fakephone — WayerPC protocol test client

  fakephone ask    -addr HOST:PORT -query TEXT          v1 /ask (FOUND / MATCHES)
  fakephone upload -addr HOST:PORT -file PATH           v1 /upload (READY … DONE)
  fakephone probe  -addr HOST:PORT                      v2 hello + ping/pong
  fakephone phone  -addr HOST:PORT -dir FOLDER [-out FOLDER]
                                                        act as a v2 phone: answer
                                                        queries, push picks, serve
                                                        get + sync manifests, and
                                                        accept offers

Examples:
  fakephone ask -addr 192.168.43.20:5000 -query ecology
  fakephone upload -addr 192.168.43.20:5000 -file ./IMG_4821.jpg
  fakephone phone -addr 192.168.43.20:5000 -dir ./phone-files -out ./from-pc
`)
}

// ---------- v1 ----------

func runAsk(args []string) error {
	fs := flag.NewFlagSet("ask", flag.ExitOnError)
	addr := fs.String("addr", "127.0.0.1:5000", "PC address")
	query := fs.String("query", "", "what to ask for")
	_ = fs.Parse(args)
	if *query == "" {
		return errors.New("ask: -query is required")
	}
	c, err := dialV1(*addr)
	if err != nil {
		return err
	}
	defer c.Close()
	r := bufio.NewReader(c)
	fmt.Fprintf(c, "/ask %s\n", *query)
	line, err := r.ReadString('\n')
	if err != nil {
		return fmt.Errorf("ask: reply: %w", err)
	}
	line = strings.TrimRight(line, "\r\n")
	switch {
	case strings.HasPrefix(line, "FOUND "):
		var size int64
		fmt.Sscanf(line, "FOUND %d", &size)
		fmt.Printf("%s (%d bytes) — streaming\n", line, size)
		n, err := io.Copy(io.Discard, io.LimitReader(r, size))
		if err != nil {
			return err
		}
		fmt.Printf("received %d / %d bytes\n", n, size)
	case strings.HasPrefix(line, "MATCHES "):
		var n int
		fmt.Sscanf(line, "MATCHES %d", &n)
		fmt.Printf("%d match(es):\n", n)
		for i := 0; i < n; i++ {
			name, err := r.ReadString('\n')
			if err != nil {
				return err
			}
			fmt.Printf("  %d. %s", i+1, name)
		}
	case strings.HasPrefix(line, "ERROR"):
		return fmt.Errorf("server said %s", line)
	default:
		fmt.Println("unexpected reply:", line)
	}
	return nil
}

func runUpload(args []string) error {
	fs := flag.NewFlagSet("upload", flag.ExitOnError)
	addr := fs.String("addr", "127.0.0.1:5000", "PC address")
	file := fs.String("file", "", "local file to push")
	_ = fs.Parse(args)
	if *file == "" {
		return errors.New("upload: -file is required")
	}
	data, err := os.ReadFile(*file)
	if err != nil {
		return err
	}
	c, err := dialV1(*addr)
	if err != nil {
		return err
	}
	defer c.Close()
	if _, err := fmt.Fprintf(c, "/upload %d %s\n", len(data), filepath.Base(*file)); err != nil {
		return err
	}
	r := bufio.NewReader(c)
	reply := make([]byte, 5)
	if _, err := io.ReadFull(r, reply); err != nil {
		return err
	}
	if string(reply) != "READY" {
		line, _ := r.ReadString('\n')
		return fmt.Errorf("server said %q%s", reply, line)
	}
	if _, err := c.Write(data); err != nil {
		return err
	}
	line, err := r.ReadString('\n')
	if err != nil {
		return err
	}
	line = strings.TrimSpace(line)
	if line != "DONE" {
		return fmt.Errorf("upload reply %q", line)
	}
	fmt.Printf("uploaded %s (%d bytes)\n", filepath.Base(*file), len(data))
	return nil
}

// ---------- v2 ----------

type v2conn struct {
	net net.Conn
	wc  *wire.Conn
}

// link serialises writes on one control connection so handler goroutines
// cannot interleave frames.
type link struct {
	v  *v2conn
	mu sync.Mutex
}

func (l *link) send(msg any) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.v.wc.WriteJSON(msg)
}

func dialV2(addr string) (*v2conn, error) {
	c, err := dialV1(addr)
	if err != nil {
		return nil, err
	}
	return &v2conn{net: c, wc: wire.NewConn(c, bufio.NewReaderSize(c, 64*1024))}, nil
}

// hello performs the v2 handshake and returns the PC's answer.
func (v *v2conn) hello(role, did string) (wire.HelloOK, error) {
	var ok wire.HelloOK
	_ = v.net.SetDeadline(time.Now().Add(connectTimeout))
	if err := v.wc.WriteJSON(wire.Hello{T: wire.THello, Proto: 2, Role: role, Device: "fakephone", DID: did}); err != nil {
		return ok, err
	}
	typ, payload, err := v.wc.ReadMessage()
	if err != nil {
		return ok, err
	}
	if typ != wire.THelloOK {
		return ok, fmt.Errorf("handshake refused (%s): %s", typ, payload)
	}
	_ = wire.Decode(payload, &ok)
	_ = v.net.SetDeadline(time.Time{})
	return ok, nil
}

func runProbe(args []string) error {
	fs := flag.NewFlagSet("prog", flag.ExitOnError)
	addr := fs.String("addr", "127.0.0.1:5000", "PC address")
	_ = fs.Parse(args)
	v, err := dialV2(*addr)
	if err != nil {
		return err
	}
	defer v.net.Close()
	ok, err := v.hello("control", "fakephone-probe")
	if err != nil {
		return err
	}
	fmt.Printf("hello_ok: proto=%d pc=%q ping=%ds\n", ok.Proto, ok.PC, ok.Ping)
	if err := v.wc.WriteJSON(wire.Ping{T: wire.TPing}); err != nil {
		return err
	}
	_ = v.net.SetReadDeadline(time.Now().Add(connectTimeout))
	typ, _, err := v.wc.ReadMessage()
	if err != nil {
		return err
	}
	if typ != wire.TPong {
		return fmt.Errorf("ping answered with %q", typ)
	}
	fmt.Println("ping/pong ok")
	return nil
}

func runPhone(args []string) error {
	fs := flag.NewFlagSet("phone", flag.ExitOnError)
	addr := fs.String("addr", "127.0.0.1:5000", "PC address")
	dir := fs.String("dir", ".", "folder this 'phone' offers")
	out := fs.String("out", "./from-pc", "folder for files pulled from the PC")
	did := fs.String("did", "fakephone-1", "device id advertised in hello")
	_ = fs.Parse(args)

	if err := os.MkdirAll(*out, 0o755); err != nil {
		return err
	}
	v, err := dialV2(*addr)
	if err != nil {
		return err
	}
	defer v.net.Close()
	ok, err := v.hello("control", *did)
	if err != nil {
		return err
	}
	fmt.Printf("connected to %q (protocol %d, ping %ds) — serving %s\n", ok.PC, ok.Proto, ok.Ping, *dir)
	lnk := &link{v: v}

	for {
		_ = v.net.SetReadDeadline(time.Now().Add(10 * time.Minute))
		typ, payload, err := v.wc.ReadMessage()
		if err != nil {
			return fmt.Errorf("connection closed: %w", err)
		}
		switch typ {
		case wire.TPing:
			_ = v.wc.WriteJSON(wire.Pong{T: wire.TPong})
		case wire.TPong:
			// nothing to do
		case wire.TQuery:
			var q wire.Query
			_ = wire.Decode(payload, &q)
			go answerQuery(lnk, *addr, *did, *dir, q)
		case wire.TPick:
			var p wire.Pick
			_ = wire.Decode(payload, &p)
			go pushPick(lnk, *addr, *did, *dir, p)
		case wire.TGet:
			var g wire.Get
			_ = wire.Decode(payload, &g)
			go pushGet(lnk, *addr, *did, *dir, g)
		case wire.TManifestReq:
			var m wire.ManifestReq
			_ = wire.Decode(payload, &m)
			_ = v.wc.WriteJSON(wire.Manifest{T: wire.TManifest, ID: m.ID, Part: 1, Last: true, Items: manifest(*dir)})
		case wire.TOffer:
			var o wire.Offer
			_ = wire.Decode(payload, &o)
			fmt.Printf("offer: %s (%d bytes) → accepting\n", o.Name, o.Size)
			if err := v.wc.WriteJSON(wire.Accept{T: wire.TAccept, ID: o.ID}); err != nil {
				return err
			}
			go pullOffer(*addr, *did, *out, o)
		case wire.TAbort:
			var a wire.Abort
			_ = wire.Decode(payload, &a)
			fmt.Println("abort for", a.ID)
		case wire.TFile:
			// Inline send on the control link (flow C single match).
			var f wire.File
			_ = wire.Decode(payload, &f)
			var buf strings.Builder
			_ = buf
			dst := filepath.Join(*out, filepath.Base(f.Name))
			n, err := v.wc.ReadRaw(openOrDiscard(dst), f.Size, nil)
			if err != nil {
				return err
			}
			fmt.Printf("inline file %s: %d bytes → %s\n", f.Name, n, dst)
			_ = v.wc.WriteJSON(wire.Done{T: wire.TDone, ID: f.ID, Bytes: n})
		case wire.TError:
			var e wire.Error
			_ = wire.Decode(payload, &e)
			fmt.Printf("error %s: %s\n", e.Code, e.Msg)
		case wire.TOk:
			var o wire.Ok
			_ = wire.Decode(payload, &o)
			fmt.Println("ok for", o.ID)
		default:
			fmt.Println("unhandled message:", typ)
		}
	}
}

// answerQuery replies with a numbered list (or not_found) from -dir.
func answerQuery(l *link, addr, did, dir string, q wire.Query) {
	queries.Store(q.ID, q.Q)
	matches := search(dir, q.Q, wire.MaxMatches)
	if len(matches) == 0 {
		_ = l.send(wire.Error{T: wire.TError, ID: q.ID, Code: wire.CodeNotFound, Msg: "no match"})
		return
	}
	items := make([]wire.Item, 0, len(matches))
	for i, m := range matches {
		items = append(items, wire.Item{N: i + 1, Name: m.Name, Size: m.Size, MTime: m.MTime})
	}
	_ = l.send(wire.Matches{T: wire.TMatches, ID: q.ID, Items: items, TTL: int(wire.QueryTTL / time.Second)})
}

// pushPick pushes match n after the PC asked for it.
func pushPick(l *link, addr, did, dir string, p wire.Pick) {
	text, ok := queries.Load(p.ID)
	query, _ := text.(string)
	if !ok || query == "" {
		fmt.Println("pick for an unknown query id — ignored")
		return
	}
	matches := search(dir, query, wire.MaxMatches)
	if p.N < 1 || p.N > len(matches) {
		_ = l.send(wire.Error{T: wire.TError, ID: p.ID, Code: wire.CodeExpired, Msg: "no such match number"})
		return
	}
	path := filepath.Join(dir, matches[p.N-1].Name)
	if err := pushFile(l, addr, did, p.ID, path); err != nil {
		fmt.Println("pick failed:", err)
	}
}

func pushGet(l *link, addr, did, dir string, g wire.Get) {
	path := filepath.Join(dir, filepath.Base(g.Path))
	if err := pushFile(l, addr, did, g.ID, path); err != nil {
		fmt.Println("get failed:", err)
	}
}

// pushFile opens a data connection and sends one file (flow B / flow E).
func pushFile(l *link, addr, did, id, path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		_ = l.send(wire.Error{T: wire.TError, ID: id, Code: wire.CodeNotFound, Msg: err.Error()})
		return err
	}
	sum := sha256Hex(data)
	v, err := dialV2(addr)
	if err != nil {
		return err
	}
	defer v.net.Close()
	if _, err := v.hello("data", did); err != nil {
		return err
	}
	if err := v.wc.WriteJSON(wire.Push{T: wire.TPush, ID: id, Name: filepath.Base(path), Size: int64(len(data)), SHA: sum}); err != nil {
		return err
	}
	typ, payload, err := v.wc.ReadMessage()
	if err != nil {
		return err
	}
	if typ != wire.TReady {
		return fmt.Errorf("push refused: %s %s", typ, payload)
	}
	var ready wire.Ready
	_ = wire.Decode(payload, &ready)
	if ready.Offset > int64(len(data)) {
		return fmt.Errorf("PC claims %d of %d bytes", ready.Offset, len(data))
	}
	if _, err := v.wc.WriteRaw(strings.NewReader(string(data[ready.Offset:])), nil); err != nil {
		return err
	}
	typ, payload, err = v.wc.ReadMessage()
	if err != nil {
		return err
	}
	if typ != wire.TDone {
		return fmt.Errorf("push failed: %s %s", typ, payload)
	}
	fmt.Printf("pushed %s (%d bytes, resumed at %d)\n", filepath.Base(path), len(data), ready.Offset)
	return nil
}

// pullOffer fetches an accepted offer into -out.
func pullOffer(addr, did, out string, o wire.Offer) {
	v, err := dialV2(addr)
	if err != nil {
		fmt.Println("pull:", err)
		return
	}
	defer v.net.Close()
	if _, err := v.hello("data", did); err != nil {
		fmt.Println("pull hello:", err)
		return
	}
	if err := v.wc.WriteJSON(wire.Pull{T: wire.TPull, ID: o.ID}); err != nil {
		fmt.Println("pull:", err)
		return
	}
	typ, payload, err := v.wc.ReadMessage()
	if err != nil || typ != wire.TFile {
		fmt.Printf("pull refused: %s %v\n", typ, err)
		return
	}
	var f wire.File
	_ = wire.Decode(payload, &f)
	dst := filepath.Join(out, filepath.Base(f.Name))
	n, err := v.wc.ReadRaw(openOrDiscard(dst), f.Size, nil)
	if err != nil {
		fmt.Println("pull raw:", err)
		return
	}
	_ = v.wc.WriteJSON(wire.Done{T: wire.TDone, ID: f.ID, Bytes: n})
	fmt.Printf("pulled %s (%d bytes) → %s\n", f.Name, n, dst)
}

// ---------- helpers ----------

type fileInfo struct {
	Name  string
	Size  int64
	MTime int64
}

func search(dir, query string, max int) []fileInfo {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	q := strings.ToLower(strings.TrimSpace(query))
	var out []fileInfo
	for _, e := range entries {
		if e.IsDir() || strings.HasPrefix(e.Name(), ".") {
			continue
		}
		if q != "" && !strings.Contains(strings.ToLower(e.Name()), q) {
			continue
		}
		info, ierr := e.Info()
		if ierr != nil {
			continue
		}
		out = append(out, fileInfo{Name: e.Name(), Size: info.Size(), MTime: info.ModTime().Unix()})
		if len(out) >= max {
			break
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

func manifest(dir string) []wire.ManifestItem {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	var items []wire.ManifestItem
	for _, e := range entries {
		if e.IsDir() || strings.HasPrefix(e.Name(), ".") {
			continue
		}
		info, ierr := e.Info()
		if ierr != nil {
			continue
		}
		items = append(items, wire.ManifestItem{P: e.Name(), Size: info.Size(), MTime: info.ModTime().Unix()})
	}
	sort.Slice(items, func(i, j int) bool { return items[i].P < items[j].P })
	return items
}

func openOrDiscard(path string) io.Writer {
	f, err := os.Create(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "cannot write %s: %v\n", path, err)
		return io.Discard
	}
	return f
}

func sha256Hex(b []byte) string {
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

func dialV1(addr string) (net.Conn, error) {
	return net.DialTimeout("tcp", addr, connectTimeout)
}
