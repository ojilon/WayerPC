# 072 Uninstall: keep-or-remove data choice

| Field | Value |
|-------|-------|
| Status | `todo` (change to `doing` / `done` as you work) |
| Priority | P2 |
| Size | S (<1 day) |
| Area | installer |
| Phase | Week 6 — release |
| Related | `047` |

## 1. Why this matters

Uninstaller always keeps `data/` (good default) but never asks. Users who wanted a clean removal leave gigabytes; users who mis-click Remove lose libraries. Ask explicitly.

Read this file, then touch the code. This issue is deliberately scoped so you can
finish it in one sitting and see the result in the running app.

## 2. Current behavior

NSIS uninstall section (read `installer.nsi`): removes bin/app, keeps data root. No checkbox page. Data-root location varies per install (`075` remembers it via registry).

Key files:

- `app.go` (Wails bridge) + `internal/` packages — see issue body for exact paths.

## 3. Problems / symptoms

- No clean-uninstall path (manual folder hunt).
- No 'your library stays at D:\… (12 GB)' confirmation for keepers.
- Accidental data loss if a future script 'cleans up'.

## 4. Proposal (what "good" looks like)

Uninstaller page: radio Keep (default, shows path+size) vs Remove (type-to-confirm or second confirm for >1GB). Log choice. Keep catalog backup (`031`) suggestion on Remove path ('Back up first?').

## 5. Step-by-step implementation

1. Add MUI page with keep/remove radios + path/size display (read registry root).
2. Default Keep; Remove needs explicit confirm (checkbox 'I understand' when >1GB).
3. Remove path: delete tree + config + profiles (`055`); keep nothing silently.
4. Test matrix in VM: keep → data intact; remove → gone; custom-root respected.
5. Screenshot both for BUILD.md.

Work in small commits. After each step, run the relevant check:

```powershell
go build ./...; if ($?) { go test ./... }
cmd /c "pnpm.cmd --dir frontend build"
```

## 6. Acceptance criteria

- [ ] Default uninstall keeps data and says so with the path.
- [ ] Remove path requires explicit confirm and fully cleans.
- [ ] Custom install dir (`075`) respected by both paths.

## 7. Tests to add / update

Manual VM only (NSIS); record results in the commit message.

## 8. Risks and rollback

Deleting the wrong tree (path confusion) — resolve + display canonical path, refuse empty/root-of-drive without typed confirm.

Rollback: `git stash` or revert the single commit for this issue. Nothing here
touches the wire protocol unless the header says *(breaking)* — and none of the
early-phase issues do.

## 9. Follow-ups (do NOT do in this issue)

Per-profile data cleanup (`055`) lists each profile's usage on this page (future).

## 10. Touch-and-learn notes

- Read the linked files first, top to bottom, before editing.
- Note one thing that surprised you in the commit message.
- If the step takes > S (<1 day) estimate, stop and split it — open a follow-up note.
