# 032 Missing-on-disk detection + prune

| Field | Value |
|-------|-------|
| Status | `todo` (change to `doing` / `done` as you work) |
| Priority | P1 |
| Size | S (<1 day) |
| Area | backend/frontend |
| Phase | Week 3 — backend reliability |
| Related | `024`, `028` |

## 1. Why this matters

Imports are paths, not copies — USB unplugged, folder renamed, drive letter changed → catalog fills with ghosts. The Files view already badges `missing on disk` (`library.ts:49,77`) but offers no bulk story.

Read this file, then touch the code. This issue is deliberately scoped so you can
finish it in one sitting and see the result in the running app.

## 2. Current behavior

`FindForRequest` filters vanished files for the phone (`catalog.go:230-242`); `CatalogList` (`app.go:315-330`) stamps `Exists` per row via `pathExists` (`helpers.go:22-25`). No count, no 'remove all missing', no re-check action. `Remove` is one-path-at-a-time (`app.go:378-391`).

Key files:

- `app.go` (Wails bridge) + `internal/` packages — see issue body for exact paths.

## 3. Problems / symptoms

- Ghost rows pollute MATCHES (filtered at request time, but UI still lists them).
- One-by-one removal doesn't scale to 500 ghosts after a drive-letter change.
- No 'relink' path for moved trees.

## 4. Proposal (what "good" looks like)

Add `CatalogMissing()` (paths with Exists=false) + `CatalogPruneMissing()` (bulk delete, returns count, confirm in UI). Files view: filter chip 'show missing only' + 'Remove all missing' button with confirm. Later: relink (out of scope, note it).

## 5. Step-by-step implementation

1. Backend: `Missing()` query (or List+stat) and `PruneMissing()` (single DELETE with stat loop, transactional).
2. Bridge both in `app.go`; emit `catalog-changed`.
3. Files view catalog segment: missing-only toggle + prune button + count in hint line.
4. Confirm dialog text: 'Remove N missing paths? Files stay on disk (they're already gone from here).'
5. Test: 3 rows, delete 2 files, Missing()=2, Prune→1 row left.

Work in small commits. After each step, run the relevant check:

```powershell
go build ./...; if ($?) { go test ./... }
cmd /c "pnpm.cmd --dir frontend build"
```

## 6. Acceptance criteria

- [ ] Badge count + bulk prune work on a fixture with 500 ghosts.
- [ ] Phone `/ask` unaffected (already filters).
- [ ] Prune is undoable only via re-import (document).

## 7. Tests to add / update

`TestPruneMissing` (fixture tree + deletions).

## 8. Risks and rollback

Pruning a temporarily-unplugged USB drive looks like data loss — warn when many missing share one root ('D:\… (412 files) — is the drive unplugged?').

Rollback: `git stash` or revert the single commit for this issue. Nothing here
touches the wire protocol unless the header says *(breaking)* — and none of the
early-phase issues do.

## 9. Follow-ups (do NOT do in this issue)

Relink-moved-tree feature (separate future issue).

## 10. Touch-and-learn notes

- Read the linked files first, top to bottom, before editing.
- Note one thing that surprised you in the commit message.
- If the step takes > S (<1 day) estimate, stop and split it — open a follow-up note.
