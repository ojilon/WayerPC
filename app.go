package main

// App is the Wails bridge: the ONLY path between the TS frontend and the
// pure-Go backend (internal/*). Async backend pushes use runtime.EventsEmit:
// "log" (string line), "stats" (Status payload), "catalog-changed" (no payload),
// "transfer" (server.Transfer), "activity" (server.Activity).

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"

	"wayerpc/internal/catalog"
	"wayerpc/internal/config"
	"wayerpc/internal/index"
	"wayerpc/internal/netutil"
	"wayerpc/internal/rank"
	"wayerpc/internal/search"
	"wayerpc/internal/server"
	"wayerpc/internal/session"
	"wayerpc/internal/storage"
	"wayerpc/internal/version"
)

const (
	maxLogLines = 2000
	// diskUsageTTL bounds how often the data-folder tree is walked. The
	// stats tick runs every second; walking a growing received/ folder on
	// every tick is what made the console feel heavy on a low-end PC.
	diskUsageTTL = 30 * time.Second
)

// VersionInfo is returned to Settings + titlebar.
type VersionInfo struct {
	AppName         string `json:"appName"`
	Version         string `json:"version"`
	ProtocolVersion int    `json:"protocolVersion"`
	Description     string `json:"description"`
}

// Status is the Console-tab payload.
type Status struct {
	Server server.Stats `json:"server"`
	// LanAddr is the PC's private IPv4 ("192.168.1.7") — the address a
	// phone can actually dial. Empty when no interface has one yet.
	LanAddr     string `json:"lanAddr"`
	DataRoot    string `json:"dataRoot"`
	Shared      string `json:"shared"`
	Received    string `json:"received"`
	Onboarded   bool   `json:"onboarded"`
	DiskUsage   int64  `json:"diskUsage"`
	CatalogSize int    `json:"catalogSize"`
	// Protocol v2 link state (phase 3): the phone name + how many control
	// links are live. Empty/0 simply means "no v2 phone yet".
	ProtocolV2   bool   `json:"protocolV2"`
	Device       string `json:"device,omitempty"`
	ControlLinks int    `json:"controlLinks"`
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

// FindOpts tunes one Find request (0 values use the defaults).
type FindOpts struct {
	// Limit caps the number of hits (default 50).
	Limit int `json:"limit"`
	// Cutoff drops weak hits (default rank.DefaultCutoff = 0.2).
	Cutoff float64 `json:"cutoff"`
}

// FindHits is the batched `find-hits` event payload.
type FindHits struct {
	SearchID string     `json:"searchId"`
	Hits     []rank.Hit `json:"hits"`
}

// FindDone closes one search (`find-done`).
type FindDone struct {
	SearchID    string `json:"searchId"`
	Total       int    `json:"total"`
	TookMS      int64  `json:"tookMs"`
	SkippedNote string `json:"skippedNote,omitempty"`
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
	// activity is the structured ring buffer shown by the Console table.
	// It is kept alongside (not instead of) the raw log lines.
	activity []server.Activity

	statsStop chan struct{}

	// idx is the local file index (nil until a data root is open) and
	// findCancels tracks in-flight Find requests so the UI can abort them.
	idx         *index.Indexer
	indexCancel context.CancelFunc
	findSeq     int
	findCancels map[string]context.CancelFunc

	// sess is the protocol-v2 session manager (nil until the first v2 use).
	sess *session.Manager
	// syncSeq numbers Sync runs so stale results can be ignored by the UI.
	syncSeq int

	// diskUsage cache (see diskUsageTTL).
	diskUsage   int64
	diskUsageAt time.Time
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
	a.startStats()
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
	// Stop the previous index walk before swapping the database out.
	if a.indexCancel != nil {
		a.indexCancel()
		a.indexCancel = nil
	}
	if a.cat != nil {
		_ = a.cat.Close()
	}
	a.store = store
	a.cat = cat
	a.idx = index.New(cat.SQL())
	if a.srv == nil {
		a.srv = server.New(a.cfg.Host, a.cfg.Port, store, cat, a.logf)
	} else {
		a.srv.Attach(store, cat)
	}
	a.srv.SetObserver(a)
	srv := a.srv
	a.mu.Unlock()

	a.cfg.DataRoot = root
	_ = config.Save(a.cfg)

	// Protocol v2 is opt-in; v1 connections never reach this handler.
	if a.cfg.ProtocolV2 {
		srv.SetV2(a.sessionManager())
	} else {
		srv.SetV2(nil)
	}

	a.invalidateDiskUsage()
	a.startIndexWarmup()
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
	a.stopStats()
	a.mu.Lock()
	if a.indexCancel != nil {
		a.indexCancel()
		a.indexCancel = nil
	}
	a.mu.Unlock()
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.srv != nil {
		_ = a.srv.Stop()
	}
	if a.cat != nil {
		_ = a.cat.Close()
	}
}

// startStats (re)starts the 1 s status ticker, stopping any previous loop
// first — openRoot runs again on every data-root relocate.
func (a *App) startStats() {
	a.stopStats()
	stop := make(chan struct{})
	a.mu.Lock()
	a.statsStop = stop
	a.mu.Unlock()
	go a.statsLoop(stop)
}

// stopStats terminates the ticker loop started by startStats.
func (a *App) stopStats() {
	a.mu.Lock()
	stop := a.statsStop
	a.statsStop = nil
	a.mu.Unlock()
	if stop != nil {
		close(stop)
	}
}

func (a *App) statsLoop(stop chan struct{}) {
	t := time.NewTicker(time.Second)
	defer t.Stop()
	for {
		select {
		case <-t.C:
			a.emit("stats", a.GetStatus())
		case <-stop:
			return
		}
	}
}

// ---------- transfers + structured activity ----------

// Transfer implements server.Observer: progress rows are forwarded to the
// UI transfer strip as they happen (~4/s per transfer, throttled in server).
func (a *App) Transfer(t server.Transfer) {
	if t.State != server.TransferActive {
		a.invalidateDiskUsage()
	}
	// A finished upload must be findable immediately, without waiting for
	// the daily rebuild.
	if t.State == server.TransferDone && t.Dir == server.DirIn && t.Name != "" {
		a.mu.Lock()
		store := a.store
		a.mu.Unlock()
		if store != nil {
			a.indexOne(filepath.Join(store.ReceivedDir(), t.Name))
		}
	}
	a.emit("transfer", t)
}

// Activity implements server.Observer: structured rows land in the ring
// buffer and are pushed to the Console table.
func (a *App) Activity(act server.Activity) {
	a.pushActivity(act)
}

// pushActivity appends one structured row (ring buffer, 2000 entries).
func (a *App) pushActivity(act server.Activity) {
	if act.TS == 0 {
		act.TS = time.Now().Unix()
	}
	a.mu.Lock()
	a.activity = append(a.activity, act)
	if len(a.activity) > maxLogLines {
		a.activity = a.activity[len(a.activity)-maxLogLines:]
	}
	a.mu.Unlock()
	a.emit("activity", act)
}

// note records a local (non-server) activity row.
func (a *App) note(kind, detail, status string) {
	a.pushActivity(server.Activity{TS: time.Now().Unix(), Kind: kind, Detail: detail, Status: status})
}

// GetActivity returns the buffered structured rows (oldest first).
func (a *App) GetActivity() []server.Activity {
	a.mu.Lock()
	defer a.mu.Unlock()
	out := make([]server.Activity, len(a.activity))
	copy(out, a.activity)
	return out
}

// CancelTransfer aborts one in-flight transfer (closes its connection).
func (a *App) CancelTransfer(id string) bool {
	a.mu.Lock()
	srv := a.srv
	a.mu.Unlock()
	if srv == nil {
		return false
	}
	return srv.Cancel(id)
}

// invalidateDiskUsage forces the next GetStatus to re-walk the data folder.
func (a *App) invalidateDiskUsage() {
	a.mu.Lock()
	a.diskUsageAt = time.Time{}
	a.mu.Unlock()
}

// cachedDiskUsage returns the data-root size, recomputed at most every
// diskUsageTTL (and after imports/transfers/relocations).
func (a *App) cachedDiskUsage() int64 {
	a.mu.Lock()
	store := a.store
	cached, at := a.diskUsage, a.diskUsageAt
	a.mu.Unlock()
	if store == nil {
		return 0
	}
	if !at.IsZero() && time.Since(at) < diskUsageTTL {
		return cached
	}
	size, err := store.DiskUsage()
	if err != nil {
		size = cached
	}
	a.mu.Lock()
	a.diskUsage = size
	a.diskUsageAt = time.Now()
	a.mu.Unlock()
	return size
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

// ClearActivity empties the structured activity ring buffer.
func (a *App) ClearActivity() {
	a.mu.Lock()
	a.activity = nil
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
	st.LanAddr = netutil.LanAddr()
	st.ProtocolV2 = a.cfg.ProtocolV2
	if m := a.currentSession(); m != nil {
		st.ControlLinks = m.ControlLinks()
		if dev, ok := m.ConnectedDevice(); ok {
			st.Device = dev.Name
		}
	}
	if store != nil {
		st.DataRoot = store.Root()
		st.Shared = store.SharedDir()
		st.Received = store.ReceivedDir()
		st.Onboarded = true
		st.DiskUsage = a.cachedDiskUsage()
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

// SearchUser powers Import's search. It now reads the local file index (the
// same one Find uses) and falls back to a live walk while the index is still
// empty/building, so a fresh install behaves exactly as before.
func (a *App) SearchUser(query string, limit int) []search.Hit {
	if limit <= 0 || limit > 1000 {
		limit = 200
	}
	query = strings.TrimSpace(query)
	if query == "" {
		return []search.Hit{}
	}
	if hits := a.searchIndex(query, limit); len(hits) > 0 {
		return hits
	}
	hits := a.liveSearch(query, limit)
	if hits == nil {
		return []search.Hit{}
	}
	return hits
}

// searchIndex ranks index candidates for the Import search box.
func (a *App) searchIndex(query string, limit int) []search.Hit {
	a.mu.Lock()
	ix := a.idx
	a.mu.Unlock()
	if ix == nil {
		return nil
	}
	q := rank.NewQuery(query)
	rows, err := ix.Candidates(q.Tokens, 2000)
	if err != nil || len(rows) == 0 {
		return nil
	}
	cands := make([]rank.Candidate, 0, len(rows))
	for _, r := range rows {
		cands = append(cands, rank.Candidate{Path: r.Path, Name: r.Name, Dir: r.Dir, Ext: r.Ext, Size: r.Size, MTime: r.MTime})
	}
	hits := rank.NewEngine(a.matchRules()).Rank(q, cands, rank.DefaultCutoff, limit)
	out := make([]search.Hit, 0, len(hits))
	for _, h := range hits {
		out = append(out, search.Hit{Name: h.Name, Path: h.Path})
	}
	return out
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
	for i := range rows {
		rows[i].Exists = pathExists(rows[i].Path)
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
			a.indexTree(p)
			continue
		}
		if abs, err := cat.AddPath(p); err != nil {
			res.Skipped++
		} else {
			res.Added++
			a.indexOne(abs)
		}
	}
	a.invalidateDiskUsage()
	a.logf(fmt.Sprintf("[IMPORT] Cataloged %d path(s) (%d skipped)", res.Added, res.Skipped))
	a.note(server.KindImport, fmt.Sprintf("%d path(s) cataloged · %d skipped", res.Added, res.Skipped), "Complete")
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
	a.indexTree(folder)
	if err != nil && added == 0 {
		return res, err
	}
	a.invalidateDiskUsage()
	a.logf(fmt.Sprintf("[IMPORT] Cataloged folder %s: %d file(s)", folder, added))
	a.note(server.KindImport, fmt.Sprintf("%d file(s) from %s", added, folder), "Complete")
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
		if isDir(path) {
			a.indexRemoveTree(path)
		} else {
			a.indexOne(path) // Add() prunes a path that no longer exists
			a.indexRemove(path)
		}
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
		info.DiskUsage = a.cachedDiskUsage()
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

// ---------- local file index + Find ----------

// indexSetting reads a setting through the catalog ("" on any problem).
func (a *App) indexSetting(key string) string {
	a.mu.Lock()
	cat := a.cat
	a.mu.Unlock()
	if cat == nil {
		return ""
	}
	v, _ := cat.GetSetting(key)
	return v
}

// indexOptions builds a walk configuration from settings, the data folders and
// the catalog (all of which are always searchable).
func (a *App) indexOptions() index.Options {
	opts := index.Options{MaxRows: index.DefaultMaxRows}

	rootsRaw := a.indexSetting("index_roots")
	if strings.TrimSpace(rootsRaw) == "" {
		opts.Roots = search.UserRoots()
	} else {
		for _, line := range strings.FieldsFunc(rootsRaw, func(r rune) bool { return r == '\n' || r == ';' || r == ',' }) {
			if p := strings.TrimSpace(line); p != "" {
				opts.Roots = append(opts.Roots, p)
			}
		}
	}

	skipRaw := a.indexSetting("index_skip")
	if strings.TrimSpace(skipRaw) == "" {
		opts.Skip = []string{".git", "node_modules"}
	} else {
		for _, line := range strings.FieldsFunc(skipRaw, func(r rune) bool { return r == '\n' || r == ';' || r == ',' }) {
			if p := strings.TrimSpace(line); p != "" {
				opts.Skip = append(opts.Skip, p)
			}
		}
	}

	a.mu.Lock()
	store, cat := a.store, a.cat
	a.mu.Unlock()
	opts.Sources = func() []string {
		var out []string
		if store != nil {
			out = append(out, store.ReceivedDir(), store.SharedDir())
		}
		if cat != nil {
			if rows, err := cat.List(); err == nil {
				for _, row := range rows {
					out = append(out, row.Path)
				}
			}
		}
		return out
	}
	return opts
}

// startIndexWarmup walks the disk in the background when the index was never
// built or is older than a day. It never blocks startup.
func (a *App) startIndexWarmup() {
	a.mu.Lock()
	ix := a.idx
	a.mu.Unlock()
	if ix == nil || !ix.NeedsBuild(24*time.Hour) {
		return
	}
	ctx, cancel := context.WithCancel(context.Background())
	a.mu.Lock()
	a.indexCancel = cancel
	a.mu.Unlock()
	go func() {
		defer cancel()
		a.logf("[INDEX] Building the local file index in the background…")
		if err := ix.Build(ctx, a.indexOptions()); err != nil {
			a.logf(fmt.Sprintf("[INDEX] Build stopped: %v", err))
			return
		}
		st := ix.State()
		a.logf(fmt.Sprintf("[INDEX] Indexed %d file(s)", st.Count))
		a.emit("index-state", st)
	}()
}

// GetIndexState reports count/builtAt/building/progress for the UI.
func (a *App) GetIndexState() index.State {
	a.mu.Lock()
	ix := a.idx
	a.mu.Unlock()
	if ix == nil {
		return index.State{}
	}
	return ix.State()
}

// RebuildIndex restarts the walk from scratch (background, cancellable).
func (a *App) RebuildIndex() string {
	a.mu.Lock()
	ix := a.idx
	if a.indexCancel != nil {
		a.indexCancel()
		a.indexCancel = nil
	}
	a.mu.Unlock()
	if ix == nil {
		return "storage not initialized"
	}
	ctx, cancel := context.WithCancel(context.Background())
	a.mu.Lock()
	a.indexCancel = cancel
	a.mu.Unlock()
	go func() {
		defer cancel()
		opts := a.indexOptions()
		if err := ix.Rebuild(ctx, opts); err != nil {
			a.logf(fmt.Sprintf("[INDEX] Rebuild stopped: %v", err))
			a.emit("index-state", ix.State())
			return
		}
		st := ix.State()
		a.logf(fmt.Sprintf("[INDEX] Rebuild complete — %d file(s)", st.Count))
		a.emit("index-state", st)
	}()
	return "rebuilding"
}

// indexOne keeps the index in step with catalog/upload changes.
func (a *App) indexOne(path string) {
	a.mu.Lock()
	ix := a.idx
	a.mu.Unlock()
	if ix == nil {
		return
	}
	_ = ix.Add(path)
}

// indexTree indexes a freshly imported folder (incremental, no full rebuild).
func (a *App) indexTree(root string) {
	a.mu.Lock()
	ix := a.idx
	a.mu.Unlock()
	if ix == nil {
		return
	}
	_ = ix.AddTree(root, a.indexOptions().Skip)
}

// indexRemove drops one path from the index.
func (a *App) indexRemove(path string) {
	a.mu.Lock()
	ix := a.idx
	a.mu.Unlock()
	if ix == nil {
		return
	}
	_ = ix.Remove(path)
}

// indexRemoveTree drops everything below a removed catalog folder.
func (a *App) indexRemoveTree(dir string) {
	a.mu.Lock()
	ix := a.idx
	a.mu.Unlock()
	if ix == nil {
		return
	}
	_ = ix.RemoveUnder(dir)
}

// matchRules returns the persisted rule config (defaults when unset).
func (a *App) matchRules() []rank.Config {
	return rank.ParseConfig(a.indexSetting("match_rules"))
}

// GetMatchRules returns the rule list in tie-break order.
func (a *App) GetMatchRules() []rank.Config { return a.matchRules() }

// SetMatchRules persists a rule list (unknown ids dropped, weights clamped).
func (a *App) SetMatchRules(rules []rank.Config) error {
	a.mu.Lock()
	cat := a.cat
	a.mu.Unlock()
	if cat == nil {
		return fmt.Errorf("storage not initialized")
	}
	raw, err := rank.MarshalConfig(rules)
	if err != nil {
		return err
	}
	if err := cat.SetSetting("match_rules", raw); err != nil {
		return err
	}
	a.note("settings", "Match rules saved", "Complete")
	return nil
}

// ResetMatchRules restores the shipped weights and order.
func (a *App) ResetMatchRules() error {
	return a.SetMatchRules(rank.DefaultConfig())
}

// StartFind kicks off a local search and returns its id. Hits arrive in
// batches via `find-hits`; `find-done` closes the search.
func (a *App) StartFind(query string, opts FindOpts) string {
	a.mu.Lock()
	a.findSeq++
	id := fmt.Sprintf("find-%d", a.findSeq)
	ix := a.idx
	if a.findCancels == nil {
		a.findCancels = map[string]context.CancelFunc{}
	}
	a.mu.Unlock()

	query = strings.TrimSpace(query)
	if query == "" {
		a.emit("find-done", FindDone{SearchID: id, Total: 0, TookMS: 0, SkippedNote: "empty query"})
		return id
	}
	ctx, cancel := context.WithCancel(context.Background())
	a.mu.Lock()
	a.findCancels[id] = cancel
	a.mu.Unlock()

	go func() {
		defer func() {
			a.mu.Lock()
			delete(a.findCancels, id)
			a.mu.Unlock()
			cancel()
		}()
		start := time.Now()
		q := rank.NewQuery(query)
		engine := rank.NewEngine(a.matchRules())
		limit := opts.Limit
		if limit <= 0 {
			limit = 50
		}
		note := ""
		var cands []index.Entry
		if ix != nil {
			rows, err := ix.Candidates(q.Tokens, 2000)
			if err != nil {
				note = "index unavailable"
			}
			cands = rows
		}
		if len(cands) == 0 {
			// Index empty or not built yet: fall back to a bounded live walk
			// so Find still answers on a fresh install.
			if count, _ := ixCount(ix); count == 0 {
				note = "index still building — searched live"
			}
			for _, hit := range a.liveSearch(query, 300) {
				cands = append(cands, index.Entry{Path: hit.Path, Name: hit.Name, Dir: filepath.Dir(hit.Path), Ext: strings.TrimPrefix(strings.ToLower(filepath.Ext(hit.Name)), ".")})
			}
		}
		if ctx.Err() != nil {
			return
		}
		candsConverted := make([]rank.Candidate, 0, len(cands))
		for _, c := range cands {
			candsConverted = append(candsConverted, rank.Candidate{
				Path: c.Path, Name: c.Name, Dir: c.Dir, Ext: c.Ext, Size: c.Size, MTime: c.MTime,
			})
		}
		hits := engine.Rank(q, candsConverted, opts.Cutoff, limit)
		// Batch the hits so the UI can paint incrementally.
		for i := 0; i < len(hits); i += 50 {
			if ctx.Err() != nil {
				return
			}
			end := i + 50
			if end > len(hits) {
				end = len(hits)
			}
			a.emit("find-hits", FindHits{SearchID: id, Hits: hits[i:end]})
		}
		a.emit("find-done", FindDone{
			SearchID: id, Total: len(hits),
			TookMS: time.Since(start).Milliseconds(), SkippedNote: note,
		})
	}()
	return id
}

func ixCount(ix *index.Indexer) (int, error) {
	if ix == nil {
		return 0, nil
	}
	return ix.Count()
}

// CancelFind aborts a running search.
func (a *App) CancelFind(searchID string) bool {
	a.mu.Lock()
	cancel, ok := a.findCancels[searchID]
	a.mu.Unlock()
	if ok {
		cancel()
		return true
	}
	return false
}

// liveSearch is the pre-index fallback: bounded walk over the user folders.
func (a *App) liveSearch(query string, limit int) []search.Hit {
	hits, _ := search.SearchUser(query, limit)
	if hits == nil {
		return []search.Hit{}
	}
	return hits
}
