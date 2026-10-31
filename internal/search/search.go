// Package search implements filesystem search in pure Go, replacing both
// libusersearch (user-storage walk) and libfilesearch (exact search inside
// the app folders). No DLLs, no Python fallback — one implementation.
package search

import (
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
)

// Hit is one search result.
type Hit struct {
	Name string `json:"name"`
	Path string `json:"path"`
}

// skipSegments mirrors the legacy _SKIP_PARTS block-list. Matching is done
// per path segment (exact, case-insensitive) rather than whole-path
// substring, so folders like "my-windows-themes" are NOT skipped while
// real system folders still are.
var skipSegments = map[string]bool{
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

// SkipRel reports whether a root-relative path must be excluded.
func SkipRel(rel string) bool {
	parts := strings.FieldsFunc(strings.ToLower(rel), func(r rune) bool {
		return r == '/' || r == '\\' || r == os.PathSeparator
	})
	for _, p := range parts {
		if skipSegments[p] {
			return true
		}
	}
	return false
}

// SkipPath reports whether a full path must be excluded from user search.
func SkipPath(path string) bool { return SkipRel(path) }

// UserRoots returns the folders user search walks: well-known profile
// folders plus every extra ready drive (D:–Z: on Windows).
func UserRoots() []string {
	var roots []string
	home, err := os.UserHomeDir()
	if err == nil {
		for _, name := range []string{
			"Documents", "Downloads", "Desktop", "Pictures", "Videos", "Music",
		} {
			p := filepath.Join(home, name)
			if st, err := os.Stat(p); err == nil && st.IsDir() {
				roots = append(roots, p)
			}
		}
	}
	if runtime.GOOS == "windows" {
		for _, drive := range readyDrives() {
			roots = append(roots, drive)
		}
	} else {
		for _, extra := range []string{"/mnt", "/media"} {
			if st, err := os.Stat(extra); err == nil && st.IsDir() {
				roots = append(roots, extra)
			}
		}
	}
	return roots
}

func readyDrives() []string {
	var out []string
	for c := 'D'; c <= 'Z'; c++ {
		drive := string([]rune{c, ':', '\\'})
		// Skip the drive(s) already covered via the profile folders?
		// No — legacy also walked whole extra drives, keep parity.
		if st, err := os.Stat(drive); err == nil && st.IsDir() {
			out = append(out, drive)
		}
	}
	return out
}

// SearchUser finds files/folders whose name contains query (case-insensitive).
// A matching folder contributes the files directly inside it (legacy parity).
func SearchUser(query string, limit int) ([]Hit, error) {
	return SearchRoots(UserRoots(), query, limit)
}

// SearchRoots is SearchUser over explicit roots (used by tests + app search).
func SearchRoots(roots []string, query string, limit int) ([]Hit, error) {
	query = strings.ToLower(strings.TrimSpace(query))
	if query == "" || limit <= 0 {
		return nil, nil
	}
	var hits []Hit
	push := func(name, path string) bool {
		hits = append(hits, Hit{Name: name, Path: path})
		return len(hits) >= limit
	}
	for _, root := range roots {
		if len(hits) >= limit {
			break
		}
		_ = filepath.WalkDir(root, func(p string, d os.DirEntry, err error) error {
			if len(hits) >= limit {
				return filepath.SkipAll
			}
			if err != nil {
				return nil // skip unreadable (parity with Python OSError skip)
			}
			if p == root {
				return nil
			}
			// Skip checks run on the root-relative path so the root itself
			// is never excluded (e.g. test temp dirs under %AppData%).
			rel, rerr := filepath.Rel(root, p)
			if rerr != nil {
				return nil
			}
			if SkipRel(rel) {
				if d.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}
			if d.IsDir() {
				if strings.Contains(strings.ToLower(d.Name()), query) {
					// Folder hit: list its files (one level, like the fallback).
					entries, rerr := os.ReadDir(p)
					if rerr != nil {
						return nil
					}
					for _, e := range entries {
						if e.IsDir() || SkipRel(filepath.Join(rel, e.Name())) {
							continue
						}
						if push(e.Name(), filepath.Join(p, e.Name())) {
							return filepath.SkipAll
						}
					}
				}
				return nil
			}
			if strings.Contains(strings.ToLower(d.Name()), query) {
				if push(d.Name(), p) {
					return filepath.SkipAll
				}
			}
			return nil
		})
	}
	sort.Slice(hits, func(i, j int) bool {
		if hits[i].Name != hits[j].Name {
			return hits[i].Name < hits[j].Name
		}
		return hits[i].Path < hits[j].Path
	})
	return hits, nil
}

// SearchAppFolders is exact-name lookup inside shared/ + received/
// (replaces libfilesearch). Returns absolute paths, sorted.
func SearchAppFolders(query string, folders []string) []string {
	query = strings.ToLower(strings.TrimSpace(query))
	if query == "" {
		return nil
	}
	var out []string
	for _, folder := range folders {
		_ = filepath.WalkDir(folder, func(p string, d os.DirEntry, err error) error {
			if err != nil || d.IsDir() {
				return nil
			}
			if strings.ToLower(d.Name()) == query {
				out = append(out, p)
			}
			return nil
		})
	}
	sort.Strings(out)
	return out
}
