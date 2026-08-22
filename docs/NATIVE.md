# Native libraries (C++23)

## Build

From the **repository root** (not `backend/`):

```bash
cmake -S . -B build
cmake --build build --config Release
```

Outputs (Windows / Linux):

```
build/native/libfilesearch.dll   (or libfilesearch.so)
build/native/libusersearch.dll
build/native/libimportcatalog.dll   # only if sqlite amalgamation is present
```

CMake 3.25+, C++23. On Windows, `libusersearch` links `shell32`, `ole32`, and `uuid` for known-folder paths.

## SQLite amalgamation (optional)

Python already uses stdlib `sqlite3`. The C++ catalog DLL is extra.

1. Download the **amalgamation** zip from https://www.sqlite.org/download.html
2. Put **only** these files here:

```
backend/third_party/sqlite/sqlite3.c
backend/third_party/sqlite/sqlite3.h
```

3. Re-run CMake. You should see:
   `SQLite amalgamation found — building libimportcatalog`

Do not commit `sqlite3.c` / `sqlite3.h` (gitignored).

## What each lib exports

**libfilesearch**

```c
int search_file(const char* shared, const char* filename,
                const char* project_root, char* out, size_t max_len);
/* 0 found, -1 dir error, -2 not found */
```

**libusersearch**

```c
int search_user_storage(const char* query, char* out_buf, size_t max_len);
int name_similarity_percent(const char* a, const char* b);
```

Skips Windows, Program Files, ProgramData, AppData, recycle bin, `/proc`, etc.

**libimportcatalog** (optional)

```c
int catalog_open(const char* db_path);
int catalog_add(const char* path);
int catalog_find(const char* query, int min_percent, char* out, size_t n);
```

Same `imports.db` schema as Python.
