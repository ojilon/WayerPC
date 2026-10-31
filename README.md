# WayerPC

Desktop server that lets an Android phone pull and push files over a hotspot.

Wayer is two repos:

- **This repo** — PC app (Go backend + Wails frameless UI, SQLite catalog)
- **[ojilon/Wayer](https://github.com/ojilon/Wayer)** — Android client

The project is under active development. Protocol basics stay stable; the UI and import catalog are the current focus.

---

## Quick start

```powershell
cmd /c "pnpm.cmd --dir frontend install"
cmd /c "pnpm.cmd --dir frontend build"
wails dev   # full app with the .data/ stub; or: wails build -platform windows/amd64 -o WayerPC.exe
```

The window starts the TCP server on `0.0.0.0:5000`. Connect the phone to the same hotspot and use the Android client. Full instructions: **[BUILD.md](BUILD.md)**.

### Installer (Windows)

```powershell
powershell -ExecutionPolicy Bypass -File scripts/build-installer.ps1
```

Pick any drive/folder during setup (e.g. `D:\projects\` → `D:\projects\WayerPC\`).
You get `bin\WayerPC.exe` on the user `PATH` (terminal + Windows search) and
data in `data\` (`shared/`, `received/`, SQLite catalog).

---

## What this generation changed

- **Stack** — pure-Go backend (`internal/`), Wails + Vite + TypeScript frameless frontend (`frontend/`). No Python, no C++/DLLs.
- **Console** — server status, traffic stats, filterable/pausable live log.
- **Import** — search user storage (system paths skipped), tick results, or browse via OS dialogs. Paths are registered, never copied.
- **Files** — Received ⇄ Imported catalog ⇄ Shared segments, sorting, reveal-in-Explorer, missing-on-disk badges.
- **Settings** — choose/relocate the data folder, server address, version. First run shows onboarding when no data folder exists yet.
- **SQLite catalog** — `Data/wayerpc.db` (migrates the old `imports.db` automatically). `/ask` and query-style `/upload` use the 20%-similarity rule.
- **Versioning** — edit `version.json` before building; Go, frontend, and the installer all derive from it (`node scripts/sync-version.mjs`).
- **`.data/`** — committed dev stub so clones build and test without touching real drives.

---

## Protocol

### `/ask <filename>`

Phone wants a file from the PC.

1. Query the import catalog (exact name, then ≥ 20% similar names).
2. If one clear hit, respond `FOUND <bytes>` and stream the file.
3. If several related names: `MATCHES <n>` then one `name` per line.
4. Fallback: exact-name search over `shared/` and `received/`.

### `/upload <filesize> <filename>`

Phone sends a file. Saved under `received/`. Cataloged so later `/ask` can find it.

### `/upload <query>`

No integer size → catalog search only (related imported names, same 20% rule).

---

## Folders (inside the data folder — chosen at install or first run)

| Folder | Purpose |
|--------|---------|
| `shared/` | Optional drop folder still searched by `/ask` |
| `received/` | Files the phone uploaded |
| `Data/wayerpc.db` | SQLite catalog of imported paths + settings |

---

## Documentation

| Doc | Contents |
|-----|----------|
| [BUILD.md](BUILD.md) | Dev, build, installer, release checklist |
| [doc/00-overview.md](doc/00-overview.md) | Rewrite goals + step order |
| [doc/01-frontend-backend-split.md](doc/01-frontend-backend-split.md) | Legacy → new module mapping |
| [doc/02-backend-go.md](doc/02-backend-go.md) | Go packages in depth |
| [doc/03-frontend-wails.md](doc/03-frontend-wails.md) | Frameless UI structure |
| [doc/04-versioning.md](doc/04-versioning.md) | Change-the-version flow |
| [doc/05-installer.md](doc/05-installer.md) | NSIS behavior |
| [doc/06-data-sqlite.md](doc/06-data-sqlite.md) | Data layout + schema |
| [doc/07-build-test-release.md](doc/07-build-test-release.md) | Commands reference |
| [legacy/](legacy/) | Previous Python/C++ implementation (reference only) |

---

## License

MIT — see [LICENSE](LICENSE).
