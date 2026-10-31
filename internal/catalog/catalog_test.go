package catalog

import (
	"os"
	"path/filepath"
	"testing"
)

func openTestDB(t *testing.T) *DB {
	t.Helper()
	d, err := Open(filepath.Join(t.TempDir(), "Data", "wayerpc.db"), "test")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { d.Close() })
	return d
}

func seedFile(t *testing.T, dir, name string) string {
	t.Helper()
	p := filepath.Join(dir, name)
	if err := os.WriteFile(p, []byte("content of "+name), 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestAddListRemove(t *testing.T) {
	d := openTestDB(t)
	dir := t.TempDir()
	p := seedFile(t, dir, "report.pdf")
	abs, err := d.AddPath(p)
	if err != nil || abs == "" {
		t.Fatalf("AddPath: %v %q", err, abs)
	}
	// Re-add is idempotent (INSERT OR REPLACE).
	if _, err := d.AddPath(p); err != nil {
		t.Fatalf("re-add: %v", err)
	}
	rows, err := d.List()
	if err != nil || len(rows) != 1 || rows[0].Name != "report.pdf" {
		t.Fatalf("List: %v %+v", err, rows)
	}
	if n, err := d.Remove(p); err != nil || n != 1 {
		t.Fatalf("Remove: %v %d", err, n)
	}
	rows, _ = d.List()
	if len(rows) != 0 {
		t.Fatalf("expected empty after remove: %+v", rows)
	}
}

func TestAddMissingFails(t *testing.T) {
	d := openTestDB(t)
	if _, err := d.AddPath(filepath.Join(t.TempDir(), "nope.txt")); err == nil {
		t.Fatal("expected error for missing file")
	}
	if _, _, err := d.AddFolder(filepath.Join(t.TempDir(), "nope")); err == nil {
		t.Fatal("expected error for missing folder")
	}
}

func TestAddFolder(t *testing.T) {
	d := openTestDB(t)
	dir := t.TempDir()
	seedFile(t, dir, "a.txt")
	sub := filepath.Join(dir, "sub")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	seedFile(t, sub, "b.txt")
	added, skipped, err := d.AddFolder(dir)
	if err != nil || added != 2 || skipped != 0 {
		t.Fatalf("AddFolder: %d %d %v", added, skipped, err)
	}
}

func TestScoreParity(t *testing.T) {
	if Score("report.pdf", "report.pdf") != 1.0 {
		t.Fatal("exact must be 1.0")
	}
	if s := Score("report", "my-report-final.pdf"); s < 0.6 {
		t.Fatalf("substring boost must be >= 0.6, got %v", s)
	}
	if s := Score("xyzzy", "report.pdf"); s >= 0.2 {
		t.Fatalf("unrelated must be < 0.2, got %v", s)
	}
	if s := Score("reprot.pdf", "report.pdf"); s < 0.2 {
		t.Fatalf("typo should still clear 0.2, got %v", s)
	}
}

func TestFindForRequestDropsMissing(t *testing.T) {
	d := openTestDB(t)
	dir := t.TempDir()
	p := seedFile(t, dir, "notes.txt")
	if _, err := d.AddPath(p); err != nil {
		t.Fatal(err)
	}
	hits, err := d.FindForRequest("notes", DefaultCutoff)
	if err != nil || len(hits) != 1 {
		t.Fatalf("expected 1 hit: %v %+v", err, hits)
	}
	if err := os.Remove(p); err != nil {
		t.Fatal(err)
	}
	hits, err = d.FindForRequest("notes", DefaultCutoff)
	if err != nil || len(hits) != 0 {
		t.Fatalf("expected 0 hits after delete: %v %+v", err, hits)
	}
}

func TestSettingsRoundTrip(t *testing.T) {
	d := openTestDB(t)
	if err := d.SetSetting("library_view", "catalog"); err != nil {
		t.Fatal(err)
	}
	v, err := d.GetSetting("library_view")
	if err != nil || v != "catalog" {
		t.Fatalf("GetSetting: %q %v", v, err)
	}
	if v, _ := d.GetSetting("missing"); v != "" {
		t.Fatalf("expected empty for missing key, got %q", v)
	}
}

func TestOpenStampsSchemaVersion(t *testing.T) {
	dir := t.TempDir()
	db, err := Open(filepath.Join(dir, "wayerpc.db"), "test")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	v, err := db.SchemaVersionNow()
	if err != nil {
		t.Fatal(err)
	}
	if v != SchemaVersion {
		t.Fatalf("user_version = %d, want %d", v, SchemaVersion)
	}
}

func TestReopenKeepsDataAndVersion(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "wayerpc.db")
	file := filepath.Join(dir, "keep.txt")
	if err := os.WriteFile(file, []byte("hi"), 0o644); err != nil {
		t.Fatal(err)
	}
	db, err := Open(path, "test")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.AddPath(file); err != nil {
		t.Fatal(err)
	}
	if err := db.SetSetting("ui_sidebar_collapsed", "1"); err != nil {
		t.Fatal(err)
	}
	db.Close()

	db2, err := Open(path, "test")
	if err != nil {
		t.Fatal(err)
	}
	defer db2.Close()
	rows, err := db2.List()
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 || rows[0].Name != "keep.txt" {
		t.Fatalf("rows after reopen = %+v", rows)
	}
	if v, _ := db2.GetSetting("ui_sidebar_collapsed"); v != "1" {
		t.Fatalf("setting lost on reopen: %q", v)
	}
	if v, _ := db2.SchemaVersionNow(); v != SchemaVersion {
		t.Fatalf("user_version after reopen = %d", v)
	}
}

func TestMigrationsAreIdempotent(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "wayerpc.db")
	for i := 0; i < 3; i++ {
		db, err := Open(path, "test")
		if err != nil {
			t.Fatalf("open #%d: %v", i, err)
		}
		db.Close()
	}
}
