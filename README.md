# WayerPC

Python desktop server that lets an Android phone pull and push files over a hotspot.

Wayer is two repos:

- **This repo** — PC app (CustomTkinter UI + Python server + optional C++23 native libs)
- **[ojilon/Wayer](https://github.com/ojilon/Wayer)** — Android client

The project is under active development. Protocol basics stay stable; the UI and import catalog are the current focus.

---

## Quick start

```bash
pip install -r requirements.txt
python frontend/main.py
```

The window starts the TCP server on `0.0.0.0:5000`. Connect the phone to the same hotspot and use the Android client.

### Native libs (optional but recommended)

```bash
cmake -S . -B build
cmake --build build --config Release
```

DLLs/SOs land in `build/native/` and are copied into the WayerPC app folder on startup:

| Library | Role |
|---------|------|
| `libfilesearch` | Exact filename search in `shared/` and the app folder |
| `libusersearch` | User-storage search (skips Windows/system/AppData) |
| `libimportcatalog` | Optional native SQLite catalog (needs amalgamation) |

Python still works if a DLL is missing (stdlib sqlite3 + a Python search fallback).

---

## What changed in this generation

- **Tabs** — Console (server + logs) stays open. Import and Received/catalog tabs can be added with **+** and closed.
- **Import** — type a file/folder name, search user storage, tick what to keep. Or **Browse files / folder** (Windows Explorer / portal).
- **SQLite catalog** — imported paths are stored in `Data/imports.db` inside the app folder. Files are **not** copied into `shared/` unless you still want that folder for ad-hoc drops.
- **`/ask`** — looks up the catalog (20%+ name similarity). Unique strong match streams the file; otherwise the phone gets a `MATCHES` list.
- **`/upload <size> <name>`** — still saves into `received/` and then catalogs the path.
- **`/upload <query>`** — (no numeric size) returns related imported names, same 20% rule.

---

## Protocol

### `/ask <filename>`

Phone wants a file from the PC.

1. Query the import catalog (exact name, then ≥ 20% similar names).
2. If one clear hit, respond `FOUND <bytes>` and stream the file.
3. If several related names: `MATCHES <n>` then `name<TAB>path` lines.
4. Fallback: `libfilesearch` over `shared/` and the app folder.

### `/upload <filesize> <filename>`

Phone sends a file. Saved under `received/`. Cataloged so later `/ask` can find it.

### `/upload <query>`

No integer size → catalog search only (related names).

---

## Folders (inside the WayerPC app directory on disk)

| Folder | Purpose |
|--------|---------|
| `shared/` | Optional drop folder still searched by `/ask` |
| `received/` | Files the phone uploaded |
| `Data/imports.db` | SQLite catalog of imported paths |

---

## Documentation

| Doc | Contents |
|-----|----------|
| [README_PC_END.md](README_PC_END.md) | Setup, run, protocol details |
| [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md) | Layout, modules, OOP rule |
| [docs/NATIVE.md](docs/NATIVE.md) | CMake, DLLs, SQLite amalgamation |
| [docs/PACKAGING.md](docs/PACKAGING.md) | PyInstaller / gitignore before a release build |

---

## License

MIT — see [LICENSE](LICENSE).
