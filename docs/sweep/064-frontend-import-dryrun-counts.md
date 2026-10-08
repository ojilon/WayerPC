# 064 Import dry-run counts with confirm threshold

| Field | Value |
|-------|-------|
| Status | `todo` (change to `doing` / `done` as you work) |
| Priority | P1 |
| Size | S (<1 day) |
| Area | frontend/backend |
| Phase | Week 3 — frontend |
| Related | `029`, `065` |

## 1. Why this matters

Importing a folder with 50k files is a leap of faith (`CatalogAddFolder` returns counts only AFTER, `app.go:360-375`; UI toasts after, `import.ts:112-121`). A fast pre-walk ('≈ N files') + confirm above a threshold (e.g. 1,000) prevents accidents.

Read this file, then touch the code. This issue is deliberately scoped so you can
finish it in one sitting and see the result in the running app.

## 2. Current behavior

No counting: Browse-folder → immediate `AddFolder`. `AddFolder` walks + inserts inline (slow, `029`). Users can't estimate cost. Threshold/confirm pattern exists elsewhere (`ctx.confirm`, `main.ts:27-40`) but unused here.

Key files:

- `app.go` (Wails bridge) + `internal/` packages — see issue body for exact paths.

## 3. Problems / symptoms

- Accidental whole-drive import (user picks D:\ instead of D:\Photos).
- No time estimate; UI appears hung during 50k inserts.
- Undo is manual prune (`032`).

## 4. Proposal (what "good" looks like)

Add `App.CountFiles(folder)` (fast walk, no DB, respects skips `039` + depth `040`); UI shows '≈ N files — Import? (est. Xs)' and confirms when N > threshold (setting, default 1000). Keep one-click for small folders.

## 5. Step-by-step implementation

1. Backend `CountFiles` (walk + count files only, fast, cancelable via `036` ctx).
2. `import.ts` folder flow: count → show → confirm-if-large → `catalogAddFolder`.
3. Estimate text: N files (~Y MB via sampled sizes? keep simple: files only first).
4. Threshold in settings kv (advanced; default 1000).
5. Test: fixture tree count exact; cancel works on huge fixture.

Work in small commits. After each step, run the relevant check:

```powershell
go build ./...; if ($?) { go test ./... }
cmd /c "pnpm.cmd --dir frontend build"
```

## 6. Acceptance criteria

- [ ] Picking 50k folder warns with exact count before any insert.
- [ ] 5-file folder imports with zero extra clicks.
- [ ] Count honors skip rules (node_modules excluded when configured).

## 7. Tests to add / update

`TestCountFiles` on fixtures (incl. skips); threshold-prompt logic test.

## 8. Risks and rollback

Count itself walks the tree (cost) — cap/timebox it ('>20k, showing 20k+') and make it cancelable.

Rollback: `git stash` or revert the single commit for this issue. Nothing here
touches the wire protocol unless the header says *(breaking)* — and none of the
early-phase issues do.

## 9. Follow-ups (do NOT do in this issue)

`029` (fast insert makes the confirm less scary), `065` (progress after confirm).

## 10. Touch-and-learn notes

- Read the linked files first, top to bottom, before editing.
- Note one thing that surprised you in the commit message.
- If the step takes > S (<1 day) estimate, stop and split it — open a follow-up note.
