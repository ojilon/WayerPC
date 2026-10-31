package main

// Sync bridge test: a fake phone answers manifest requests and performs the
// pushes/pulls the plan asks for. Proves the scan verdicts and that ApplySync
// moves bytes in the arrow direction only.

import (
	"bufio"
	"bytes"
	"context"
	"net"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"wayerpc/internal/server"
	"wayerpc/internal/wire"
)

type phoneShare struct {
	mu     sync.Mutex
	files  map[string][]byte
	mtimes map[string]int64
}

func (p *phoneShare) snapshot() []wire.ManifestItem {
	p.mu.Lock()
	defer p.mu.Unlock()
	out := make([]wire.ManifestItem, 0, len(p.files))
	for name, body := range p.files {
		out = append(out, wire.ManifestItem{P: name, Size: int64(len(body)), MTime: p.mtimes[name]})
	}
	return out
}

// servePhone answers manifests and transfers until the connection closes.
func servePhone(t *testing.T, srv *server.Server, phone *fakePhone, share *phoneShare) {
	t.Helper()
	addr := srv.ListenAddr()
	for {
		_ = phone.conn.SetReadDeadline(time.Now().Add(30 * time.Second))
		typ, payload, err := phone.wc.ReadMessage()
		if err != nil {
			return
		}
		switch typ {
		case wire.TManifestReq:
			var req wire.ManifestReq
			_ = wire.Decode(payload, &req)
			_ = phone.wc.WriteJSON(wire.Manifest{T: wire.TManifest, ID: req.ID, Part: 1, Last: true, Items: share.snapshot()})
		case wire.TOffer:
			var offer wire.Offer
			_ = wire.Decode(payload, &offer)
			_ = phone.wc.WriteJSON(wire.Accept{T: wire.TAccept, ID: offer.ID})
			go func() {
				c, derr := net.Dial("tcp", addr)
				if derr != nil {
					t.Logf("phone data dial: %v", derr)
					return
				}
				defer c.Close()
				dwc := wire.NewConn(c, bufio.NewReaderSize(c, 64*1024))
				_ = dwc.WriteJSON(wire.Hello{T: wire.THello, Proto: 2, Role: "data", DID: "did-e2e"})
				if typ, _, rerr := dwc.ReadMessage(); rerr != nil || typ != wire.THelloOK {
					t.Logf("data hello: %q %v", typ, rerr)
					return
				}
				if werr := dwc.WriteJSON(wire.Pull{T: wire.TPull, ID: offer.ID}); werr != nil {
					t.Logf("pull: %v", werr)
					return
				}
				typ, payload, rerr := dwc.ReadMessage()
				if rerr != nil || typ != wire.TFile {
					t.Logf("file: %q %v payload=%s", typ, rerr, payload)
					return
				}
				var file wire.File
				_ = wire.Decode(payload, &file)
				var body bytes.Buffer
				if _, rerr := dwc.ReadRaw(&body, file.Size, nil); rerr != nil {
					t.Logf("raw: %v", rerr)
					return
				}
				_ = dwc.WriteJSON(wire.Done{T: wire.TDone, ID: file.ID, Bytes: file.Size, SHA: sha256Hex(body.Bytes())})
				share.mu.Lock()
				share.files[file.Name] = body.Bytes()
				share.mtimes[file.Name] = time.Now().Unix()
				share.mu.Unlock()
			}()
		case wire.TGet:
			var get wire.Get
			_ = wire.Decode(payload, &get)
			share.mu.Lock()
			body := share.files[get.Path]
			share.mu.Unlock()
			go func() {
				c, derr := net.Dial("tcp", addr)
				if derr != nil {
					return
				}
				defer c.Close()
				dwc := wire.NewConn(c, bufio.NewReaderSize(c, 64*1024))
				_ = dwc.WriteJSON(wire.Hello{T: wire.THello, Proto: 2, Role: "data", DID: "did-e2e"})
				if typ, _, rerr := dwc.ReadMessage(); rerr != nil || typ != wire.THelloOK {
					return
				}
				_ = dwc.WriteJSON(wire.Push{T: wire.TPush, ID: get.ID, Name: get.Path, Size: int64(len(body)), SHA: sha256Hex(body)})
				if typ, _, rerr := dwc.ReadMessage(); rerr != nil || typ != wire.TReady {
					return
				}
				_, _ = dwc.WriteRaw(bytes.NewReader(body), nil)
				_, _, _ = dwc.ReadMessage()
			}()
		}
	}
}

