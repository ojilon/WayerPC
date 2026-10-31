package index

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"wayerpc/internal/catalog"
	"wayerpc/internal/search"
)

func testIndexer(t *testing.T) (*Indexer, string) {
	t.Helper()
	dir := t.TempDir()
	db, err := catalog.Open(filepath.Join(dir, "Data", "wayerpc.db"), "test")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	return New(db.SQL()), dir
}

func writeFile(t *testing.T, dir, name string) string {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	p := filepath.Join(dir, name)
	if err := os.WriteFile(p, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestBuildIndexesFilesAndSkipsNoise(t *testing.T) {
	ix, dir := testIndexer(t)
	root := filepath.Join(dir, "Documents")
	writeFile(t, root, "notes.txt")
	writeFile(t, filepath.Join(root, "reports"), "ecology-report.pdf")
	writeFile(t, filepath.Join(root, ".git"), "config")
	writeFile(t, filepath.Join(root, "node_modules", "pkg"), "index.js")
	writeFile(t, filepath.Join(root, ".cache"), "blob")
	writeFile(t, filepath.Join(root, "AppData", "Local"), "secret.db")

	if err := ix.Build(context.Background(), Options{Roots: []string{root}}); err != nil {
		t.Fatal(err)
	}
	n, err := ix.Count()
	if err != nil {
		t.Fatal(err)
	}
	if n != 2 {
		t.Fatalf("indexed %d rows, want 2 (.git/node_modules/.cache/AppData skipped)", n)
	}
	if ix.BuiltAt().IsZero() {
		t.Fatal("built_at was not stamped")
	}
	if ix.NeedsBuild(24 * time.Hour) {
		t.Fatal("fresh index should not need a rebuild")
	}
}

func TestSkipGuardMatchesLiveSearch(t *testing.T) {
	// The index must never surface something the live search hides.
	paths := []string{
		"Documents/todo.txt",
		"Windows/System32/cmd.exe",
		"Program Files/app/a.txt",
		"AppData/Local/x.db",
		"$RECYCLE.BIN/foo",
		"projects/WayerPC/data/shared/a.txt",
		"archive/Reports/ecology.pdf",
	}
	skip := buildSkip(nil)
	for _, p := range paths {
		liveSkipped := search.SkipRel(p)
		indexSkipped := skip(filepath.FromSlash(p), false)
		if liveSkipped != indexSkipped {
			t.Errorf("%s: search skips=%v, index skips=%v", p, liveSkipped, indexSkipped)
		}
	}
}

func TestExtraSkipList(t *testing.T) {
	skip := buildSkip([]string{"SteamLibrary", "Temp"})
	if !skip(filepath.FromSlash("Games/SteamLibrary/x.bin"), false) {
		t.Fatal("extra skip segment ignored")
	}
	if !skip("Temp/x.log", false) {
		t.Fatal("extra skip segment ignored (relative)")
	}
	if skip("Documents/steam.txt", false) {
		t.Fatal("substring of a skip segment must not be skipped")
	}
}

func TestIncrementalUpdate(t *testing.T) {
	ix, dir := testIndexer(t)
	root := filepath.Join(dir, "Docs")
	writeFile(t, root, "first.txt")
	if err := ix.Build(context.Background(), Options{Roots: []string{root}}); err != nil {
		t.Fatal(err)
	}
	if n, _ := ix.Count(); n != 1 {
		t.Fatalf("count after build = %d, want 1", n)
	}

	// A new file appears after the walk: Add() indexes just it.
	added := writeFile(t, root, "second-report.pdf")
	if err := ix.Add(added); err != nil {
		t.Fatal(err)
	}
	if n, _ := ix.Count(); n != 2 {
		t.Fatalf("count after Add = %d, want 2", n)
	}
	hits, err := ix.Candidates([]string{"second"}, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(hits) != 1 || hits[0].Name != "second-report.pdf" || hits[0].Ext != "pdf" {
		t.Fatalf("candidates = %+v", hits)
	}

	// Removing the file drops its row (Add on a missing path prunes).
	if err := os.Remove(added); err != nil {
		t.Fatal(err)
	}
	if err := ix.Add(added); err != nil {
		t.Fatal(err)
	}
	if n, _ := ix.Count(); n != 1 {
		t.Fatalf("count after remove = %d, want 1", n)
	}
}

func TestSourcesIncludeDataFoldersAndCatalog(t *testing.T) {
	ix, dir := testIndexer(t)
	dataRoot := filepath.Join(dir, "WayerPC-data")
	received := filepath.Join(dataRoot, "received")
	writeFile(t, received, "from-phone.pdf")
	writeFile(t, received, "ignore.tmp") // no skip on plain sources
	catalogPath := writeFile(t, filepath.Join(dir, "elsewhere"), "catalogued.docx")

	err := ix.Build(context.Background(), Options{
		Roots: nil,
		Sources: func() []string {
			return []string{received, catalogPath}
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	hits, err := ix.Candidates([]string{"from-phone"}, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(hits) != 1 {
		t.Fatalf("received/ source not indexed: %+v", hits)
	}
	if hits, _ := ix.Candidates([]string{"catalogued"}, 10); len(hits) != 1 {
		t.Fatalf("catalog source not indexed")
	}
}

func TestRebuildClearsStaleRows(t *testing.T) {
	ix, dir := testIndexer(t)
	root := filepath.Join(dir, "Docs")
	keep := writeFile(t, root, "keep.txt")
	gone := writeFile(t, root, "gone.txt")
	if err := ix.Build(context.Background(), Options{Roots: []string{root}}); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(gone); err != nil {
		t.Fatal(err)
	}
	if err := ix.Rebuild(context.Background(), Options{Roots: []string{root}}); err != nil {
		t.Fatal(err)
	}
	if n, _ := ix.Count(); n != 1 {
		t.Fatalf("count after rebuild = %d, want 1", n)
	}
	if hits, _ := ix.Candidates([]string{"keep"}, 10); len(hits) != 1 || hits[0].Path != keep {
		t.Fatalf("kept file missing: %+v", hits)
	}
}

func TestCandidatePrefilter(t *testing.T) {
	ix, dir := testIndexer(t)
	root := filepath.Join(dir, "Docs")
	writeFile(t, root, "ecology-report.pdf")
	writeFile(t, root, "budget.xlsx")
	if err := ix.Build(context.Background(), Options{Roots: []string{root}}); err != nil {
		t.Fatal(err)
	}
	hits, err := ix.Candidates([]string{"ecology"}, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(hits) != 1 || hits[0].Name != "ecology-report.pdf" {
		t.Fatalf("prefilter = %+v", hits)
	}
	// Every token must match (AND), and wildcards are ignored, not executed.
	if hits, _ := ix.Candidates([]string{"ecology", "budget"}, 10); len(hits) != 0 {
		t.Fatalf("AND prefilter returned %+v", hits)
	}
	if hits, _ := ix.Candidates([]string{"%", "_"}, 10); len(hits) != 0 {
		t.Fatalf("wildcards must be ignored, got %+v", hits)
	}
}

func TestCapStopsWalk(t *testing.T) {
	ix, dir := testIndexer(t)
	root := filepath.Join(dir, "Docs")
	for i := 0; i < 12; i++ {
		writeFile(t, root, "file-"+itoa64(int64(i))+".txt")
	}
	if err := ix.Build(context.Background(), Options{Roots: []string{root}, MaxRows: 5}); err != nil {
		t.Fatal(err)
	}
	n, _ := ix.Count()
	if n > 5 {
		t.Fatalf("cap ignored: %d rows", n)
	}
	if st := ix.State(); st.Note == "" {
		t.Fatal("cap should be reported in the state note")
	}
}

func TestCancelStopsWalk(t *testing.T) {
	ix, dir := testIndexer(t)
	root := filepath.Join(dir, "Docs")
	for i := 0; i < 50; i++ {
		writeFile(t, root, "file-"+itoa64(int64(i))+".txt")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := ix.Build(ctx, Options{Roots: []string{root}}); err == nil {
		t.Fatal("cancelled build should return the context error")
	}
	if ix.BuiltAt().IsZero() == false {
		t.Fatal("a cancelled walk must not stamp built_at")
	}
	if st := ix.State(); st.Building {
		t.Fatal("building flag left set after cancel")
	}
}

func TestRemoveUnder(t *testing.T) {
	ix, dir := testIndexer(t)
	root := filepath.Join(dir, "Imported")
	writeFile(t, root, "a.txt")
	writeFile(t, filepath.Join(root, "sub"), "b.txt")
	if err := ix.Build(context.Background(), Options{Roots: []string{root}}); err != nil {
		t.Fatal(err)
	}
	if err := ix.RemoveUnder(root); err != nil {
		t.Fatal(err)
	}
	if n, _ := ix.Count(); n != 0 {
		t.Fatalf("RemoveUnder left %d rows", n)
	}
}
