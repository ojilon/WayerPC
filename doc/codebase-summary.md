Repo root: D:\projects\WayerPC. Go + Wails (TS/Vite) desktop app: phone ↔ PC file transfer over TCP port 5000 (hotspot). App version 0.2.0, protocol version 1.
1. Repo map (depth 3, one line per entry)
docs/                         # curated codebase summary (this file)
doc/                          # rewrite plan (00-overview ... 07-build-test-release, PLAN-checklist)
legacy/                       # archived Python/C++ implementation (reference only; no runtime use)
    backend/                  # old TCP server, catalog, usersearch, filesearch, native DLLs, manage_storage
    c/                        # dead C code (ARCHITECTURE.md says deletable)
    frontend-py/              # old CustomTkinter UI
    packaging/                # old PyInstaller spec
build/                        # wails build artifacts (bin/WayerPC.exe, windows manifest/icon/NSIS info)
out/                          # WayerPC-0.2.0-setup.exe (built installer, gitignored)
frontend/                     # Vite + TypeScript + HTML/CSS frameless UI (pnpm)
    src/                      # main.ts, api.ts, dev-mock.ts, version.ts, styles.css, views/*
    dist/                     # built output (gitignored), embedded by Go via //go:embed all:frontend/dist
    wailsjs/                  # generated Wails TS bindings (models.ts, go/main/App.js/.d.ts, runtime/*)
internal/                     # pure-Go backend (no Python/C++)
    catalog/                  # SQLite import catalog (catalog.go, catalog_test.go)
    config/                   # config.json load/save, data-root resolution (config.go, config_test.go)
    search/                   # filesystem user/app search (search.go, search_test.go)
    server/                   # TCP phone-link server + wire protocol (server.go, protocol.go, transfer.go, server_test.go)
    storage/                  # data-root layout shared/received/Data, relocate (storage.go, storage_test.go)
    version/                  # embedded version.json mirror (version.go, version_test.go, version.json)
