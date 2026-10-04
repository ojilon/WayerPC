package search

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSkipPath(t *testing.T) {
	blocked := []string{
		`C:\Windows\System32\cmd.exe`,
		`C:\Program Files\app\a.txt`,
		`C:\Users\me\AppData\Local\x`,
		`D:\$RECYCLE.BIN\foo`,
		`D:\projects\WayerPC\data\shared\f.txt`,
	}
	for _, p := range blocked {
		if !SkipPath(p) {
			t.Errorf("expected skip: %s", p)
		}
	}
	if SkipPath(`D:\projects\notes\todo.txt`) {
		t.Error("unexpected skip of user file")
	}
}

func TestSearchRootsFindsFilesAndFolders(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "report-final.pdf"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	sub := filepath.Join(root, "my reports")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(sub, "inside.txt"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	hits, err := SearchRoots([]string{root}, "report", 200)
	if err != nil {
		t.Fatal(err)
	}
	names := map[string]bool{}
	for _, h := range hits {
		names[h.Name] = true
	}
	if !names["report-final.pdf"] {
		t.Errorf("missing file hit: %+v", hits)
	}
	if !names["inside.txt"] {
		t.Errorf("missing folder-contents hit: %+v", hits)
	}
}

func TestSearchRootsLimit(t *testing.T) {
	root := t.TempDir()
	for i := 0; i < 10; i++ {
		name := filepath.Join(root, "qfile.txt")
		if i > 0 {
			name = filepath.Join(root, "qfile"+string(rune('0'+i))+".txt")
		}
		if err := os.WriteFile(name, []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	hits, _ := SearchRoots([]string{root}, "qfile", 3)
	if len(hits) != 3 {
		t.Fatalf("expected 3 hits, got %d", len(hits))
	}
}

func TestSearchAppFoldersExact(t *testing.T) {
	shared := filepath.Join(t.TempDir(), "shared")
	if err := os.MkdirAll(shared, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(shared, "Photo.JPG"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	got := SearchAppFolders("photo.jpg", []string{shared})
	if len(got) != 1 {
		t.Fatalf("expected 1 exact hit (case-insensitive), got %+v", got)
	}
	if got := SearchAppFolders("photo", []string{shared}); len(got) != 0 {
		t.Fatalf("partial must not match exact search: %+v", got)
	}
}
