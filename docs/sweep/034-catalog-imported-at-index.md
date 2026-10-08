# 034 Index imported_at + deterministic ordering

| Field | Value |
|-------|-------|
| Status | `todo` (change to `doing` / `done` as you work) |
| Priority | P2 |
| Size | S (<1 day) |
| Area | backend |
| Phase | Week 4 — backend depth |
| Related | `028`, `050` |

## 1. Why this matters

'Newest' sort in Files (`library.ts:144-151`, `storage` dates) and future catalog paging (`028`) need a total order. Today `List()` is `ORDER BY name` (`catalog.go:169-171`) with no date index and nondeterministic ties.

Read this file, then touch the code. This issue is deliberately scoped so you can
finish it in one sitting and see the result in the running app.

## 2. Current behavior

One index: `idx_imported_files_name`. `imported_at` is `datetime('now')` UTC text (second precision — collisions within a batch import). `sortByScore` insertion sort for searches; UI sorts dates as strings (`localeCompare` on ISO-ish text — works but fragile).

Key files:

- `app.go` (Wails bridge) + `internal/` packages — see issue body for exact paths.

## 3. Problems / symptoms

- 'Newest first' on 1k files imported in one batch = arbitrary order.
- Paging (`028`) with nondeterministic order shows duplicates/skips across pages.
- Date-string sorting breaks if format ever varies.

## 4. Proposal (what "good" looks like)

Add `(imported_at, id)` index; order catalog queries by `imported_at DESC, id DESC` where recency matters; keep name ordering where alphabetical matters — but always with `id` tiebreak. Store `imported_at` with millisecond precision going forward.

## 5. Step-by-step implementation

1. Add index + millisecond timestamp format for new rows (keep parsing old format).
2. Add `ListRecent(limit)` used by Files default view + `028` pager.
3. Tiebreak every ORDER BY with `id`.
4. Frontend: prefer `ListRecent` for catalog segment default; keep sort dropdown.
5. Test: insert 10 rows same-second → stable newest-first order.

Work in small commits. After each step, run the relevant check:

```powershell
go build ./...; if ($?) { go test ./... }
cmd /c "pnpm.cmd --dir frontend build"
```

## 6. Acceptance criteria

- [ ] Batch import order is stable and newest-first.
- [ ] Pager never duplicates/skips rows across pages.
- [ ] Old timestamp strings still parse.

## 7. Tests to add / update

Ordering-stability test with same-second inserts.

## 8. Risks and rollback

Changing default List order could surprise — keep `List()` name-ordered; add `ListRecent` separately.

Rollback: `git stash` or revert the single commit for this issue. Nothing here
touches the wire protocol unless the header says *(breaking)* — and none of the
early-phase issues do.

## 9. Follow-ups (do NOT do in this issue)

`028` (pager uses the total order), `050` (storage sorting parity).

## 10. Touch-and-learn notes

- Read the linked files first, top to bottom, before editing.
- Note one thing that surprised you in the commit message.
- If the step takes > S (<1 day) estimate, stop and split it — open a follow-up note.
