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

	"wayerpc/internal/textmatch"
)

// DefaultCutoff mirrors find_related(query, cutoff=0.2): ≥20% similarity.
const DefaultCutoff = 0.2

// schemaV1 is migration 1: the import catalog + settings kv. Kept
// byte-compatible with the Python-era imports.db (see MigrateLegacy).
const schemaV1 = `
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

// schemaV2 is migration 2: the local file index (internal/index) plus its
// bookkeeping table. Paths are absolute; name_lower is pre-folded because
// every lookup lowercases the query anyway.
const schemaV2 = `
CREATE TABLE IF NOT EXISTS file_index (
  path TEXT PRIMARY KEY,
  name TEXT NOT NULL,
  name_lower TEXT NOT NULL,
  dir TEXT NOT NULL,
  ext TEXT NOT NULL,
  size INTEGER NOT NULL DEFAULT 0,
  mtime INTEGER NOT NULL DEFAULT 0
);
CREATE INDEX IF NOT EXISTS idx_file_index_name_lower ON file_index(name_lower);
CREATE INDEX IF NOT EXISTS idx_file_index_mtime ON file_index(mtime);
CREATE TABLE IF NOT EXISTS index_meta (
  key TEXT PRIMARY KEY,
  value TEXT NOT NULL
);
`

// schemaV3 is migration 3: protocol-v2 bookkeeping — the device registry and
// the lazy sha256 cache (never hash whole drives; one row per file).
const schemaV3 = `
CREATE TABLE IF NOT EXISTS devices (
  did TEXT PRIMARY KEY,
  name TEXT NOT NULL DEFAULT '',
  last_seen INTEGER NOT NULL DEFAULT 0,
  last_ip TEXT NOT NULL DEFAULT ''
);
CREATE TABLE IF NOT EXISTS hash_cache (
  path TEXT PRIMARY KEY,
  size INTEGER NOT NULL,
  mtime INTEGER NOT NULL,
  sha256 TEXT NOT NULL
);
`

// migrations is append-only: index i upgrades the database from schema
// version i to i+1. The live version lives in PRAGMA user_version, so
// upgrading an existing install only runs the statements it is missing.
var migrations = []string{schemaV1, schemaV2, schemaV3}

// SchemaVersion is the version this build expects after Open.
var SchemaVersion = len(migrations)

// Entry is one catalog row.
type Entry struct {
	ID         int64   `json:"id"`
	Path       string  `json:"path"`
	Name       string  `json:"name"`
	SizeBytes  int64   `json:"sizeBytes"`
	ImportedAt string  `json:"importedAt"`
	Score      float64 `json:"score,omitempty"`
	// Exists is filled by callers that stat the path (List leaves it false).
	Exists bool `json:"exists,omitempty"`
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
		// busy_timeout: the UI thread and the phone link write concurrently
		// (imports vs uploads); wait up to 5 s for the writer lock instead
		// of failing the statement with SQLITE_BUSY.
		"PRAGMA busy_timeout=5000;",
	} {
		if _, err := sqlDB.Exec(pragma); err != nil {
			sqlDB.Close()
			return nil, fmt.Errorf("catalog: pragma: %w", err)
		}
	}
	d := &DB{sql: sqlDB, path: dbPath}
	if err := d.migrate(); err != nil {
		sqlDB.Close()
		return nil, err
	}
	if appVersion != "" {
		_ = d.SetSetting("app_version", appVersion)
	}
	return d, nil
}

// migrate applies the pending schema migrations using PRAGMA user_version.
// A pre-user_version database reads back 0, so it gets every migration in
// order (all statements are idempotent: CREATE ... IF NOT EXISTS).
func (d *DB) migrate() error {
	var current int
	if err := d.sql.QueryRow(`PRAGMA user_version`).Scan(&current); err != nil {
		return fmt.Errorf("catalog: read user_version: %w", err)
	}
	if current > len(migrations) {
		return fmt.Errorf("catalog: database schema %d is newer than this build (%d) — update WayerPC", current, len(migrations))
	}
	for v := current; v < len(migrations); v++ {
		tx, err := d.sql.Begin()
		if err != nil {
			return fmt.Errorf("catalog: begin migration %d: %w", v+1, err)
		}
		if _, err := tx.Exec(migrations[v]); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("catalog: migration %d: %w", v+1, err)
		}
		if _, err := tx.Exec(fmt.Sprintf(`PRAGMA user_version=%d`, v+1)); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("catalog: stamp migration %d: %w", v+1, err)
		}
		if err := tx.Commit(); err != nil {
			return fmt.Errorf("catalog: commit migration %d: %w", v+1, err)
		}
	}
	return nil
}

// SchemaVersion reports the live PRAGMA user_version (tests + migrations).
func (d *DB) SchemaVersionNow() (int, error) {
	var v int
	err := d.sql.QueryRow(`PRAGMA user_version`).Scan(&v)
	return v, err
}

// SQL exposes the shared handle for packages that own their own tables in
// the same database (internal/index). Callers must not hold catalog locks.
func (d *DB) SQL() *sql.DB { return d.sql }

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

// ---------- protocol v2: hash cache + device registry ----------

// CachedHash returns the sha256 stored for path when the file's size and
// mtime still match (otherwise "" and false — the caller hashes for real).
func (d *DB) CachedHash(path string, size, mtime int64) (string, bool) {
	d.mu.Lock()
	defer d.mu.Unlock()
	var sha string
	var gotSize, gotMTime int64
	err := d.sql.QueryRow(`SELECT sha256, size, mtime FROM hash_cache WHERE path = ?`, path).
		Scan(&sha, &gotSize, &gotMTime)
	if err != nil || gotSize != size || gotMTime != mtime {
		return "", false
	}
	return sha, true
}

// PutHash remembers a file's sha256 (cheap de-duplication of hashing work).
func (d *DB) PutHash(path, sha string, size, mtime int64) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	_, err := d.sql.Exec(
		`INSERT OR REPLACE INTO hash_cache(path, size, mtime, sha256) VALUES (?,?,?,?)`,
		path, size, mtime, sha)
	return err
}

// ForgetHash drops one cache row (catalog removal / deleted file).
func (d *DB) ForgetHash(path string) {
	d.mu.Lock()
	defer d.mu.Unlock()
	_, _ = d.sql.Exec(`DELETE FROM hash_cache WHERE path = ?`, path)
}

// Device is one phone known to this PC.
type Device struct {
	DID      string `json:"did"`
	Name     string `json:"name"`
	LastSeen int64  `json:"lastSeen"`
	LastIP   string `json:"lastIp"`
}

// UpsertDevice records a phone handshake (called on every hello).
func (d *DB) UpsertDevice(did, name, ip string, seen int64) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	_, err := d.sql.Exec(
		`INSERT INTO devices(did, name, last_seen, last_ip) VALUES (?,?,?,?)
		 ON CONFLICT(did) DO UPDATE SET
		   name = CASE WHEN excluded.name <> '' THEN excluded.name ELSE devices.name END,
		   last_seen = excluded.last_seen,
		   last_ip = excluded.last_ip`,
		did, name, seen, ip)
	return err
}

// Devices lists known phones, most recently seen first.
func (d *DB) Devices() ([]Device, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	rows, err := d.sql.Query(`SELECT did, name, last_seen, last_ip FROM devices ORDER BY last_seen DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Device
	for rows.Next() {
		var dev Device
		if err := rows.Scan(&dev.DID, &dev.Name, &dev.LastSeen, &dev.LastIP); err != nil {
			return nil, err
		}
		out = append(out, dev)
	}
	return out, rows.Err()
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

// diceBigrams is the Sørensen–Dice coefficient over character bigrams
// (see internal/textmatch — the rank engine measures similarity the same way).
func diceBigrams(a, b string) float64 {
	return textmatch.Dice(a, b)
}
