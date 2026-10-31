// Package index owns the searchable copy of the local filesystem: one row per
// file in the shared SQLite database (table file_index, created by catalog
// migration 2).
//
// It is a *cache*, never a source of truth: the walker is background,
// cancellable and yields between directories so a low-end PC stays
// responsive, and every path it stores is re-checked before use. System
// locations are excluded with the same guard the live search uses
// (search.SkipPath/skipSegments) plus a user-editable extra list.
package index

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"
)

// DefaultMaxRows caps the index (a 500k-row cap ≈ tens of MB of SQLite and
// keeps the SQL prefilter fast on a Celeron).
const DefaultMaxRows = 500000

// batchSize is how many rows go into one transaction.
const batchSize = 400

// Entry is one indexed file.
type Entry struct {
	Path  string `json:"path"`
	Name  string `json:"name"`
	Dir   string `json:"dir"`
	Ext   string `json:"ext"`
	Size  int64  `json:"size"`
	MTime int64  `json:"mtime"`
}

// State is what the Settings/Find screens show.
type State struct {
	Count    int     `json:"count"`
	BuiltAt  int64   `json:"builtAt"`  // unix seconds, 0 = never built
	Building bool    `json:"building"` // walker running
	Progress float64 `json:"progress"` // 0..1 while building
	Note     string  `json:"note,omitempty"`
}

// Options configures a walk.
type Options struct {
	// Roots are the folders to walk (setting index_roots).
	Roots []string
	// Skip is the extra segment block-list (setting index_skip): ".git",
	// "node_modules", hidden dot-folders are added automatically.
	Skip []string
	// MaxRows caps the table (DefaultMaxRows).
	MaxRows int
	// Sources returns extra paths to index unconditionally: the data root's
	// received/ + shared/ plus every catalog entry. Called once per walk.
	Sources func() []string
}

// Indexer is the walker + query front end. Safe for concurrent use.
type Indexer struct {
	db *sql.DB

	mu       sync.Mutex
	building bool
	progress float64
	cancel   context.CancelFunc
	note     string
}

// New wraps a shared database handle (catalog.DB.SQL()).
func New(db *sql.DB) *Indexer {
	return &Indexer{db: db}
}

// ---------- state ----------

// State reports count/builtAt/building/progress.
func (ix *Indexer) State() State {
	ix.mu.Lock()
	building, progress, note := ix.building, ix.progress, ix.note
	ix.mu.Unlock()
	st := State{Building: building, Progress: progress, Note: note}
	if n, err := ix.Count(); err == nil {
		st.Count = n
	}
	st.BuiltAt = ix.metaInt("built_at")
	return st
}

// Count returns the number of indexed rows.
func (ix *Indexer) Count() (int, error) {
	var n int
	err := ix.db.QueryRow(`SELECT COUNT(*) FROM file_index`).Scan(&n)
	return n, err
}

// BuiltAt returns the last successful walk time (zero when never).
func (ix *Indexer) BuiltAt() time.Time {
	ts := ix.metaInt("built_at")
	if ts == 0 {
		return time.Time{}
	}
	return time.Unix(ts, 0)
}

// NeedsBuild reports whether the index has never been built or is older than
// maxAge (the app rescans at most once a day, in the background).
func (ix *Indexer) NeedsBuild(maxAge time.Duration) bool {
	built := ix.BuiltAt()
	if built.IsZero() {
		return true
	}
	return time.Since(built) > maxAge
}

func (ix *Indexer) metaInt(key string) int64 {
	var v string
	if err := ix.db.QueryRow(`SELECT value FROM index_meta WHERE key = ?`, key).Scan(&v); err != nil {
		return 0
	}
	var n int64
	_, _ = fmt.Sscanf(v, "%d", &n)
	return n
}

func (ix *Indexer) setMeta(key, value string) error {
	_, err := ix.db.Exec(`INSERT OR REPLACE INTO index_meta(key, value) VALUES (?,?)`, key, value)
	return err
}

