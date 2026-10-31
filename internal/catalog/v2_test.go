package catalog

import (
	"path/filepath"
	"testing"
)

func TestHashCacheRoundTrip(t *testing.T) {
	db, err := Open(filepath.Join(t.TempDir(), "w.db"), "test")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	if _, ok := db.CachedHash("/a/b.pdf", 10, 100); ok {
		t.Fatal("empty cache reported a hit")
	}
	if err := db.PutHash("/a/b.pdf", "abc123", 10, 100); err != nil {
		t.Fatal(err)
	}
	if sha, ok := db.CachedHash("/a/b.pdf", 10, 100); !ok || sha != "abc123" {
		t.Fatalf("cache miss after put: %q %v", sha, ok)
	}
	// A different size or mtime invalidates the row without a write.
	if _, ok := db.CachedHash("/a/b.pdf", 11, 100); ok {
		t.Fatal("size change served a stale hash")
	}
	if _, ok := db.CachedHash("/a/b.pdf", 10, 101); ok {
		t.Fatal("mtime change served a stale hash")
	}
	db.ForgetHash("/a/b.pdf")
	if _, ok := db.CachedHash("/a/b.pdf", 10, 100); ok {
		t.Fatal("forget left the row behind")
	}
}

func TestDevicesUpsertAndList(t *testing.T) {
	db, err := Open(filepath.Join(t.TempDir(), "w.db"), "test")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	if err := db.UpsertDevice("did-1", "Pixel 8", "192.168.43.21", 100); err != nil {
		t.Fatal(err)
	}
	// A later hello with an empty name keeps the previous one.
	if err := db.UpsertDevice("did-1", "", "192.168.43.30", 200); err != nil {
		t.Fatal(err)
	}
	if err := db.UpsertDevice("did-2", "Galaxy A14", "192.168.43.77", 150); err != nil {
		t.Fatal(err)
	}
	devices, err := db.Devices()
	if err != nil {
		t.Fatal(err)
	}
	if len(devices) != 2 {
		t.Fatalf("devices = %+v", devices)
	}
	if devices[0].DID != "did-1" || devices[0].Name != "Pixel 8" || devices[0].LastIP != "192.168.43.30" {
		t.Fatalf("newest device = %+v", devices[0])
	}
}
