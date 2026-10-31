# 01 — Frontend / backend split

## Rule

- `internal/**` — **pure Go**. No Python, no C++, no HTML. Owns TCP server,
  SQLite catalog, filesystem search, storage layout, config, logging.
- `frontend/**` — **TS + HTML + CSS only** (Vite, pnpm). No business logic
  beyond rendering: every action calls a Wails-bound `App.*` method or a local
  UI helper. Dev-mode mocks live in `frontend/src/dev-mock.ts` and are never
  bundled into decisions.
- `main.go` / `app.go` are the **only bridge**: Wails binds `App` methods;
  backend pushes async events (`log`, `stats`, `catalog-changed`) via
  `runtime.EventsEmit`. Frontend never opens sockets or SQLite directly.

## Legacy → new mapping

| Legacy (Python)                              | New (Go, `internal/`)                  |
|----------------------------------------------|----------------------------------------|
| `backend/server/server4.py`                  | `internal/server/server.go` (listener, per-conn goroutine, stats) |
| `backend/server/ExecuteServerCommand.py`      | `internal/server/protocol.go` (`/ask`, `/upload`, `MATCHES`/`FOUND`/`READY`/`DONE`/`ERROR`) |
| `backend/server/transfer.py`                 | `internal/server/transfer.go` (stream file, receive with limit) |
| `backend/server/config.py` (`HOST/PORT`)     | `internal/config/config.go` (host/port/data-root, env overrides) |
| `backend/server/Locate.py`, `ItemImporter.py`| folded into `storage` + `search` (no direct port; legacy copy helpers only) |
| `backend/catalog/import_catalog.py`          | `internal/catalog/catalog.go` (modernc.org/sqlite, same table + `settings` kv) |
| `backend/usersearch/` + `find_user.cpp`      | `internal/search/usersearch.go` (pure-Go walk, same skip-list + roots) |
| `backend/filesearch/find_exact.cpp`          | `internal/search/appsearch.go` (walk `<dataRoot>/{shared,received}` + catalog) |
| `backend/native/dll_loader.py`               | **deleted** — no DLLs; pure Go replaces all three native libs |
| `backend/manage_storage/*`                   | `internal/storage/storage.go` + `internal/config` (data-root init, `shared/`, `received/`, `Data/`) |
| `backend/third_party/sqlite/*`, `CMakeLists` | **deleted** — `modernc.org/sqlite` needs no amalgamation/C toolchain |
| `frontend/main.py`, `theme.py`               | `frontend/src/*` (frameless shell, dark theme tokens in CSS, not Python) |
| `frontend/import_panel.py`                   | `frontend/src/views/import.ts` (search → tick → import, browse via Go dialogs) |
| `frontend/library_panel.py`                  | `frontend/src/views/library.ts` (Received / Imported catalog segments) |
| `c/`                                         | **deleted** (was already dead code) |

## Data flow

```
Android phone ──TCP:5000──▶ internal/server (Go)
                                ├─ catalog (SQLite wayerpc.db)
                                └─ storage (shared/, received/, Data/)
                                        ▲
Wails App bindings (app.go) ────────────┘
        ▲  EventsEmit(log/stats/catalog-changed)
        ▼
frontend (TS) ── window.go.main.App.* ──▶ user (frameless window)
```

## Testing seams

- `go test ./internal/...` covers scoring/cutoff, protocol branches
  (`FOUND`/`MATCHES`/`READY`/`DONE`/`ERROR`), catalog CRUD, storage init —
  using `t.TempDir()`, never the real drive.
- Frontend `pnpm build` (tsc + vite) must pass with mocked
  `window.go` absent (browser dev mode).
