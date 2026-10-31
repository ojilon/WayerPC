// Package storage owns the on-disk data-root layout:
//
//	<dataRoot>/
//	├── shared/            # optional drop folder, also served by /ask
//	├── received/          # phone uploads land here
//	└── Data/
//	    └── wayerpc.db     # SQLite catalog (see internal/catalog)
//
// It replaces backend/manage_storage/*.py. Imported items are paths, not
// copies — nothing is ever duplicated into shared/.
package storage

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"time"
)

const (
	SharedDirName   = "shared"
	ReceivedDirName = "received"
	DataDirName     = "Data"
	DBFileName      = "wayerpc.db"
	// LegacyDBFileName is the Python-era database we migrate from.
	LegacyDBFileName = "imports.db"
)

// Store is a rooted handle to a data root.
type Store struct{ root string }

// Init creates the tree and returns a Store rooted at dataRoot.
func Init(dataRoot string) (*Store, error) {
	if dataRoot == "" {
		return nil, fmt.Errorf("storage: empty data root")
	}
	s := &Store{root: dataRoot}
	for _, d := range []string{s.SharedDir(), s.ReceivedDir(), s.DataDir()} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			return nil, fmt.Errorf("storage: mkdir %s: %w", d, err)
		}
	}
	return s, nil
}

// Root returns the data root.
func (s *Store) Root() string { return s.root }

// SharedDir is the optional drop folder.
func (s *Store) SharedDir() string { return filepath.Join(s.root, SharedDirName) }

// ReceivedDir holds phone uploads.
func (s *Store) ReceivedDir() string { return filepath.Join(s.root, ReceivedDirName) }

// DataDir holds the SQLite database.
func (s *Store) DataDir() string { return filepath.Join(s.root, DataDirName) }

// DbPath is the live database file.
func (s *Store) DbPath() string { return filepath.Join(s.DataDir(), DBFileName) }

// LegacyDbPath is the Python-era database (for one-time migration).
func (s *Store) LegacyDbPath() string { return filepath.Join(s.DataDir(), LegacyDBFileName) }

// FileInfo describes one file for the Files views.
type FileInfo struct {
	Name    string    `json:"name"`
	Path    string    `json:"path"`
	Size    int64     `json:"size"`
	ModTime time.Time `json:"modTime"`
	IsDir   bool      `json:"isDir"`
}

// ListFiles walks dir (non-recursive top level first, then recursive) and
// returns entries newest-first. Missing dir → empty list, nil error.
func (s *Store) ListFiles(dir string) ([]FileInfo, error) {
	st, err := os.Stat(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	if !st.IsDir() {
		return nil, fmt.Errorf("storage: not a directory: %s", dir)
	}
	var out []FileInfo
	err = filepath.WalkDir(dir, func(p string, d os.DirEntry, err error) error {
		if err != nil {
			return nil // skip unreadable entries (parity: Python skipped on OSError)
		}
		if p == dir {
			return nil
		}
		if d.IsDir() {
			out = append(out, FileInfo{Name: d.Name(), Path: p, IsDir: true})
			return nil
		}
		info, err := d.Info()
		if err != nil {
			return nil
		}
		out = append(out, FileInfo{
			Name: d.Name(), Path: p,
			Size: info.Size(), ModTime: info.ModTime(),
		})
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].IsDir != out[j].IsDir {
			return !out[i].IsDir // files before folders
		}
		return out[i].Name < out[j].Name
	})
	return out, nil
}

// DiskUsage returns total bytes under the data root.
func (s *Store) DiskUsage() (int64, error) {
	var total int64
	err := filepath.WalkDir(s.root, func(_ string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		if info, ierr := d.Info(); ierr == nil {
			total += info.Size()
		}
		return nil
	})
	return total, err
}

// Relocate moves the whole tree to newRoot (same-drive rename when possible,
// copy+verify otherwise) and re-roots the store.
func (s *Store) Relocate(newRoot string) error {
	if newRoot == "" {
		return fmt.Errorf("storage: empty destination")
	}
	if samePath(newRoot, s.root) {
		return nil
	}
	if err := os.MkdirAll(newRoot, 0o755); err != nil {
		return fmt.Errorf("storage: mkdir destination: %w", err)
	}
	if err := os.Rename(s.root, newRoot); err == nil {
		s.root = newRoot
		return nil
	}
	// Cross-drive: copy then swap.
	tmp := newRoot + ".relocate-tmp"
	_ = os.RemoveAll(tmp)
	if err := copyTree(s.root, tmp); err != nil {
		return fmt.Errorf("storage: copy: %w", err)
	}
	if err := os.RemoveAll(newRoot); err != nil {
		return fmt.Errorf("storage: clear destination: %w", err)
	}
	if err := os.Rename(tmp, newRoot); err != nil {
		return fmt.Errorf("storage: swap: %w", err)
	}
	s.root = newRoot
	return nil
}

func samePath(a, b string) bool {
	aa, errA := filepath.Abs(a)
	bb, errB := filepath.Abs(b)
	if errA != nil || errB != nil {
		return a == b
	}
	return aa == bb
}

func copyTree(src, dst string) error {
	return filepath.WalkDir(src, func(p string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, p)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		if d.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		return copyFile(p, target)
	})
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, in)
	return err
}

// RevealInExplorer opens the parent folder and selects path when possible.
func RevealInExplorer(path string) error {
	switch runtime.GOOS {
	case "windows":
		// explorer /select, works for files and folders.
		return exec.Command("explorer", "/select,", path).Start()
	case "darwin":
		return exec.Command("open", "-R", path).Start()
	default:
		dir := path
		if st, err := os.Stat(path); err == nil && !st.IsDir() {
			dir = filepath.Dir(path)
		}
		return exec.Command("xdg-open", dir).Start()
	}
}