scripts/                      # sync-version.mjs, make-icon.go, build-installer.ps1
main.go                       # Wails entry: flag --version, wails.Run, binds App, embeds frontend/dist
app.go                        # the ONLY frontend↔backend bridge (App struct + all bound methods)
helpers.go                    # base/isDir/pathExists/sameDir small helpers
go.mod / go.sum               # module wayerpc, Go 1.27.0, wails v2.15.0, modernc.org/sqlite
wails.json                    # Wails config: frontend dir, install/build commands, dev URL
version.json                  # single source of truth: appName, version, protocolVersion, description
README.md / README_PC_END.md  # user docs (README_PC_END.md describes legacy Python end)
BUILD.md                      # dev/build/installer/release commands
LICENSE
2. Stack
- Go: go 1.27.0 (go.mod line 3). Docs say ≥1.23.
- Wails: github.com/wailsapp/wails/v2 v2.15.0; Wails CLI v2 for wails dev / wails build. main.go calls wails.Run(&options.App{...}) with Bind: []any{app}, frameless window, asset server embedding frontend/dist.
- Frontend: Vanilla TypeScript + Vite, no framework (React/Vue/Svelte). Hand-rolled views. frontend/ is its own pnpm workspace.
- CSS: frontend/src/styles.css (dark tokens --bg:#0f1419, --accent:#3b82f6, etc.). Linked from frontend/index.html via /src/styles.css.
- Frontend build: pnpm build = node ../scripts/sync-version.mjs && tsc --noEmit && vite build → frontend/dist/. Dev: pnpm dev (vite on http://localhost:5173, uses dev-mock.ts when window.go absent). Wails dev URL http://localhost:5173 (wails.json).
- Exact commands (from BUILD.md / doc/07-build-test-release.md):
- Backend tests: go test ./... (or ./internal/...)
- Frontend install: cmd /c "pnpm.cmd --dir frontend install"
- Frontend dev (browser mock): cmd /c "pnpm.cmd --dir frontend dev" → http://localhost:5173
- Full app dev: $env:WAYERPC_DATA_DIR = "$PWD/.data"; wails dev
- Production build: node scripts/sync-version.mjs → cmd /c "pnpm.cmd --dir frontend build" → wails build -platform windows/amd64 -o bin/WayerPC.exe
- Installer: powershell -ExecutionPolicy Bypass -File scripts/build-installer.ps1 → out/WayerPC-<ver>-setup.exe
3. Go ↔ frontend bridge
app.go binds the single App struct (Bind: []any{app} in main.go). Every frontend call goes through window.go.main.App.<method> (wrapped by frontend/src/api.ts call()). Async backend pushes use runtime.EventsEmit.
Bound Go methods (full signatures, app.go)
#	Method	Signature
1	GetVersion	func (a *App) GetVersion() VersionInfo
2	GetStatus	func (a *App) GetStatus() Status
3	StartServer	func (a *App) StartServer() string
4	StopServer	func (a *App) StopServer() string
5	GetLogs	func (a *App) GetLogs() []string
6	ClearLogs	func (a *App) ClearLogs()
7	SearchUser	func (a *App) SearchUser(query string, limit int) []search.Hit
8	SearchApp	func (a *App) SearchApp(query string) []search.Hit
9	CatalogList	func (a *App) CatalogList() []catalog.Entry
10	CatalogAddPaths	func (a *App) CatalogAddPaths(paths []string) (ImportResult, error)
11	CatalogAddFolder	func (a *App) CatalogAddFolder(folder string) (ImportResult, error)
12	CatalogRemove	func (a *App) CatalogRemove(path string) error
13	ListReceived	func (a *App) ListReceived() []storage.FileInfo
14	ListShared	func (a *App) ListShared() []storage.FileInfo
15	GetStorageInfo	func (a *App) GetStorageInfo() StorageInfo
16	SetDataRoot	func (a *App) SetDataRoot(newRoot string) (StorageInfo, error)
17	PickFolder	func (a *App) PickFolder(title string) string
18	PickFiles	func (a *App) PickFiles() []string
19	RevealInExplorer	func (a *App) RevealInExplorer(path string) string
20	GetSetting	func (a *App) GetSetting(key string) string
21	SetSetting	func (a *App) SetSetting(key, value string) error
Generated TS declarations: frontend/wailsjs/go/main/App.d.ts (and App.js which does window['go']['main']['App'][method]). api.ts does not import those; it calls window.go?.main?.App[method] directly with a dev-mock.ts fallback.
Key payload structs (Go, app.go)
type VersionInfo struct {
    AppName         string `json:"appName"`
    Version         string `json:"version"`
    ProtocolVersion int    `json:"protocolVersion"`
    Description     string `json:"description"`
}

type Status struct {
    Server      server.Stats `json:"server"`
    DataRoot    string       `json:"dataRoot"`
    Shared      string       `json:"shared"`
    Received    string       `json:"received"`
    Onboarded   bool         `json:"onboarded"`
    DiskUsage   int64        `json:"diskUsage"`
    CatalogSize int          `json:"catalogSize"`
}

type StorageInfo struct {
    DataRoot   string `json:"dataRoot"`
    InstallDir string `json:"installDir"`
    Shared     string `json:"shared"`
    Received   string `json:"received"`
    DbPath     string `json:"dbPath"`
    DiskUsage  int64  `json:"diskUsage"`
    Onboarded  bool   `json:"onboarded"`
}

type ImportResult struct {
    Added   int `json:"added"`
    Skipped int `json:"skipped"`
}
server.Stats (internal/server/server.go):
type Stats struct {
    Running       bool   `json:"running"`
    Addr          string `json:"addr"`
    Active        int    `json:"active"`
    Total         int64  `json:"total"`
    BytesSent     int64  `json:"bytesSent"`
    BytesReceived int64  `json:"bytesReceived"`
    StartTime     int64  `json:"startTimeUnix"`
    UptimeSecs    int64  `json:"uptimeSecs"`
}
Runtime events emitted to the frontend
Emitted via App.emit(name, data...) → runtime.EventsEmit(a.ctx, name, data...) (app.go:194). Three event names:
Event	Payload shape	Where emitted
log	string line	logf() (app.go:179) — every backend log line; also catalog-changed if line contains [UPLOAD OK]
stats	Status (full struct above)	statsLoop() every 1s (app.go:164)
catalog-changed	no payload	openRoot, CatalogAddPaths, CatalogAddFolder, CatalogRemove
Note: despite doc/03-frontend-wails.md mentioning a search-progress event, no such event is emitted anywhere in the current Go code (grep confirmed — the doc is aspirational).
How the frontend subscribes
api.ts:
export function onEvent(event: string, cb: (data: any) => void): () => void {
  const rt = window.runtime;
  if (rt?.EventsOn) {
    rt.EventsOn(event, cb);
    return () => rt.EventsOff?.(event);
  }
  // dev-mock fallback ...
}
Example — Console view (frontend/src/views/console.ts:85-89):
const offs = [
    onEvent("log", (line) => append(String(line ?? ""))),
    onEvent("stats", (st) => paint(st as Status)),
    onEvent("catalog-changed", () => api.getStatus().then(paint).catch(() => {})),
];
End-to-end example
1. User clicks "Start server" in Console → api.startServer() → window.go.main.App.StartServer() → Go App.StartServer() calls srv.Start() in a goroutine, which binds TCP 0.0.0.0:5000 and accepts phone connections.
2. Phone sends /ask report.pdf → Go handleCommand → streamFile writes FOUND 12345\n + raw bytes over the same TCP conn.
3. Separately, Go App.logf("[ASK OK] Sent file: report.pdf") → runtime.EventsEmit(ctx, "log", line).
4. Frontend onEvent("log", ...) appends [hh:mm:ss] [ASK OK] Sent file: report.pdf to the log panel.
5. Every 1s, statsLoop → emit("stats", a.GetStatus()) → Console paints active links / bytes sent / uptime.
6. A phone upload triggers [UPLOAD OK] in the log → emit("catalog-changed") → Library view re-lists catalogList().
4. Wire protocol with the phone (most important)
Source: internal/server/server.go, protocol.go, transfer.go. Port of legacy backend/server/server4.py + ExecuteServerCommand.py, byte-compatible per the package doc.
Transport and framing
- Transport: TCP, PC listens on host:port (default 0.0.0.0:5000, config internal/config DefaultHost/DefaultPort). Phone is the client; PC is strictly the TCP server. One listener, one goroutine per connection (serveConn), many concurrent connections allowed (net.Listen accept loop spawning go s.serveConn(conn)).
- Framing: line-based text, newline-terminated (\n or \r ends a command immediately). No length prefix, no binary header. Commands are ASCII/UTF-8 text like /ask report.pdf.
- Newline-optional settle window: If the client sends no line ending, the command terminates after settleTimeout = 500 * time.Millisecond of silence (sliding per byte, so split TCP segments still join). This exists because the Android client sends the header with no trailing newline then waits for the reply (server.go:275-298, readCommand). Idle read deadline between commands: idleTimeout = 300 * time.Second. Control write deadline: writeTimeout = 60 * time.Second (refreshed per chunk during bulk transfer). Max command line: maxLine = 64 * 1024 bytes.
- Byte order: N/A — text protocol, no multi-byte binary integers on the wire (sizes are decimal text in FOUND <size>\n).
Max sizes
- DefaultMaxUploadBytes = int64(2 << 30) (2 GiB) per upload; larger → ERROR upload_too_large (protocol.go:120).
- maxLine = 64 KiB per command line.
Handshake / version field ("proto 1")
There is no handshake and no version field exchanged over the wire. The connection starts directly with commands (/ask, /upload). protocolVersion (currently 1, from version.json) appears only in:
- VersionInfo returned by GetVersion() to the frontend (titlebar/Settings).
- version.String() log line "WayerPC v0.2.0 (proto 1)".
- Stored in catalog settings(app_version).
No mismatch handling exists on the wire — there's nothing to mismatch. docs/SUGGESTIONS-v0.2.1.md proposes a /hello <client-version> handshake and pairing code as future work (not implemented).
Every message/command type
Two commands, both text lines:
/ask <name> — phone wants a file from PC.
- handleCommand (protocol.go:20-35): tolerates a duplicated prefix (/ask /ask file), strips it.
- Resolution order (resolveAndSend, protocol.go:47):
1. Catalog fuzzy match (cat.FindForRequest(query, catalog.DefaultCutoff) where DefaultCutoff = 0.2).
2. If exactly one exact name match (case-insensitive) → stream that file.
3. If exactly one hit with score ≥ 0.8 → stream it.
4. If >1 hits, or one weak hit (<0.8) → sendMatches: MATCHES <n>\n then one name\n per hit (max 50).
5. Else fallback search.SearchAppFolders(query, shared/ + received/) exact name → stream first found.
6. Else ERROR file_not_found.
- Stream: sendFile (transfer.go:22) writes FOUND <size>\n then raw file bytes in 32 KiB chunks, refreshing the write deadline per chunk.
- Reply tokens on the wire: FOUND <size>\n + bytes / MATCHES <n>\n<name>\n… / ERROR file_not_found / ERROR file_access_denied.
/upload <size> <name> — phone sends a file to PC (saved to received/, cataloged).
- handleUpload (protocol.go:86-175): parses sizeToken (must be digits), filename = filepath.Base(join(parts[2:], " ")) (names with spaces joined; legacy took parts[2] only).
- Size cap check vs s.maxUpload.
- Replies bare READY (no newline — see suggestions #1), then recvExact (transfer.go:63) reads exactly filesize bytes into <name>.part in received/, 32 KiB chunks, refreshing read deadline per chunk. On complete → os.Rename(partPath, finalPath), reply DONE (with newline: sendLine writes DONE\n), cat.AddPath(finalPath) to catalog.
- Errors: ERROR write_failed, ERROR upload_incomplete (partial .part removed), ERROR upload_too_large.
- Reply tokens: READY / DONE\n / ERROR ….
/upload <query> (no integer size, or fewer than 3 parts) — catalog search only, no transfer: returns MATCHES <n>\n<name>\n… (same as /ask multi-hit) or ERROR file_not_found.
Status codes (internal, logged, not sent over wire)
internal/server/server.go:24-37:
const (
    StatusAskOK = 0   // file streamed to phone
    StatusUploadOK = 3 // upload saved
    StatusMatches = 4  // MATCHES list sent
    StatusProtocol = 1 // protocol/validation problem
    StatusStorage = 2  // storage problem
    StatusFail = -1    // failure already sent to client
)
Log prefixes: [ASK OK], [UPLOAD OK], [MATCHES], [PROTOCOL], [STORAGE ERROR], [COMMAND FAIL].
File send/receive detail
Send (PC → phone, /ask): sendFile opens file, stats size, writes FOUND %d\n, then loops f.Read into 32 KiB buffer + conn.Write, refreshing writeTimeout per chunk. No checksum, no progress events to frontend beyond bytesSent counter, no resume.
Receive (phone → PC, /upload): header /upload <size> <name> → READY → recvExact reads exactly size bytes to <name>.part, fsync, rename to final name, DONE. Partial files removed on error. No checksum, no progress beyond bytesReceived counter, no resume. docs/SUGGESTIONS-v0.2.1.md proposes DONE <sha256> and resume (not implemented).
Who may initiate
- PC is strictly the TCP server; phone strictly the client. The PC listens on 0.0.0.0:5000. The phone connects.
- Can the PC push a message to a connected phone today? No. The protocol is request/response over the phone-initiated connection: phone sends /ask or /upload, PC replies on the same conn. There is no server-initiated message, no callback channel, no push. A connection can carry multiple sequential commands (the serveConn loop persists until disconnect/timeout), but each is phone-initiated.
- Connections: not short-lived per-request by design — serveConn loops handling multiple commands per connection until EOF/timeout/fatal. But in practice each phone action opens a connection, sends one command, gets the reply, and the phone closes. Many concurrent connections supported (goroutine per conn); no per-IP cap (suggested as future work).
- "Active links" and "Handled": Stats.Active = srv.active (incremented on Accept, decremented in serveConn defer — concurrent connections right now). Stats.Total = srv.total (cumulative connections accepted). Displayed in Console as "Active links" / "Handled" (frontend/src/views/console.ts:29-32).
- Pairing / auth / device identity: None. Anyone on the hotspot who can reach port 5000 can /ask or /upload. No pairing code, no auth, no device identity. docs/SUGGESTIONS-v0.2.1.md proposes a 6-digit pairing code + /auth (future, breaking).
Concurrency
Many connections at once (goroutine per conn). active/total/bytesSent/bytesReceived guarded by s.mu. store/cat behind backendMu (RWMutex) so data-root relocation can hot-swap backends without stopping the listener. Catalog is single-writer (catalog.DB.mu). Server.Stop() closes the listener; running connections drain (not forcibly closed).
5. Storage
Directory layout (internal/storage/storage.go:1-11, doc/06-data-sqlite.md)
<dataRoot>/
├── shared/            # optional drop folder, also served by /ask fallback
├── received/          # phone uploads land here (/upload saves here)
└── Data/
    └── wayerpc.db     # SQLite catalog (modernc.org/sqlite, WAL mode)
What each means
- shared/: optional user drop folder. Files placed here are servable via /ask fallback (resolveAndSend searches shared/ + received/ exact name). Not auto-cataloged (SUGGESTIONS proposes an fsnotify watcher — not implemented). Imported items are never copied into shared/.
- received/: phone uploads land here (/upload saves filename here, then cat.AddPath(finalPath) so it's immediately /ask-able).
- catalog (Data/wayerpc.db): registered imported paths (absolute paths, not copies).
SQLite schema (exact, internal/catalog/catalog.go:23-36)
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
Applied in catalog.Open with pragmas journal_mode=WAL; synchronous=NORMAL; foreign_keys=ON;. settings holds app_version (and per docs data_root, host, port, library_view, onboarded).
How files are identified
By absolute path (imported_files.path TEXT UNIQUE), plus name (basename), size_bytes (from os.Stat at import time), imported_at timestamp. No hash, no content fingerprint stored. Modified time is read at import (st.Size()) but mod time is not stored in the catalog (only size_bytes); storage.FileInfo.ModTime comes from os.Stat at list time for received//shared/ views. Files are identified by path; FindForRequest re-stats each hit and drops missing-on-disk rows.
Legacy migration: catalog.MigrateLegacy copies rows from <root>/Data/imports.db (Python-era) into wayerpc.db when the new DB is empty (same schema, INSERT OR IGNORE).
6. Existing search/import code
Import screen search (internal/search/search.go)
SearchUser(query, limit) → SearchRoots(UserRoots(), query, limit).
Roots (UserRoots, search.go:57-82):
- Windows: %USERPROFILE%\{Documents,Downloads,Desktop,Pictures,Videos,Music} (only if they exist) + every ready drive D:–Z: (readyDrives).
- Non-Windows: /mnt, /media.
Skip rules (skipSegments, search.go:24-37, matched per path segment, case-insensitive, exact match — so my-windows-themes is NOT skipped):
windows, "program files", "program files (x86)", programdata, appdata,
$recycle.bin, "system volume information", windowsapps, wayerpc,
proc, sys, dev
SkipPath/SkipRel applied to root-relative rel paths. Test expectations (search_test.go:10-24) confirm C:\Windows\System32\cmd.exe, C:\Program Files\app\a.txt, C:\Users\me\AppData\Local\x, D:\$RECYCLE.BIN\foo, and anything under .../WayerPC/data/shared/ are skipped; D:\projects\notes\todo.txt is not.
Match semantics: case-insensitive substring on file and folder names. A matching folder contributes its directly-contained files (one level). Bounded by limit; unreadable entries skipped; result sorted by name then path. SearchAppFolders(query, folders) does exact (case-insensitive) name match inside shared/ + received/ (used by /ask fallback and SearchApp).
"Browse files/folder"
- Browse files: api.pickFiles() → App.PickFiles() → runtime.OpenMultipleFilesDialog → api.catalogAddPaths(paths) → App.CatalogAddPaths (each path: cat.AddPath, dirs would expand via cat.AddFolder — though PickFiles returns files; CatalogAddPaths handles dirs too).
- Browse folder: api.pickFolder(title) → App.PickFolder() → runtime.OpenDirectoryDialog → api.catalogAddFolder(folder) → App.CatalogAddFolder → cat.AddFolder (recursive walk, registers every regular file).
- All import registers paths only, never copies files (storage.go doc: "Imported items are paths, not copies — nothing is ever duplicated into shared/"). Catalog AddPath stats the file (must exist), stores abs path + name + size_bytes.
7. Config
Persisted at OS config dir: config.Dir() = %LOCALAPPDATA%\WayerPC\config.json on Windows (os.UserConfigDir + AppName); file config.json via config.FilePath().
Struct (internal/config/config.go:33-39)
type Config struct {
    Host     string `json:"host"`
    Port     int    `json:"port"`
    DataRoot string `json:"data_root"`
    InstallDir string `json:"install_dir,omitempty"`
}
Defaults() = {Host: "0.0.0.0", Port: 5000} (no DataRoot). Load() reads config.json leniently (missing/corrupt → defaults). Also understands legacy keys app_dir/sub_dir (rawFile). Save() marshals indent + trailing newline, creates config dir.
Resolution order (ResolveDataRoot, config.go:142-150)
1. WAYERPC_DATA_DIR env var (dev stub .data/, tests).
2. cfg.DataRoot from config.json.
3. Else "" → caller shows onboarding.
Plus DevDataStub() = <cwd>/.data if it exists (fresh-clone runs).
How the data folder is changed at runtime
App.SetDataRoot(newRoot) (app.go:442-459): trims, rejects empty; if an old store exists and differs, old.Relocate(newRoot) (same-drive os.Rename, else copy+verify+swap via .relocate-tmp); then a.openRoot(newRoot) which re-inits store, opens/migrates catalog, (re)attaches server, saves config, starts server goroutine, emits catalog-changed. Frontend: Settings → "Choose folder…" → confirm dialog ("Existing data moves with it") → api.setDataRoot(folder) → GetStorageInfo repaint. storage.Store.Relocate (storage.go:141-169).
8. Logging / activity feed
Produced by App.logf(line string) (app.go:179-192):
- Appends to in-memory a.logs []string, capped at maxLogLines = 2000 (drops oldest).
- If line contains [UPLOAD OK], also emits catalog-changed.
- Always emits runtime.EventsEmit(ctx, "log", line).
GetLogs() returns a copy of the buffer; ClearLogs() empties it. Console view (console.ts) subscribes to log and paints [hh:mm:ss] <line>, keeping max 2000 DOM nodes, with pause/filter/clear. ClearLogs clears both UI and Go buffer. Logs are not persisted to disk (only in memory; SUGGESTIONS §2 proposes Data/logs.jsonl rotation — not implemented). Server log lines ([CONNECT], [ASK OK], [UPLOAD OK], [MATCHES], [PROTOCOL], [STORAGE ERROR], [COMMAND FAIL], [TIMEOUT], [DISCONNECT]) flow through the same logf callback wired at server.New(..., a.logf).
9. Tests, dependencies, fragile/duplicated/half-finished items
Tests
go test ./... covers: internal/server (protocol over net.Pipe: ask exact/matches/not-found/fallback-shared, upload round trip incl. READY→bytes→DONE, incomplete upload cleanup, newline-less headers, split-header joining, fatal on undeliverable READY, start/stop), internal/catalog (add/list/remove, missing fails, folder add, score parity exact/substring/typo, FindForRequest drops missing, settings roundtrip), internal/search (skip rules, roots/limit, exact app-folder search), internal/storage (init tree, empty root fails, list files missing/present), internal/version (version_test.go). All use t.TempDir().
go.mod highlights
module wayerpc
go 1.27.0
require github.com/wailsapp/wails/v2 v2.15.0
require modernc.org/sqlite v1.60.1   (pure Go SQLite, no cgo)
Everything else indirect (wails brings gorilla/websocket, go-webview2, etc.). No cgo/C toolchain needed.
Fragile / duplicated / half-finished
- No wire handshake/version exchange — protocolVersion only lives in version.json/frontend; Android client may expect something else (unknown). Risk of silent incompatibility.
- READY sent bare with no \n (protocol.go:147, transfer.go sendLine adds \n to everything else). Android client special-cases this; SUGGESTIONS #1 flags it.
- No checksum on transfers — DONE carries no hash; silent corruption possible on hotspot (SUGGESTIONS #2).
- No resume — partial .part removed on failure; whole file re-sent (SUGGESTIONS resume).
- No auth/pairing / per-IP cap — open hotspot, unbounded goroutines per IP (SUGGESTIONS).
- DiskUsage() walks the whole tree on every 1s stats tick (SUGGESTIONS cache it).
- No busy_timeout on SQLite despite WAL; upload vs UI import race could hit SQLITE_BUSY (SUGGESTIONS).
- Status.Addr reports 0.0.0.0, useless to type into a phone (SUGGESTIONS: show first private IPv4).
- search-progress event documented in doc/03-frontend-wails.md but never emitted — long SearchUser walks give no frontend feedback.
- dev-mock.ts drift risk — mock hand-maintained, no CI parity check (SUGGESTIONS #5). Mock differs from real Go (e.g. mock SearchUser takes 1 arg, real takes 2).
- Version sync must run before build — internal/version/version.json is a generated mirror; go:embed forbids .., so root version.json is copied by scripts/sync-version.mjs. Out-of-sync mirror would embed stale version.
- catalog.FindRelated is a linear scan over all rows with per-name bigram-Dice scoring — fine to ~10k rows, sloggy past 100k; FTS5 proposed, not implemented.
- Legacy code under legacy/ is reference only — old Python server4.py/ExecuteServerCommand.py, C++ DLLs, CustomTkinter UI; not imported by the Go build. docs/ARCHITECTURE.md and doc/01-frontend-backend-split.md describe the old world (DLL loading, native SQLite amalgamation) which is deleted in the rewrite.
- Duplication: skip-list logic exists once in Go (search.skipSegments); legacy Python had _SKIP_PARTS. Scoring logic duplicated conceptually (Go bigram-Dice ≈ Python difflib). No functional duplication in the active codebase beyond that.
- app.go holds statsStop chan and statsLoop; on SetDataRoot→openRoot a new statsLoop goroutine is started but the old statsStop is not closed before replacement (leak of the old ticker goroutine on relocate) — minor.
10. Notes about the Android app / protocol expectations
README.md line 6: Android client is a separate repo: [ojilon/Wayer](https://github.com/ojilon/Wayer) — not in this repo, separate release train. docs/SUGGESTIONS-v0.2.1.md:162-163: "Rewriting the Android client here — it lives in ojilon/Wayer, separate repo and release train. PC side stays client-tolerant instead."
Protocol expectations the PC side codes for (from server.go comments and tests):
- Android sends /ask <name> with no trailing newline, then waits for the reply → hence the 500 ms settle window (readCommand).
- Android reads READY as a bare token (5 bytes, no \n), then streams bytes, then expects DONE (test TestUploadBinaryRoundTrip reads exactly 5 bytes for "READY").
- Android default port 5000 (config.DefaultPort comment: "mirrors the Python server and the Android client default").
- No handshake, no auth, no pairing in the current client.
There are no protocol spec documents, no API contract files, and no Android source in this repo describing what the Android app expects beyond the wire behavior the Go server replicates.
Things I couldn't determine
 1. Exact Android client framing behavior — the repo has no Android source; inferred from Go server comments/tests only (newline-less header, bare READY read, port 5000). Actual read() buffer sizes, retry/backoff logic, and whether the Android side tolerates READY\n are unknown.
 2. What happens on protocol version mismatch — there is no version field on the wire at all, so mismatch behavior is undefined (nothing to compare).
 3. Whether the phone ever reconnects/reuses a single long-lived connection — serveConn supports multiple commands per connection, but it's unknown whether the Android client keeps one socket open across operations or opens/closes per command.
 4. Expected MATCHES format on the phone — names-only today (MATCHES <n>\n<name>\n…); whether the Android parser expects tabs/sizes/dates is unknown (SUGGESTIONS proposes richer MATCHES as breaking).
 5. File size limits the Android side enforces — PC caps at 2 GiB; phone-side limit unknown.
 6. Whether connections are authenticated by IP/MAC or any pairing code today — none exists; whether the Android app implements any out-of-band pairing is unknown.
 7. Exact Android timeout values — PC uses 300 s idle / 500 ms settle / 60 s write; phone-side timeouts unknown.
 8. Whether shared/ auto-cataloging or fsnotify exists on the phone side — not on PC; Android behavior unknown.
 9. Installer EnVar plugin issue — docs/SUGGESTIONS-v0.2.1.md claims installer.nsi uses non-stock EnVar::AddValue/DeleteValue and fails on fresh NSIS; I did not verify installer.nsi contents directly (truncated in doc/05-installer.md).
10. Whether wails dev actually works end-to-end — dev requires WAYERPC_DATA_DIR env; not tested in this session.