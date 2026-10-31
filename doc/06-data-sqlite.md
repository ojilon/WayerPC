# 06 — Data + SQLite

All persistent state lives under **one data root** (`<chosen>\WayerPC\data`
when installed, `.data/` for dev — see `doc/07-build-test-release.md`):

```
<dataRoot>/
├── shared/            # optional drop folder, still served by /ask fallback
├── received/          # phone uploads land here
└── Data/
    └── wayerpc.db     # SQLite (modernc.org/sqlite, WAL mode)
```

## Schema (`internal/catalog/schema.sql`, applied on open)

```sql
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
```

Legacy migration: if `Data/imports.db` exists and `wayerpc.db.imported_files`
is empty, rows are copied over (same schema, `INSERT OR IGNORE`), then the
old file is left untouched.

`settings` keys: `app_version`, `data_root`, `host`, `port`,
`library_view` (last Files segment), `onboarded` (bool).

## `.data/` stub (committed)

```
.data/
├── README.md          # "dev stub — real DB is gitignored"
├── shared/.gitkeep
├── received/.gitkeep
└── Data/.gitkeep
```

`wayerpc.db*` and any real uploads are gitignored. `internal/config`
prefers `WAYERPC_DATA_DIR` (tests + `wails dev` set it to `<repo>/.data`),
so cloning + `go test ./...` never touches `%LOCALAPPDATA%` or `D:`.

## Rules

- Imported items are **paths, not copies** (large files aren't duplicated).
- `FindForRequest` filters missing-on-disk every read; Files view badges
  them instead of failing `/ask`.
- One SQLite writer (mutex in `catalog.DB`); TCP handlers share the handle.
- `VACUUM` on version-upgrade open; pragmas `journal_mode=WAL`,
  `synchronous=NORMAL`, `foreign_keys=ON`.
