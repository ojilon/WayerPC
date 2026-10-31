# legacy — pre-Go implementation (reference only)

Archived 2026-10-04 during the Go + Wails rewrite (see `doc/`).

- `backend/` — Python TCP server, SQLite catalog, storage helpers + C++23
  native libs (`filesearch`, `usersearch`, optional `importcatalog`).
- `frontend-py/` — CustomTkinter desktop UI.
- `c/` — dead C search helper (was already unused).
- `CMakeLists.txt`, `requirements.txt`, `build.bat`, `zipper.bat`,
  `packaging/` — old CMake / pip / PyInstaller flow.

Nothing under `legacy/` is built or shipped anymore. The active code is:

- `internal/` — pure-Go backend,
- `frontend/` — Vite + TypeScript Wails UI,
- `build/installer.nsi` — Windows installer.
