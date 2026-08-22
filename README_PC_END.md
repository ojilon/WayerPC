# WayerPC — PC end guide

## Requirements

- Python 3.11+ (3.14 is fine)
- `pip install -r requirements.txt` (`customtkinter`; `pyinstaller` only for packaging)
- Optional native build: CMake 3.25+, a C++23 compiler (MSVC, MinGW, or GCC)

## Run

From the repository root:

```bash
python frontend/main.py
```

On first launch you pick a drive (default `D:\`). WayerPC creates:

```
<drive>\WayerPC\
  shared\
  received\
  Data\
    imports.db
  libfilesearch.dll   (copied when present)
  libusersearch.dll
  libimportcatalog.dll
```

Config lives at `%LOCALAPPDATA%\WayerPC\config.json`.

## UI

- **Console** — server status, traffic, logs. Cannot be closed.
- **Import** — search by file/folder name across Documents/Downloads/Desktop/extra drives. System paths and AppData are skipped. Tick results and **Import selected**, or use **Browse files** / **Browse folder**.
- **Received** — files in `received/` plus the **Imported catalog** segment.
- **+** — open another Import or View-files tab. **Close tab** removes extra tabs only.

Imported items are **paths in SQLite**, not copies. That avoids duplicating large files into `shared/`.

## Protocol (phone → PC, TCP port 5000)

```
/ask report.pdf
    → FOUND 12345\n + raw bytes
    → or MATCHES 3\nname\tpath\n...
    → or ERROR file_not_found

/upload 12345 photo.jpg
    → READY  then the client sends 12345 bytes
    → DONE

/upload photo
    → MATCHES n  (related imported names, ≥ 20% similar)
```

## Building native libraries

See [docs/NATIVE.md](docs/NATIVE.md).

```bash
cmake -S . -B build
cmake --build build --config Release
```

## Packaging a Windows app

See [docs/PACKAGING.md](docs/PACKAGING.md). Short version:

```powershell
powershell -ExecutionPolicy Bypass -File packaging/build_windows.ps1
```

Output: `dist/WayerPC/WayerPC.exe`.

## Troubleshooting

| Issue | What to check |
|-------|----------------|
| Import search empty | Native `libusersearch` missing is OK (Python fallback). Query may be too specific, or files sit under AppData (skipped on purpose). |
| `/ask` not found | File must be in the catalog or in `shared/`. Open the Received tab → Imported catalog. |
| DLL not found | Build native libs, restart the app so they copy into the app folder. |
| Port in use | Change `HOST`/`PORT` in `backend/server/server4.py` (and `config.py`). |
| `io.UnsupportedOperation` on config | Fixed: folder helpers no longer overwrite `config.json` read-only. Pull latest. |
