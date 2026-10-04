# WayerPC rewrite plan — Python/C++ → Go backend + Wails (TS/Vite) frameless frontend

## 0. Where we are (legacy, surveyed 2026-10-04)

- `frontend/*.py` — CustomTkinter window: **Console** (pinned tab: status,
  network stats, activity log), **Import** tabs (search user storage, tick hits,
  `Browse files/folder`, register paths), **Received/Library** tabs
  (`received/` + `Imported catalog`), `+` opens extra tabs, `Close tab` removes
  extras. Server runs in a daemon thread, stats polled every 1s.
- `backend/server/server4.py` — TCP `0.0.0.0:5000`, per-client thread,
  `handle_client` loop (`recv(1024)` command line, 300s timeout), shared
  `server_stats` dict + lock.
- `backend/server/ExecuteServerCommand.py` — the whole protocol:
  `/ask <name>` → catalog lookup (cutoff 0.2) → unique/exact/strong hit streams
  file (`FOUND <bytes>` + raw bytes), else `MATCHES n` + `name` lines, else
  `libfilesearch` over `shared/`+app folder, else `ERROR ...`.
  `/upload <size> <name>` → `READY`, receive bytes into `received/`, `DONE`,
  catalog the path. `/upload <query>` (no digits) → catalog `MATCHES` search.
  Status codes: 0 ask-streamed, 3 upload-saved, 4 matches-sent, 1 protocol,
  2 storage, -1 logical failure already reported to client.
- `backend/catalog/import_catalog.py` — stdlib sqlite3 at
  `<appDir>/Data/imports.db`, table `imported_files(path UNIQUE, name,
  size_bytes, imported_at)`. `add_imported_path` (no copy, INSERT OR REPLACE),
  `add_imported_folder` (rglob), `find_related(query, cutoff=0.2)` scoring:
  exact=1.0, substring=`max(0.6, len(q)/len(n))`, else
  `difflib.SequenceMatcher.ratio()`. `find_for_request` drops missing-on-disk.
- `backend/usersearch/user_search.py` + `backend/usersearch/find_user.cpp` —
  `search_user_storage(query, limit=200)` over Documents/Downloads/Desktop/
  Pictures/Videos/Music + extra NTFS drives, skipping Windows/Program
  Files/ProgramData/AppData/recycle/system. Native DLL if present, else Python
  `os.walk` fallback. Returns `[{name, path}]`.
- `backend/filesearch/find_exact.cpp` — exact search in `shared/` then app
  folder (replaces `c/file_search.c`, which is dead code).
- `backend/manage_storage/create_app_folder.py` — first run picks drive
  (default `D:\`), creates `<drive>\WayerPC\{shared,received,Data}`, config at
  `%LOCALAPPDATA%\WayerPC\config.json` (`{app_dir, sub_dir}`).
  `backend/manage_storage/bridge_to_app_folder.py` — `obtain_folder_location`,
  `obtain_file_location`, `ensure_shared/received_folder`.
- `packaging/` — PyInstaller folder-build → `dist/WayerPC/WayerPC.exe`.
- `c/` — leftover, unused (ARCHITECTURE.md says deletable).

## 1. Target

```
WayerPC/                       # Wails project root (= repo root)
├── wails.json                 # frameless, vite, version in name
├── version.json               # SINGLE source of truth (edit before build)
├── go.mod / main.go / app.go  # Wails entry + frontend bindings
├── internal/                  # PURE GO backend (no Python/C++)
│   ├── version/  config/  storage/  catalog/  search/  server/
├── frontend/                  # Vite + TS + HTML + CSS (pnpm), frameless
├── build/installer.nsi        # NSIS installer (drive/folder chooser)
├── scripts/                   # sync-version, build-installer, dev helpers
├── .data/                     # dev/test stub (shared/, received/, Data/)
├── doc/                       # THIS plan (you are here)
├── legacy/                    # old Python/C++/spec (reference only)
└── BUILD.md                   # how to dev / build / package
```

Non-goals: Android client changes, protocol breaking changes, copying the
CustomTkinter look pixel-for-pixel (frontend is a redesign, frameless).

## 2. Step order (each step = one commit)

1. `doc/` plan (this folder) — commit `docs: add Go+Wails rewrite plan`.
2. `legacy/` archive (`git mv backend/frontend/c/CMakeLists/requirements…`)
   — commit `chore: archive legacy Python/C++ implementation`.
3. `version.json` + `internal/version` + sync script — commit `feat: version
   single-source-of-truth`.
4. `go.mod` + `internal/{config,storage,catalog,search,server}` pure-Go
   backend + `go test ./...` — commit(s) `feat(backend): …`.
5. `main.go` + `app.go` Wails bindings + `wails.json` — commit
   `feat: wails shell + frameless config`.
6. `frontend/` Vite+TS frameless UI (Console/Import/Library/Settings,
   titlebar, dark theme) — commit(s) `feat(frontend): …`.
7. `.data/` stub + dev fallback (`WAYERPC_DATA_DIR`) — commit
   `chore: add .data dev stub`.
8. NSIS installer + PATH/Start-Menu/terminal support — commit
   `feat(installer): nsis with drive+folder choice`.
9. `BUILD.md` + root README pointer — commit `docs: build instructions`.

See `doc/07-build-test-release.md` for commands and `doc/PLAN-checklist.md`
for the exact commit-per-step list.
