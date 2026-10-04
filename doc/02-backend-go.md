# 02 — Backend (pure Go)

Module `wayerpc`, Go ≥ 1.23. Extra deps: `modernc.org/sqlite` (pure Go,
no cgo/C toolchain), `github.com/wailsapp/wails/v2` (shell only).
No numpy/cgo/DLL loading.

## `internal/version`

- `//go:embed ../../version.json`, exposes `Info{AppName, Version,
  ProtocolVersion, BuildDate}` + `String()`. Frontend + installer read the
  same file (see `doc/04-versioning.md`).

## `internal/config`

- `Config{Host, Port, DataRoot, BufferSize}`. Defaults `0.0.0.0:5000`.
- Resolution order: explicit `SetDataRoot` → `WAYERPC_DATA_DIR` env (dev stub
  `.data/`) → `%LOCALAPPDATA%\WayerPC\config.json` (`{app_dir,data_root}`)
  → installer default (`<chosenDir>\WayerPC`) → `.data/` fallback in dev.
- `Load/Save`, `ConfigDir()`, `ReadConfigJSON` tolerant of missing/corrupt
  files (same leniency as legacy `load_saved_paths`).

## `internal/storage`

- `Init(dataRoot)` creates `<root>/{shared,received,Data}` (+ `.nomedia`
  not needed). `EnsureShared/Received/DataDir`, `SharedDir/ReceivedDir/DbPath`,
  `ListFiles(dir)` → `[{name,path,size,modTime}]`, `RevealInExplorer` (rundll
  / open / xdg-open), `Relocate(newRoot)` (move + update config).
- `DbPath()` = `<root>/Data/wayerpc.db`. On first open, if legacy
  `<root>/Data/imports.db` exists and new DB is empty, import its
  `imported_files` rows (zero-downtime migration).

## `internal/catalog` (SQLite)

- Same `imported_files(id, path UNIQUE, name, size_bytes, imported_at)` plus
  `settings(key TEXT PRIMARY KEY, value TEXT)` for app prefs.
- API: `Open(dbPath)`, `AddPath(path)` (stat, `INSERT OR REPLACE`, no copy),
  `AddFolder(root)` (`(added, skipped)` walk), `Remove(path)`,
  `List()` (order by name), `FindRelated(query, cutoff=0.2)` with legacy
  scoring: exact→1.0; substring→`max(0.6, len(q)/len(name))`;
  else bigram-Dice ≈ `difflib` ratio (documented approximation);
  `FindForRequest` filters to existing files, sorts desc by score.
- Single-writer mutex + WAL mode; every mutating call emits
  `catalog-changed` from `app.go`.

## `internal/search`

- `UserSearch(query, limit)` — roots: `%USERPROFILE%\{Documents,Downloads,
  Desktop,Pictures,Videos,Music}` + existing `D:`–`Z:` drives (else
  `/mnt`, `/media`); skips `windows`, `program files*`, `programdata`,
  `appdata`, `$recycle.bin`, `system volume information`, `windowsapps`,
  any `wayerpc` segment (same as legacy `_SKIP_PARTS`). Case-insensitive
  substring on file **and** folder names; folder hit lists its files
  (parity with Python fallback; the C++ DLL behavior is superseded).
- `AppSearch(query, dataRoot, limit)` — exact/fuzzy walk over
  `shared/` + `received/` (replaces `libfilesearch`).
- Pure stdlib (`filepath.WalkDir`), bounded `limit`, permission errors
  skipped, no symlinks followed outside root.

## `internal/server`

- `Server{cfg, store, catalog, stats, logger, onEvent}`. `Start()` binds
  `HOST:PORT`, `Accept` loop, per-conn goroutine (mirrors
  `handle_client`), 300s read deadline, line-buffered commands
  (`bufio.Reader.ReadString('\n')`, trims; multi-command lines handled).
- `protocol.go` `handleCommand(line, conn, state)` implements **byte-identical
  wire behavior**: `FOUND <size>\n`+bytes / `MATCHES <n>\nname\n…` (names only,
  first 50, legacy `_send_matches`), `READY`→bytes→`DONE`, `ERROR <code>`
  codes: `invalid_command`, `file_not_found`, `file_access_denied`,
  `file_directory_missing_or_unreadable`, `invalid_upload_command`,
  `upload_incomplete`, `write_failed`, `unknown_protocol_command.`
  (trailing dot kept).
- Uploads capped (`MaxUploadBytes`, default 2 GiB) to avoid disk-fill;
  partial files removed. `transfer.go` uses 32 KiB chunks.
- Stats `{Running, Active, Total, BytesSent, BytesReceived, StartTime}` under
  mutex; `Snapshot()` for the Console tab (1s poll) + `Logf` callback wired
  to Wails events. Status codes 0/3/4/1/2/-1 preserved internally for log
  parity (`[ASK OK]`, `[UPLOAD OK]`, `[MATCHES]`, `[PROTOCOL]`,
  `[STORAGE ERROR]`, `[COMMAND FAIL]`).

## `app.go` bindings (called from TS)

`GetVersion, GetStatus, StartServer, StopServer, GetLogs, ClearLogs,
SearchUser, SearchApp, CatalogList/AddPaths/AddFolder/Remove,
ListReceived, ListShared, GetStorageInfo, SetDataRoot, PickFiles,
PickFolder, RevealInExplorer, GetSetting, SetSetting`.
Long searches run in goroutines with incremental `search-progress` events;
`Pick*` uses `runtime.OpenFileDialog` so no frontend file-input hacks.
