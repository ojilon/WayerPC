# 024 shared/ auto-catalog watcher

| Field | Value |
|-------|-------|
| Status | `todo` (change to `doing` / `done` as you work) |
| Priority | P2 |
| Size | M (1-3 days) |
| Area | backend |
| Phase | Week 4 — backend depth |
| Related | `032`, `046` |

## 1. Why this matters

Files dropped into `shared/` are servable (`SearchAppFolders`) but invisible in the catalog UI. The drop folder feels dead — users ask 'where did my file go?'.

Read this file, then touch the code. This issue is deliberately scoped so you can
finish it in one sitting and see the result in the running app.

## 2. Current behavior

No filesystem watcher. `ListShared` (`app.go:399-401`) lists; catalog knows nothing until manual import. `SUGGESTIONS-v0.2.1.md` §2 suggests `fsnotify` + debounced auto-`AddPath`.

Key files:

- `app.go` (Wails bridge) + `internal/` packages — see issue body for exact paths.

## 3. Problems / symptoms

- Two sources of truth (folder vs catalog) confuse the Files view.
- Manual re-import after every drop is friction.
- Bulk drops (unzip 500 files) with no feedback.

## 4. Proposal (what "good" looks like)

`fsnotify` watcher on `shared/` (and optionally `received/`): create → debounced auto-`AddPath`; delete → mark missing (don't auto-remove — see `032`). Toast + `catalog-changed` emit on batch. Toggle in Settings.

## 5. Step-by-step implementation

1. Add `fsnotify` dep (or stdlib polling fallback if dep weight is a concern).
2. Watch `store.SharedDir()` from `openRoot` (`app.go:109-148`); re-arm on relocate.
3. Debounce 500ms per path; batch-commit with counts log `[WATCH] Cataloged N new file(s) in shared/`.
4. Emit `catalog-changed`; toast in Files view.
5. Settings toggle (default ON for shared, OFF for received).
6. Test: drop 3 files → catalog rows appear within 2s; delete → `exists=false` badge (`library.ts:77`).

Work in small commits. After each step, run the relevant check:

```powershell
go build ./...; if ($?) { go test ./... }
cmd /c "pnpm.cmd --dir frontend build"
```

## 6. Acceptance criteria

- [ ] Drop file → appears in Imported catalog without clicks.
- [ ] 500-file unzip handled as one batch log, no 500 toasts.
- [ ] Toggle persists; relocate re-arms watcher.

## 7. Tests to add / update

Watcher integration test with temp dir (create → poll catalog).

## 8. Risks and rollback

Watcher storms (AV scanners, temp files) — debounce + ignore `*.part`, `*.tmp`, dotfiles.

Rollback: `git stash` or revert the single commit for this issue. Nothing here
touches the wire protocol unless the header says *(breaking)* — and none of the
early-phase issues do.

## 9. Follow-ups (do NOT do in this issue)

`032` (prune policy for watcher-deleted files).

## 10. Touch-and-learn notes

- Read the linked files first, top to bottom, before editing.
- Note one thing that surprised you in the commit message.
- If the step takes > M (1-3 days) estimate, stop and split it — open a follow-up note.
