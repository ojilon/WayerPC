# PLAN checklist — commit per step

- [x] Step 0 — `doc/00–07` plan (this folder). Commit: `docs: add Go+Wails rewrite plan`.
- [ ] Step 1 — Archive legacy (`git mv backend legacy/backend`,
      `git mv frontend legacy/frontend-py`, `git mv c legacy/c`,
      `git mv CMakeLists.txt legacy/`, `git mv requirements.txt legacy/`,
      keep `docs/` history). Commit: `chore: archive legacy Python/C++ implementation`.
- [ ] Step 2 — `version.json` + `internal/version` + `scripts/sync-version.mjs`.
      Commit: `feat: version single-source-of-truth`.
- [ ] Step 3 — `internal/config` + `internal/storage` (+ tests).
      Commit: `feat(backend): config and storage with data-root`.
- [ ] Step 4 — `internal/catalog` (SQLite) + `internal/search` (+ tests).
      Commit: `feat(backend): sqlite catalog and user/app search`.
- [ ] Step 5 — `internal/server` (TCP protocol + stats) (+ tests).
      Commit: `feat(backend): tcp server with ask/upload protocol`.
- [ ] Step 6 — `main.go` + `app.go` + `wails.json` + icons.
      Commit: `feat: wails shell with frameless config and bindings`.
- [ ] Step 7 — `frontend/` Vite+TS+pnpn frameless UI.
      Commit: `feat(frontend): frameless console/import/files/settings ui`.
- [ ] Step 8 — `.data/` stub + `.gitignore` update + `go test ./...` green.
      Commit: `chore: add .data dev stub`.
- [ ] Step 9 — `build/installer.nsi` + `scripts/build-installer.ps1`.
      Commit: `feat(installer): nsis drive+folder chooser with path+shortcuts`.
- [ ] Step 10 — `BUILD.md` + README pointer. Commit: `docs: build instructions`.
