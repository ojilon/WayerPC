package main

// App is the Wails bridge: the ONLY path between the TS frontend and the
// pure-Go backend (internal/*). Async backend pushes use runtime.EventsEmit:
// "log" (string line), "stats" (Status payload), "catalog-changed" (no payload).

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"

	"wayerpc/internal/catalog"
	"wayerpc/internal/config"
	"wayerpc/internal/search"
	"wayerpc/internal/server"
	"wayerpc/internal/storage"
	"wayerpc/internal/version"
)

const maxLogLines = 2000

// VersionInfo is returned to Settings + titlebar.
type VersionInfo struct {
	AppName         string `json:"appName"`
	Version         string `json:"version"`
	ProtocolVersion int    `json:"protocolVersion"`
	Description     string `json:"description"`
}

// Status is the Console-tab payload.
type Status struct {
	Server      server.Stats `json:"server"`
	DataRoot    string       `json:"dataRoot"`
	Shared      string       `json:"shared"`
	Received    string       `json:"received"`
	Onboarded   bool         `json:"onboarded"`
	DiskUsage   int64        `json:"diskUsage"`
	CatalogSize int          `json:"catalogSize"`
}

// StorageInfo is the Settings-tab payload.
type StorageInfo struct {
	DataRoot   string `json:"dataRoot"`
	InstallDir string `json:"installDir"`
	Shared     string `json:"shared"`
	Received   string `json:"received"`
	DbPath     string `json:"dbPath"`
	DiskUsage  int64  `json:"diskUsage"`
	Onboarded  bool   `json:"onboarded"`
}

// ImportResult counts a bulk import.
type ImportResult struct {
	Added   int `json:"added"`
	Skipped int `json:"skipped"`
}

// App holds Wails context + backend handles.
type App struct {
	ctx     context.Context
	version version.Info
	cfg     config.Config

	mu    sync.Mutex
	store *storage.Store
	cat   *catalog.DB
	srv   *server.Server
	logs  []string

	statsStop chan struct{}
}

// NewApp builds the bridge (wiring happens in startup, where ctx exists).
func NewApp() *App {
	return &App{version: version.Current(), cfg: config.Defaults()}
}

// ---------- lifecycle ----------

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	a.emit("version", a.GetVersion())

	cfg, _ := config.Load()
	a.cfg = cfg

	root := config.ResolveDataRoot(cfg)
	if root == "" {
		root = config.DevDataStub() // fresh-clone runs use .data/
	}
	if root == "" {
		a.logf("No data folder yet — open Settings and choose where WayerPC should live.")
		a.emit("catalog-changed")
		return
	}
	if err := a.openRoot(root); err != nil {
		a.logf(fmt.Sprintf("Storage init failed for %s: %v", root, err))
		return
	}
	go a.statsLoop()
}

// openRoot wires store+catalog+server to a data root (startup + relocate).
func (a *App) openRoot(root string) error {
	store, err := storage.Init(root)
	if err != nil {
		return err
	}
	cat, err := catalog.Open(store.DbPath(), a.version.Version)
	if err != nil {
		return err
	}
	if n, _ := cat.MigrateLegacy(store.LegacyDbPath()); n > 0 {
		a.logf(fmt.Sprintf("Migrated %d catalog row(s) from the previous install.", n))
	}

	a.mu.Lock()
	if a.cat != nil {
		_ = a.cat.Close()
	}
	a.store = store
	a.cat = cat
	if a.srv == nil {
		a.srv = server.New(a.cfg.Host, a.cfg.Port, store, cat, a.logf)
	} else {
		a.srv.Attach(store, cat)
	}
	srv := a.srv
	a.mu.Unlock()

	a.cfg.DataRoot = root
	_ = config.Save(a.cfg)

	a.logf(fmt.Sprintf("Ready — %s", a.version.String()))
	a.logf(fmt.Sprintf("Data folder: %s", root))
	go func() {
		if err := srv.Start(); err != nil {
			a.logf(fmt.Sprintf("Server stopped: %v", err))
		}
	}()
	a.emit("catalog-changed")
	return nil
}

