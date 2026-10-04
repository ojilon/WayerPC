# SQLite amalgamation (optional native catalog)

Python already uses the **stdlib `sqlite3` module** for the import catalog.
You only need these files if you also want CMake to build `libimportcatalog`
(C++ talking to the same `imports.db`).

## Where to put the files

Download the official amalgamation from SQLite and extract **exactly these two
files** into **this folder**:

```
backend/third_party/sqlite/sqlite3.c
backend/third_party/sqlite/sqlite3.h
```

Do **not** commit `sqlite3.c` / `sqlite3.h` (they are large and redistributed
by SQLite). They are gitignored.

## Download

1. Open https://www.sqlite.org/download.html
2. Under **Source Code**, get **sqlite-amalgamation-XXXXXXX.zip**
   (the year/version zip that contains `sqlite3.c` and `sqlite3.h`).
3. Copy those two files here.

Direct pattern (version number changes):

```
https://www.sqlite.org/2024/sqlite-amalgamation-3460100.zip
```

Use the current amalgamation zip listed on the download page.

## Build

From the repo root:

```bash
cmake -S . -B build
cmake --build build --config Release
```

If the amalgamation is present, CMake prints
`SQLite amalgamation found — building libimportcatalog`
and writes `build/native/libimportcatalog.dll` (or `.so`).

WayerPC copies that DLL into the app folder on startup, next to
`libfilesearch` and `libusersearch`.
