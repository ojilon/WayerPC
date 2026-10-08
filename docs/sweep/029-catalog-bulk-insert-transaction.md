# 029 Bulk import inside one transaction

| Field | Value |
|-------|-------|
| Status | `todo` (change to `doing` / `done` as you work) |
| Priority | P1 |
| Size | S (<1 day) |
| Area | backend |
| Phase | Week 3 — backend reliability |
| Related | `064`, `065` |

## 1. Why this matters

`AddFolder` (`catalog.go:120-137`) calls `AddPath` per file, each its own `INSERT` + mutex + fsync-ish commit. Importing 50k files is thousands of transactions — minutes instead of seconds.

Read this file, then touch the code. This issue is deliberately scoped so you can
finish it in one sitting and see the result in the running app.

## 2. Current behavior

Per-file `INSERT OR REPLACE` with `d.mu` held each time; `filepath.WalkDir` callback inserts inline. `CatalogAddFolder` (`app.go:360-375`) reports counts but gives no progress. WAL helps; batching helps 10x more.

Key files:

- `app.go` (Wails bridge) + `internal/` packages — see issue body for exact paths.

## 3. Problems / symptoms

- Folder import of large trees is painfully slow.
- Partial failure leaves half-committed state with no resume marker.
- UI blocked with no feedback during the long insert.

## 4. Proposal (what "good" looks like)

Wrap `AddFolder` (and multi-`AddPath`) in a single transaction (or 1k-row batches): BEGIN → inserts → COMMIT. Keep per-file skip counting; on error ROLLBACK the batch, report counts. Combine with `065` progress events.

## 5. Step-by-step implementation

1. Refactor `AddPath` core into `insertLocked(abs, size)` usable inside/outside txn.
2. `AddFolder`: BEGIN; batch COMMIT every 1000 rows; final COMMIT.
3. Preserve `added/skipped` semantics; collect first error but continue batch.
4. Benchmark: 5k-file fixture before/after (expect ≥5x).
5. Ensure `023` busy-timeout + txn interact sanely (no nested BEGIN).

Work in small commits. After each step, run the relevant check:

```powershell
go build ./...; if ($?) { go test ./... }
cmd /c "pnpm.cmd --dir frontend build"
```

## 6. Acceptance criteria

- [ ] 5k-file import ≥5x faster (record numbers).
- [ ] Counts identical to old path on a fixture tree.
- [ ] Error mid-folder doesn't corrupt DB (integrity check passes).

## 7. Tests to add / update

Bulk-import test (temp tree, counts + timing log); txn-rollback test (inject bad row).

## 8. Risks and rollback

Long write txn blocks readers (WAL mitigates; batching bounds it). Keep batches ≤1000.

Rollback: `git stash` or revert the single commit for this issue. Nothing here
touches the wire protocol unless the header says *(breaking)* — and none of the
early-phase issues do.

## 9. Follow-ups (do NOT do in this issue)

`064` (dry-run counts), `065` (progress per batch).

## 10. Touch-and-learn notes

- Read the linked files first, top to bottom, before editing.
- Note one thing that surprised you in the commit message.
- If the step takes > S (<1 day) estimate, stop and split it — open a follow-up note.