func TestSyncScanAndApply(t *testing.T) {
	app, srv, dir := newBridgedApp(t)
	received := filepath.Join(dir, "received")
	shared := filepath.Join(dir, "shared")

	writeFile := func(p string, body string, mtime int64) {
		if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
		ts := time.Unix(mtime, 0)
		if err := os.Chtimes(p, ts, ts); err != nil {
			t.Fatal(err)
		}
	}
	// PC side
	writeFile(filepath.Join(received, "only-pc.txt"), "pc only", 1000)
	writeFile(filepath.Join(received, "dup.txt"), "older", 100)
	writeFile(filepath.Join(shared, "dup.txt"), "newer", 5000) // newer wins
	writeFile(filepath.Join(shared, "conflict.txt"), "pc version", 5000)
	writeFile(filepath.Join(received, "same.txt"), "identical", 2000)

	phone := dialPhone(t, srv)
	share := &phoneShare{
		files: map[string][]byte{
			"only-phone.jpg": []byte("phone only"),
			"conflict.txt":   []byte("phone version!!"), // same size-ish, older mtime
			"same.txt":       []byte("identical"),
		},
		mtimes: map[string]int64{
			"only-phone.jpg": 3000,
			"conflict.txt":   900,
			"same.txt":       2000,
		},
	}
	go servePhone(t, srv, phone, share)
	waitForCond(t, func() bool { return app.GetStatus().ControlLinks == 1 })

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	res, _ := app.scanSync(ctx)
	if res.Error != "" {
		t.Fatalf("scan error: %s", res.Error)
	}
	verdicts := map[string]string{}
	for _, e := range res.Rows {
		verdicts[e.Name] = e.Verdict
	}
	want := map[string]string{
		"only-pc.txt":    "only_pc",
		"only-phone.jpg": "only_phone",
		"conflict.txt":   "newer_pc",
		"same.txt":       "same",
		"dup.txt":        "only_pc", // flattened, newer copy, no phone counterpart
	}
	for name, verdict := range want {
		if verdicts[name] != verdict {
			t.Fatalf("%s = %q, want %q (all: %+v)", name, verdicts[name], verdict, verdicts)
		}
	}

	// Apply two files: one push (only-pc.txt) and one pull (only-phone.jpg).
	app.ApplySync([]string{"only-pc.txt", "only-phone.jpg"})

	waitForCond(t, func() bool {
		share.mu.Lock()
		defer share.mu.Unlock()
		_, ok := share.files["only-pc.txt"]
		return ok
	})
	waitForCond(t, func() bool {
		_, err := os.Stat(filepath.Join(received, "only-phone.jpg"))
		return err == nil
	})
	// The pulled file really landed with its bytes.
	body, err := os.ReadFile(filepath.Join(received, "only-phone.jpg"))
	if err != nil || string(body) != "phone only" {
		t.Fatalf("pulled file = %q (%v)", body, err)
	}
	// The next scan sees both files as settled: the push is already on the
	// phone, the pull already on the PC.
	waitForCond(t, func() bool {
		next, _ := app.scanSync(ctx)
		for _, e := range next.Rows {
			if e.Name == "only-pc.txt" && e.Verdict == "only_pc" {
				return false
			}
			if e.Name == "only-phone.jpg" && e.Verdict == "only_phone" {
				return false
			}
		}
		return true
	})
}

func TestSyncWithoutPhone(t *testing.T) {
	app, _, dir := newBridgedApp(t)
	if err := os.WriteFile(filepath.Join(dir, "received", "a.txt"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	res, _ := app.scanSync(ctx)
	if !contains(res.Error, "No phone connected") {
		t.Fatalf("error = %q", res.Error)
	}
	// The event-driven entry points must survive a missing link too.
	app.StartSyncScan()
	app.ApplySync([]string{"a.txt"})
	time.Sleep(100 * time.Millisecond)
}

func contains(haystack, needle string) bool {
	return bytes.Contains([]byte(haystack), []byte(needle))
}
