# Architecture

OOP is limited to the **frontend** (CustomTkinter frames). Backend modules are functions and data.

```
WayerPC/
├── frontend/                 # UI only (classes allowed)
│   ├── main.py               # window, tabs, server thread
│   ├── theme.py
│   ├── import_panel.py       # ImportTab
│   └── library_panel.py      # LibraryTab (received + catalog)
├── backend/
│   ├── server/               # TCP protocol
│   │   ├── server4.py
│   │   ├── ExecuteServerCommand.py
│   │   ├── ItemImporter.py   # legacy copy helper
│   │   └── Locate.py
│   ├── catalog/
│   │   └── import_catalog.py # sqlite3 catalog (stdlib)
│   ├── usersearch/
│   │   ├── find_user.cpp/.hpp
│   │   └── user_search.py    # ctypes glue + Python fallback
│   ├── filesearch/
│   │   └── find_exact.cpp/.hpp
│   ├── native/
│   │   ├── dll_loader.py     # copy lib*.dll/so into the app folder
│   │   └── catalog_sqlite.cpp/.hpp  # optional, needs amalgamation
│   ├── third_party/sqlite/   # drop sqlite3.c/h here (not committed)
│   └── manage_storage/       # app folder, shared/, received/
├── CMakeLists.txt            # builds all native libs
├── packaging/                # PyInstaller
└── docs/
```

The `c/` tree is leftover C code. It is unused; you can delete it.

## Import catalog

`backend/catalog/import_catalog.py` owns `Data/imports.db`:

```sql
CREATE TABLE imported_files (
  id INTEGER PRIMARY KEY,
  path TEXT NOT NULL UNIQUE,
  name TEXT NOT NULL,
  size_bytes INTEGER,
  imported_at TEXT
);
```

`find_related(query, cutoff=0.2)` scores names with `difflib` (and substring boosts). `/ask` and query-style `/upload` both use it.

## Native load order

`sync_native_libs_to_app_folder()` walks `build/native/` (and the repo) for:

- `libfilesearch.dll|.so`
- `libusersearch.dll|.so`
- `libimportcatalog.dll|.so`

and copies them next to the app folder so a packaged build can still find them after you ship the DLLs beside the exe.
