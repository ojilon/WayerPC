// Package catalog is the SQLite import catalog. Files are NOT copied on
// import — absolute paths are registered and later resolved by /ask.
//
// Schema is compatible with the Python-era imports.db so old installs
// migrate with a single INSERT OR IGNORE (see MigrateLegacy).
package catalog

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	_ "modernc.org/sqlite"
)

// DefaultCutoff mirrors find_related(query, cutoff=0.2): ≥20% similarity.
const DefaultCutoff = 0.2

const schema = `
CREATE TABLE IF NOT EXISTS imported_files (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  path TEXT NOT NULL UNIQUE,
  name TEXT NOT NULL,
  size_bytes INTEGER,
  imported_at TEXT NOT NULL DEFAULT (datetime('now'))
);
CREATE INDEX IF NOT EXISTS idx_imported_files_name ON imported_files(name);
CREATE TABLE IF NOT EXISTS settings (
  key TEXT PRIMARY KEY,
  value TEXT NOT NULL
);
`

// Entry is one catalog row.
type Entry struct {
	ID         int64   `json:"id"`
	Path       string  `json:"path"`
	Name       string  `json:"name"`
	SizeBytes  int64   `json:"sizeBytes"`
	ImportedAt string  `json:"importedAt"`
	Score      float64 `json:"score,omitempty"`
}

// DB is a single-writer SQLite handle.
type DB struct {
	mu   sync.Mutex
	sql  *sql.DB
	path string
}

// Open creates the parent dir, opens (or creates) the database, applies
// pragmas + schema, and stamps the app version.
func Open(dbPath, appVersion string) (*DB, error) {
	if err := os.MkdirAll(filepath.Dir(dbPath), 0o755); err != nil {
		return nil, fmt.Errorf("catalog: mkdir: %w", err)
	}
	sqlDB, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("catalog: open: %w", err)
	}
	for _, pragma := range []string{
		"PRAGMA journal_mode=WAL;",
		"PRAGMA synchronous=NORMAL;",
		"PRAGMA foreign_keys=ON;",
	} {
		if _, err := sqlDB.Exec(pragma); err != nil {
			sqlDB.Close()
			return nil, fmt.Errorf("catalog: pragma: %w", err)
		}
	}
	if _, err := sqlDB.Exec(schema); err != nil {
		sqlDB.Close()
		return nil, fmt.Errorf("catalog: schema: %w", err)
	}
	d := &DB{sql: sqlDB, path: dbPath}
	if appVersion != "" {
		_ = d.SetSetting("app_version", appVersion)
	}
	return d, nil
}

// Close releases the handle.
func (d *DB) Close() error { return d.sql.Close() }

// Path returns the database file path.
func (d *DB) Path() string { return d.path }