func (a *App) shutdown(_ context.Context) {
	if a.statsStop != nil {
		close(a.statsStop)
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.srv != nil {
		_ = a.srv.Stop()
	}
	if a.cat != nil {
		_ = a.cat.Close()
	}
}

func (a *App) statsLoop() {
	t := time.NewTicker(time.Second)
	defer t.Stop()
	for {
		select {
		case <-t.C:
			a.emit("stats", a.GetStatus())
		case <-a.statsStop:
			return
		}
	}
}

// ---------- logging ----------

func (a *App) logf(line string) {
	a.mu.Lock()
	a.logs = append(a.logs, line)
	if len(a.logs) > maxLogLines {
		a.logs = a.logs[len(a.logs)-maxLogLines:]
	}
	// A phone upload changes the catalog out-of-band — refresh Files views.
	needsCatalogRefresh := strings.Contains(line, "[UPLOAD OK]")
	a.mu.Unlock()
	a.emit("log", line)
	if needsCatalogRefresh {
		a.emit("catalog-changed")
	}
}

func (a *App) emit(name string, data ...any) {
	if a.ctx == nil {
		return
	}
	runtime.EventsEmit(a.ctx, name, data...)
}

// GetLogs returns the buffered log lines.
func (a *App) GetLogs() []string {
	a.mu.Lock()
	defer a.mu.Unlock()
	out := make([]string, len(a.logs))
	copy(out, a.logs)
	return out
}

// ClearLogs empties the buffer.
func (a *App) ClearLogs() {
	a.mu.Lock()
	a.logs = nil
	a.mu.Unlock()
}

// ---------- version / status ----------

// GetVersion returns the embedded version (from version.json).
func (a *App) GetVersion() VersionInfo {
	v := a.version
	return VersionInfo{AppName: v.AppName, Version: v.Version, ProtocolVersion: v.ProtocolVersion, Description: v.Description}
}

// GetStatus snapshots server stats + storage for the Console tab.
func (a *App) GetStatus() Status {
	a.mu.Lock()
	srv := a.srv
	store := a.store
	cat := a.cat
	a.mu.Unlock()

	st := Status{DataRoot: a.cfg.DataRoot}
	if srv != nil {
		st.Server = srv.Snapshot()
	}
	if store != nil {
		st.DataRoot = store.Root()
		st.Shared = store.SharedDir()
		st.Received = store.ReceivedDir()
		st.Onboarded = true
		st.DiskUsage, _ = store.DiskUsage()
	}
	if cat != nil {
		if rows, err := cat.List(); err == nil {
			st.CatalogSize = len(rows)
		}
	}
	return st
}

// StartServer starts the TCP listener (auto-started at boot; manual resume).
func (a *App) StartServer() string {
	a.mu.Lock()
	srv := a.srv
	a.mu.Unlock()
	if srv == nil {
		return "storage not initialized — choose a data folder in Settings first"
	}
	if srv.Snapshot().Running {
		return "already running"
	}
	go func() {
		if err := srv.Start(); err != nil {
			a.logf(fmt.Sprintf("Server error: %v", err))
		}
	}()
	return "starting"
}

// StopServer stops the TCP listener (connections drain).
func (a *App) StopServer() string {
	a.mu.Lock()
	srv := a.srv
	a.mu.Unlock()
	if srv == nil {
		return "not running"
	}
	_ = srv.Stop()
	return "stopped"
}

// ---------- search (Import tab) ----------

// SearchUser searches profile folders + extra drives (skips system paths).
func (a *App) SearchUser(query string, limit int) []search.Hit {
	if limit <= 0 || limit > 1000 {
		limit = 200
	}
	hits, _ := search.SearchUser(query, limit)
	if hits == nil {
		return []search.Hit{}
	}
	return hits
}

// SearchApp searches shared/ + received/ by exact filename.
func (a *App) SearchApp(query string) []search.Hit {
	a.mu.Lock()
	store := a.store
	a.mu.Unlock()
	if store == nil {
		return []search.Hit{}
	}
	var out []search.Hit
	for _, p := range search.SearchAppFolders(query, []string{store.SharedDir(), store.ReceivedDir()}) {
		out = append(out, search.Hit{Name: base(p), Path: p})
	}
	return out
}

// ---------- catalog + files ----------

// CatalogList returns all imported paths.
func (a *App) CatalogList() []catalog.Entry {
	a.mu.Lock()
	cat := a.cat
	a.mu.Unlock()
	if cat == nil {
		return []catalog.Entry{}
	}
	rows, err := cat.List()
	if err != nil || rows == nil {
		return []catalog.Entry{}
	}
	return rows
}

// CatalogAddPaths registers files/folders (folders expand recursively).
func (a *App) CatalogAddPaths(paths []string) (ImportResult, error) {
	a.mu.Lock()
	cat := a.cat
	a.mu.Unlock()
	if cat == nil {
		return ImportResult{}, fmt.Errorf("storage not initialized")
	}
	var res ImportResult
	for _, p := range paths {
		if isDir(p) {
			added, skipped, _ := cat.AddFolder(p)
			res.Added += added
			res.Skipped += skipped
			continue
		}
		if _, err := cat.AddPath(p); err != nil {
			res.Skipped++
		} else {
			res.Added++
		}
	}
	a.logf(fmt.Sprintf("[IMPORT] Cataloged %d path(s) (%d skipped)", res.Added, res.Skipped))
	a.emit("catalog-changed")
	return res, nil
}

// CatalogAddFolder registers a whole folder tree.
func (a *App) CatalogAddFolder(folder string) (ImportResult, error) {
	a.mu.Lock()
	cat := a.cat
	a.mu.Unlock()
	if cat == nil {
		return ImportResult{}, fmt.Errorf("storage not initialized")
	}
	added, skipped, err := cat.AddFolder(folder)
	res := ImportResult{Added: added, Skipped: skipped}
	if err != nil && added == 0 {
		return res, err
	}
	a.logf(fmt.Sprintf("[IMPORT] Cataloged folder %s: %d file(s)", folder, added))
	a.emit("catalog-changed")
	return res, nil
}

// CatalogRemove unregisters a path.
func (a *App) CatalogRemove(path string) error {
	a.mu.Lock()
	cat := a.cat
	a.mu.Unlock()
	if cat == nil {
		return fmt.Errorf("storage not initialized")
	}
	_, err := cat.Remove(path)
	if err == nil {
		a.logf(fmt.Sprintf("[CATALOG] Removed %s", path))
		a.emit("catalog-changed")
	}
	return err
}

// ListReceived lists received/ newest-first.
func (a *App) ListReceived() []storage.FileInfo {
	return a.listDir("received")
}

// ListShared lists shared/.
func (a *App) ListShared() []storage.FileInfo {
	return a.listDir("shared")
}

func (a *App) listDir(which string) []storage.FileInfo {
	a.mu.Lock()
	store := a.store
	a.mu.Unlock()
	if store == nil {
		return []storage.FileInfo{}
	}
	dir := store.ReceivedDir()
	if which == "shared" {
		dir = store.SharedDir()
	}
	files, _ := store.ListFiles(dir)
	if files == nil {
		return []storage.FileInfo{}
	}
	return files
}

// ---------- storage / settings ----------

// GetStorageInfo reports install + data locations.
func (a *App) GetStorageInfo() StorageInfo {
	a.mu.Lock()
	store := a.store
	a.mu.Unlock()
	info := StorageInfo{InstallDir: a.cfg.InstallDir, DataRoot: a.cfg.DataRoot}
	if store != nil {
		info.DataRoot = store.Root()
		info.Shared = store.SharedDir()
		info.Received = store.ReceivedDir()
		info.DbPath = store.DbPath()
		info.DiskUsage, _ = store.DiskUsage()
		info.Onboarded = true
	}
	return info
}

// SetDataRoot relocates the data tree (Settings onboarding + move).
// Existing data is MOVED when the old root exists; otherwise fresh init.
func (a *App) SetDataRoot(newRoot string) (StorageInfo, error) {
	newRoot = strings.TrimSpace(newRoot)
	if newRoot == "" {
		return a.GetStorageInfo(), fmt.Errorf("empty folder")
	}
	a.mu.Lock()
	old := a.store
	a.mu.Unlock()
	if old != nil && !sameDir(old.Root(), newRoot) {
		if err := old.Relocate(newRoot); err != nil {
			return a.GetStorageInfo(), err
		}
	}
	if err := a.openRoot(newRoot); err != nil {
		return a.GetStorageInfo(), err
	}
	return a.GetStorageInfo(), nil
}

// PickFolder opens a directory chooser (import + settings).
func (a *App) PickFolder(title string) string {
	if a.ctx == nil {
		return ""
	}
	path, err := runtime.OpenDirectoryDialog(a.ctx, runtime.OpenDialogOptions{Title: title})
	if err != nil {
		return ""
	}
	return path
}

// PickFiles opens a multi-file chooser (import).
func (a *App) PickFiles() []string {
	if a.ctx == nil {
		return nil
	}
	paths, err := runtime.OpenMultipleFilesDialog(a.ctx, runtime.OpenDialogOptions{Title: "Choose files to import"})
	if err != nil || paths == nil {
		return nil
	}
	return paths
}

// RevealInExplorer opens path in Explorer/Finder/files.
func (a *App) RevealInExplorer(path string) string {
	if err := storage.RevealInExplorer(path); err != nil {
		return err.Error()
	}
	return "ok"
}

// GetSetting reads a settings kv (library view, prefs).
func (a *App) GetSetting(key string) string {
	a.mu.Lock()
	cat := a.cat
	a.mu.Unlock()
	if cat == nil {
		return ""
	}
	v, _ := cat.GetSetting(key)
	return v
}

// SetSetting writes a settings kv.
func (a *App) SetSetting(key, value string) error {
	a.mu.Lock()
	cat := a.cat
	a.mu.Unlock()
	if cat == nil {
		return fmt.Errorf("storage not initialized")
	}
	return cat.SetSetting(key, value)
}