// ---------- walk ----------

// Rebuild clears the table, then walks every configured source.
func (ix *Indexer) Rebuild(ctx context.Context, opts Options) error {
	if _, err := ix.db.Exec(`DELETE FROM file_index`); err != nil {
		return fmt.Errorf("index: clear: %w", err)
	}
	_ = ix.setMeta("built_at", "0")
	return ix.Build(ctx, opts)
}

// Build walks the roots and refreshes their rows. Cancelling ctx leaves the
// rows written so far in place (they are all valid) and does not stamp
// built_at, so the next run tries again.
func (ix *Indexer) Build(ctx context.Context, opts Options) error {
	if opts.MaxRows <= 0 {
		opts.MaxRows = DefaultMaxRows
	}
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	ix.mu.Lock()
	if ix.building {
		ix.mu.Unlock()
		return fmt.Errorf("index: a walk is already running")
	}
	ix.building = true
	ix.progress = 0
	ix.note = ""
	ix.cancel = cancel
	ix.mu.Unlock()
	defer func() {
		ix.mu.Lock()
		ix.building = false
		ix.cancel = nil
		ix.mu.Unlock()
	}()

	skip := buildSkip(opts.Skip)
	buf := make([]Entry, 0, batchSize)
	flushed := 0
	capped := false
	ct, _ := ix.Count()

	flush := func(force bool) error {
		if len(buf) == 0 || (!force && len(buf) < batchSize) {
			return nil
		}
		if err := ix.insertBatch(buf); err != nil {
			return err
		}
		flushed += len(buf)
		buf = buf[:0]
		return nil
	}

	roots := opts.Roots
	total := len(roots) + 1 // +1 for the extra sources
	done := 0
	for _, root := range roots {
		if err := ctx.Err(); err != nil {
			return err
		}
		walkErr := filepath.WalkDir(root, func(p string, d os.DirEntry, err error) error {
			if err != nil {
				return nil // unreadable entry: skip, like the live search
			}
			if cerr := ctx.Err(); cerr != nil {
				return cerr
			}
			rel, rerr := filepath.Rel(root, p)
			if rerr != nil {
				return nil
			}
			if rel == "." {
				return nil
			}
			if skip(rel, d.IsDir()) {
				if d.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}
			if d.IsDir() {
				// Yield between directories: never hog the disk queue.
				runtime.Gosched()
				return nil
			}
			if ct+flushed+len(buf) >= opts.MaxRows {
				capped = true
				return filepath.SkipAll
			}
			info, ierr := d.Info()
			if ierr != nil {
				return nil
			}
			buf = append(buf, entryFor(p, info))
			return flush(false)
		})
		if walkErr != nil && walkErr != context.Canceled {
			ix.setNote(fmt.Sprintf("could not read %s: %v", root, walkErr))
		}
		done++
		ix.setProgress(float64(done) / float64(total))
		if capped {
			break
		}
	}

	// Extra sources: received/ + shared/ + every catalog entry. These bypass
	// the system-path guard (the data folder may live under a skipped name
	// such as .../WayerPC/...) but still honour the extra skip list.
	if !capped && opts.Sources != nil {
		for _, src := range opts.Sources() {
			if err := ctx.Err(); err != nil {
				return err
			}
			st, err := os.Stat(src)
			if err != nil {
				_ = ix.Remove(src) // vanished: drop the stale row
				continue
			}
			if !st.IsDir() {
				if ct+flushed+len(buf) >= opts.MaxRows {
					capped = true
					break
				}
				buf = append(buf, entryFor(src, st))
				if err := flush(false); err != nil {
					return err
				}
				continue
			}
			walkErr := filepath.WalkDir(src, func(p string, d os.DirEntry, err error) error {
				if err != nil {
					return nil
				}
				if cerr := ctx.Err(); cerr != nil {
					return cerr
				}
				rel, rerr := filepath.Rel(src, p)
				if rerr != nil || rel == "." {
					return nil
				}
				if skip(rel, d.IsDir()) && !strings.HasPrefix(rel, ".partial") {
					if d.IsDir() {
						return filepath.SkipDir
					}
					return nil
				}
				if d.IsDir() {
					runtime.Gosched()
					return nil
				}
				if ct+flushed+len(buf) >= opts.MaxRows {
					capped = true
					return filepath.SkipAll
				}
				info, ierr := d.Info()
				if ierr != nil {
					return nil
				}
				buf = append(buf, entryFor(p, info))
				return flush(false)
			})
			if walkErr != nil && walkErr != context.Canceled {
				ix.setNote(fmt.Sprintf("could not read %s: %v", src, walkErr))
			}
		}
	}

	if err := flush(true); err != nil {
		return err
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if capped {
		ix.setNote(fmt.Sprintf("index capped at %d files", opts.MaxRows))
	}
	_ = ix.setMeta("built_at", itoa64(time.Now().Unix()))
	ix.setProgress(1)
	return nil
}

// Cancel stops a running walk (RebuildIndex while building).
func (ix *Indexer) Cancel() {
	ix.mu.Lock()
	cancel := ix.cancel
	ix.mu.Unlock()
	if cancel != nil {
		cancel()
	}
}

func (ix *Indexer) setProgress(p float64) {
	ix.mu.Lock()
	ix.progress = p
	ix.mu.Unlock()
}

func (ix *Indexer) setNote(note string) {
	ix.mu.Lock()
	if ix.note == "" {
		ix.note = note
	}
	ix.mu.Unlock()
}

// ---------- single-path maintenance ----------

// Add indexes (or refreshes) exactly one file. Missing paths are removed.
func (ix *Indexer) Add(path string) error {
	st, err := os.Stat(path)
	if err != nil || st.IsDir() {
		return ix.Remove(path)
	}
	return ix.insertBatch([]Entry{entryFor(path, st)})
}

// AddTree indexes a folder tree (used right after an import).
func (ix *Indexer) AddTree(root string, skip []string) error {
	skipFn := buildSkip(skip)
	var buf []Entry
	err := filepath.WalkDir(root, func(p string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		rel, rerr := filepath.Rel(root, p)
		if rerr != nil || rel == "." {
			return nil
		}
		if skipFn(rel, d.IsDir()) {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if d.IsDir() {
			runtime.Gosched()
			return nil
		}
		info, ierr := d.Info()
		if ierr != nil {
			return nil
		}
		buf = append(buf, entryFor(p, info))
		if len(buf) >= batchSize {
			if err := ix.insertBatch(buf); err != nil {
				return err
			}
			buf = buf[:0]
		}
		return nil
	})
	if err != nil {
		return err
	}
	if len(buf) > 0 {
		return ix.insertBatch(buf)
	}
	return nil
}

// Remove deletes one path.
func (ix *Indexer) Remove(path string) error {
	_, err := ix.db.Exec(`DELETE FROM file_index WHERE path = ?`, path)
	return err
}

// RemoveUnder deletes every row below dir (catalog entry removed).
func (ix *Indexer) RemoveUnder(dir string) error {
	prefix := strings.TrimRight(dir, `/\`) + string(os.PathSeparator) + "%"
	_, err := ix.db.Exec(`DELETE FROM file_index WHERE path = ? OR path LIKE ?`, dir, prefix)
	return err
}

func (ix *Indexer) insertBatch(entries []Entry) error {
	if len(entries) == 0 {
		return nil
	}
	tx, err := ix.db.Begin()
	if err != nil {
		return err
	}
	stmt, err := tx.Prepare(`INSERT OR REPLACE INTO file_index(path, name, name_lower, dir, ext, size, mtime)
		VALUES (?,?,?,?,?,?,?)`)
	if err != nil {
		_ = tx.Rollback()
		return err
	}
	defer stmt.Close()
	for _, e := range entries {
		if _, err := stmt.Exec(e.Path, e.Name, strings.ToLower(e.Name), e.Dir, e.Ext, e.Size, e.MTime); err != nil {
			_ = tx.Rollback()
			return err
		}
	}
	return tx.Commit()
}

// ---------- query ----------

// Candidates returns rows whose name or folder contains every token (SQL
// prefilter with LIKE), newest first. The bridge ranks them in Go.
func (ix *Indexer) Candidates(tokens []string, limit int) ([]Entry, error) {
	if limit <= 0 || limit > 5000 {
		limit = 2000
	}
	var where []string
	var args []any
	for _, tok := range tokens {
		tok = strings.TrimSpace(strings.ToLower(tok))
		if tok == "" {
			continue
		}
		if strings.ContainsAny(tok, "%_") {
			continue // wildcards from the user would widen the scan
		}
		where = append(where, `(name_lower LIKE ? OR dir LIKE ?)`)
		args = append(args, "%"+tok+"%", "%"+tok+"%")
	}
	if len(where) == 0 {
		return nil, nil
	}
	query := `SELECT path, name, dir, ext, size, mtime FROM file_index WHERE ` +
		strings.Join(where, " AND ") + ` ORDER BY mtime DESC LIMIT ?`
	args = append(args, limit)
	rows, err := ix.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]Entry, 0, 64)
	for rows.Next() {
		var e Entry
		if err := rows.Scan(&e.Path, &e.Name, &e.Dir, &e.Ext, &e.Size, &e.MTime); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

// ---------- helpers ----------

func entryFor(path string, info os.FileInfo) Entry {
	dir := filepath.Dir(path)
	return Entry{
		Path:  path,
		Name:  info.Name(),
		Dir:   dir,
		Ext:   strings.TrimPrefix(strings.ToLower(filepath.Ext(path)), "."),
		Size:  info.Size(),
		MTime: info.ModTime().Unix(),
	}
}

// buildSkip combines the user list with the defaults (dot-folders, VCS,
// dependency folders) and the system guard. A segment matches exactly
// (case-insensitive) so "my.git" is not treated as ".git".
func buildSkip(extra []string) func(rel string, isDir bool) bool {
	segments := map[string]bool{
		".git": true, "node_modules": true,
	}
	for _, s := range extra {
		s = strings.ToLower(strings.TrimSpace(s))
		if s == "" {
			continue
		}
		segments[s] = true
	}
	return func(rel string, isDir bool) bool {
		parts := strings.FieldsFunc(strings.ToLower(rel), func(r rune) bool {
			return r == '/' || r == '\\'
		})
		for _, p := range parts {
			if p == "" || p == "." || p == ".." {
				continue
			}
			if segments[p] {
				return true
			}
			// Hidden dot-folders are always skipped (cache/IDE noise), but a
			// dot-*file* (".env.local") stays visible.
			if isDir && strings.HasPrefix(p, ".") && p != ".partial" {
				return true
			}
		}
		return skipSystem(parts)
	}
}

// skipSystem applies search's segment block-list to already-split segments.
func skipSystem(parts []string) bool {
	for _, p := range parts {
		if systemSegments[p] {
			return true
		}
	}
	return false
}

// systemSegments mirrors search.skipSegments (kept in sync by the test in
// index_test.go, which asserts both lists reject the same paths).
var systemSegments = map[string]bool{
	"windows":                   true,
	"program files":             true,
	"program files (x86)":       true,
	"programdata":               true,
	"appdata":                   true,
	"$recycle.bin":              true,
	"system volume information": true,
	"windowsapps":               true,
	"wayerpc":                   true,
	"proc":                      true,
	"sys":                       true,
	"dev":                       true,
}

func itoa64(n int64) string {
	if n == 0 {
		return "0"
	}
	var buf [20]byte
	i := len(buf)
	neg := n < 0
	if neg {
		n = -n
	}
	for n > 0 && i > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	if neg && i > 0 {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}
