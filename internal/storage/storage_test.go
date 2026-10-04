package storage

import (
	"os"
	"path/filepath"
	"testing"
)

func TestInitCreatesTree(t *testing.T) {
	root := filepath.Join(t.TempDir(), "WayerPC", "data")
	s, err := Init(root)
	if err != nil {
		t.Fatal(err)
	}
	for _, d := range []string{s.SharedDir(), s.ReceivedDir(), s.DataDir()} {
		if st, err := os.Stat(d); err != nil || !st.IsDir() {
			t.Fatalf("missing dir %s: %v", d, err)
		}
	}
	if s.DbPath() != filepath.Join(root, "Data", "wayerpc.db") {
		t.Fatalf("bad db path: %s", s.DbPath())
	}
}

func TestInitEmptyRootFails(t *testing.T) {
	if _, err := Init(""); err == nil {
		t.Fatal("expected error for empty root")
	}
}

func TestListFilesMissingDir(t *testing.T) {
	s, _ := Init(t.TempDir())
	files, err := s.ListFiles(filepath.Join(t.TempDir(), "nope"))
	if err != nil || len(files) != 0 {
		t.Fatalf("expected empty+nil, got %v %v", files, err)
	}
}

func TestListFilesRoundTrip(t *testing.T) {
	s, _ := Init(t.TempDir())
	if err := os.WriteFile(filepath.Join(s.ReceivedDir(), "a.txt"), []byte("hi"), 0o644); err != nil {
		t.Fatal(err)
	}
	files, err := s.ListFiles(s.ReceivedDir())
	if err != nil || len(files) != 1 || files[0].Name != "a.txt" {
		t.Fatalf("unexpected listing: %v %v", files, err)
	}
}