// AddPath registers one file (no copy). Missing file → error (legacy
// returned (1, reason); Go callers surface err.Error() instead).
func (d *DB) AddPath(filePath string) (string, error) {
	st, err := os.Stat(filePath)
	if err != nil || st.IsDir() {
		return "", fmt.Errorf("catalog: not a file: %s", filePath)
	}
	abs, err := filepath.Abs(filePath)
	if err != nil {
		return "", err
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	_, err = d.sql.Exec(
		`INSERT OR REPLACE INTO imported_files(path, name, size_bytes, imported_at)
		 VALUES (?,?,?,?)`,
		abs, filepath.Base(abs), st.Size(),
		time.Now().UTC().Format("2006-01-02 15:04:05"),
	)
	if err != nil {
		return "", fmt.Errorf("catalog: insert: %w", err)
	}
	return abs, nil
}

// AddFolder registers every regular file under root. Returns (added, skipped).
func (d *DB) AddFolder(root string) (added, skipped int, err error) {
	st, err := os.Stat(root)
	if err != nil || !st.IsDir() {
		return 0, 1, fmt.Errorf("catalog: not a directory: %s", root)
	}
	err = filepath.WalkDir(root, func(p string, de os.DirEntry, err error) error {
		if err != nil || de.IsDir() {
			return nil
		}
		if _, aerr := d.AddPath(p); aerr != nil {
			skipped++
		} else {
			added++
		}
		return nil
	})
	return added, skipped, err
}

// Remove deletes a path. Returns rows removed.
func (d *DB) Remove(filePath string) (int64, error) {
	abs := filePath
	if a, err := filepath.Abs(filePath); err == nil {
		abs = a
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	res, err := d.sql.Exec(`DELETE FROM imported_files WHERE path = ?`, abs)
	if err != nil {
		return 0, err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		// Legacy stored resolved paths; retry with evaluation of symlinks.
		if eval, eerr := filepath.EvalSymlinks(abs); eerr == nil && eval != abs {
			res, err = d.sql.Exec(`DELETE FROM imported_files WHERE path = ?`, eval)
			if err != nil {
				return 0, err
			}
			n, _ = res.RowsAffected()
		}
	}
	return n, nil
}

// List returns all rows ordered by name.
func (d *DB) List() ([]Entry, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	rows, err := d.sql.Query(
		`SELECT id, path, name, COALESCE(size_bytes,0), COALESCE(imported_at,'')
		 FROM imported_files ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Entry
	for rows.Next() {
		var e Entry
		if err := rows.Scan(&e.ID, &e.Path, &e.Name, &e.SizeBytes, &e.ImportedAt); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

// Score mirrors the Python _score: exact → 1.0, substring boost →
// max(0.6, len(q)/len(n)), else bigram-Dice ≈ difflib ratio.
func Score(query, name string) float64 {
	q := strings.ToLower(strings.TrimSpace(query))
	n := strings.ToLower(name)
	if q == "" {
		return 0
	}
	if q == n {
		return 1.0
	}
	if strings.Contains(n, q) {
		boost := float64(len([]rune(q))) / float64(max(1, len([]rune(n))))
		if boost < 0.6 {
			boost = 0.6
		}
		return boost
	}
	return diceBigrams(q, n)
}

// FindRelated returns rows scoring ≥ cutoff, sorted best-first.
func (d *DB) FindRelated(query string, cutoff float64) ([]Entry, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return nil, nil
	}
	rows, err := d.List()
	if err != nil {
		return nil, err
	}
	var out []Entry
	for _, r := range rows {
		if s := Score(query, r.Name); s >= cutoff {
			r.Score = s
			out = append(out, r)
		}
	}
	sortByScore(out)
	return out, nil
}

// FindForRequest is FindRelated minus files that vanished from disk.
func (d *DB) FindForRequest(query string, cutoff float64) ([]Entry, error) {
	hits, err := d.FindRelated(query, cutoff)
	if err != nil {
		return nil, err
	}
	alive := hits[:0]
	for _, h := range hits {
		if st, err := os.Stat(h.Path); err == nil && !st.IsDir() {
			alive = append(alive, h)
		}
	}
	return alive, nil
}

// MigrateLegacy copies rows from a Python-era imports.db when this DB's
// catalog is empty. The old file is left untouched.
func (d *DB) MigrateLegacy(legacyPath string) (int64, error) {
	if _, err := os.Stat(legacyPath); err != nil {
		return 0, nil
	}
	var count int64
	if err := d.sql.QueryRow(`SELECT COUNT(*) FROM imported_files`).Scan(&count); err != nil {
		return 0, err
	}
	if count > 0 {
		return 0, nil
	}
	legacy, err := sql.Open("sqlite", legacyPath)
	if err != nil {
		return 0, err
	}
	defer legacy.Close()
	rows, err := legacy.Query(
		`SELECT path, name, COALESCE(size_bytes,0), COALESCE(imported_at, datetime('now'))
		 FROM imported_files`)
	if err != nil {
		return 0, nil // old DB without the table → nothing to do
	}
	defer rows.Close()
	d.mu.Lock()
	defer d.mu.Unlock()
	var copied int64
	for rows.Next() {
		var path, name string
		var size int64
		var importedAt string
		if err := rows.Scan(&path, &name, &size, &importedAt); err != nil {
			continue
		}
		if _, err := d.sql.Exec(
			`INSERT OR IGNORE INTO imported_files(path, name, size_bytes, imported_at)
			 VALUES (?,?,?,?)`, path, name, size, importedAt); err == nil {
			copied++
		}
	}
	return copied, nil
}

// SetSetting upserts a settings kv pair.
func (d *DB) SetSetting(key, value string) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	_, err := d.sql.Exec(`INSERT OR REPLACE INTO settings(key, value) VALUES (?,?)`, key, value)
	return err
}

// GetSetting reads a settings value ("", nil when absent).
func (d *DB) GetSetting(key string) (string, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	var v string
	err := d.sql.QueryRow(`SELECT value FROM settings WHERE key = ?`, key).Scan(&v)
	if err == sql.ErrNoRows {
		return "", nil
	}
	return v, err
}

// --- similarity internals ---

func sortByScore(in []Entry) {
	for i := 1; i < len(in); i++ {
		for j := i; j > 0 && in[j].Score > in[j-1].Score; j-- {
			in[j], in[j-1] = in[j-1], in[j]
		}
	}
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// diceBigrams is the Sørensen–Dice coefficient over character bigrams,
// a close, dependency-free stand-in for difflib.SequenceMatcher.ratio.
func diceBigrams(a, b string) float64 {
	ar, br := []rune(a), []rune(b)
	if len(ar) == 0 || len(br) == 0 {
		return 0
	}
	if len(ar) == 1 && len(br) == 1 {
		if ar[0] == br[0] {
			return 1
		}
		return 0
	}
	counts := make(map[string]int, len(ar))
	for i := 0; i+1 < len(ar); i++ {
		counts[string(ar[i:i+2])]++
	}
	if len(counts) == 0 {
		if strings.Contains(b, a) || strings.Contains(a, b) {
			return 0.6
		}
		return 0
	}
	var inter int
	for i := 0; i+1 < len(br); i++ {
		bg := string(br[i : i+2])
		if counts[bg] > 0 {
			counts[bg]--
			inter++
		}
	}
	den := (len(ar) - 1) + (len(br) - 1)
	if den == 0 {
		return 0
	}
	return 2 * float64(inter) / float64(den)
}
