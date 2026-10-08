# 026 Content-hash dedup for imports and uploads

| Field | Value |
|-------|-------|
| Status | `todo` (change to `doing` / `done` as you work) |
| Priority | P2 |
| Size | M (1-3 days) |
| Area | backend |
| Phase | Week 6 — advanced backend |
| Related | `005`, `033` |

## 1. Why this matters

Re-importing or re-receiving identical bytes today writes/stores duplicates and adds catalog rows. Hotspot time is precious — skip the write when bytes already exist.

Read this file, then touch the code. This issue is deliberately scoped so you can
finish it in one sitting and see the result in the running app.

## 2. Current behavior

No hash column (`catalog.go` schema has path/name/size/imported_at only). `AddPath` inserts per path; same content at two paths = two rows, two phone downloads. `005` checksum computes hashes transiently but stores nothing.

Key files:

- `app.go` (Wails bridge) + `internal/` packages — see issue body for exact paths.

## 3. Problems / symptoms

- Duplicate uploads waste hotspot minutes.
- Catalog grows with redundant rows; MATCHES noisier.
- No primitive for 'do I already have this file?' (`008` resume wants it).

## 4. Proposal (what "good" looks like)

Add `sha256` column (nullable, backfilled lazily); on import/upload compute hash, and when identical bytes exist, link/skip with a clear log (`[DEDUP]`). Start with received/ dedup (highest value), then catalog-wide. Keep both paths registered but flagged — or skip second row; decide explicitly and document.

## 5. Step-by-step implementation

1. Schema migration: `ALTER TABLE imported_files ADD COLUMN sha256 TEXT` + index.
2. Hash on `AddPath` (stream, not whole-read) — reuse `005` helper.
3. Policy: same-hash + same-name → skip with log; same-hash + new-name → add row flagged `duplicate_of`? (decide; simplest: skip + log first, flag later).
4. Upload path: check hash before rename — harder (need temp hash); do import-side first.
5. Benchmark import slowdown on 1k files (hash cost).
6. UI: Files view shows 'duplicate' hint (needs `033` columns).

Work in small commits. After each step, run the relevant check:

```powershell
go build ./...; if ($?) { go test ./... }
cmd /c "pnpm.cmd --dir frontend build"
```

## 6. Acceptance criteria

- [ ] Re-import same file → `skipped` + `[DEDUP]` log, no new row (or flagged dup).
- [ ] Import of 1k files slows <20%.
- [ ] Migration keeps old DBs working (nullable column).

## 7. Tests to add / update

Dedup unit tests (same bytes/diff paths); migration test on legacy DB.

## 8. Risks and rollback

Hash cost on huge files during import — stream + show progress (`065`). Policy choice (skip vs link) needs care; start conservative (skip + log).

Rollback: `git stash` or revert the single commit for this issue. Nothing here
touches the wire protocol unless the header says *(breaking)* — and none of the
early-phase issues do.

## 9. Follow-ups (do NOT do in this issue)

`033` (schema home for the column), `008` (resume uses hashes).

## 10. Touch-and-learn notes

- Read the linked files first, top to bottom, before editing.
- Note one thing that surprised you in the commit message.
- If the step takes > M (1-3 days) estimate, stop and split it — open a follow-up note.
