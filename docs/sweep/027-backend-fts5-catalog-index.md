# 027 FTS5 catalog index for 100k-row libraries

| Field | Value |
|-------|-------|
| Status | `todo` (change to `doing` / `done` as you work) |
| Priority | P2 |
| Size | L (3+ days) |
| Area | backend |
| Phase | Week 7 — advanced backend |
| Related | `095`, `038` |

## 1. Why this matters

`FindRelated` (`catalog.go:209-227`) is a linear scan with per-name scoring — fine to ~10k rows, sloggy past 100k. Don't optimize blindly: benchmark first (`095`), then add FTS5 for the substring fast-path, keeping fuzzy scoring on top candidates.

Read this file, then touch the code. This issue is deliberately scoped so you can
finish it in one sitting and see the result in the running app.

## 2. Current behavior

`List()` all rows → `Score()` each → `sortByScore` (insertion sort, O(n²) worst). Every `/ask` and query-`/upload` pays this. `idx_imported_files_name` helps exact prefix only, not substring/fuzzy.

Key files:

- `app.go` (Wails bridge) + `internal/` packages — see issue body for exact paths.

## 3. Problems / symptoms

- `/ask` latency grows linearly with library size; phone waits.
- Insertion sort on 100k hits is itself a bottleneck.
- No numbers: nobody knows where the knee is.

## 4. Proposal (what "good" looks like)

After `095` numbers: add FTS5 auxiliary index on `name`; query narrows to top candidates (e.g. 500), then existing `Score()` ranks. Keep `DefaultCutoff=0.2` semantics. Fix `sortByScore` to `sort.Slice` regardless (free win, do first).

## 5. Step-by-step implementation

1. Do `095` benchmark first — record p50/p99 at 1k/10k/100k.
2. Immediate: replace insertion sort with `sort.Slice` (5-line win).
3. Add FTS5 virtual table + triggers (insert/update/delete) in schema.
4. `FindRelated`: FTS `MATCH` prefilter → score top-N → sort.
5. Fallback: if FTS unavailable (old SQLite), linear path stays.
6. Re-benchmark; document cutoff/index policy with numbers.

Work in small commits. After each step, run the relevant check:

```powershell
go build ./...; if ($?) { go test ./... }
cmd /c "pnpm.cmd --dir frontend build"
```

## 6. Acceptance criteria

- [ ] 100k-row `/ask` p99 < 200ms on dev machine (or documented miss + next step).
- [ ] Results identical (or better-ranked) vs linear on a fixture set.
- [ ] modernc.org/sqlite FTS5 availability verified in test.

## 7. Tests to add / update

`095` fixtures double as regression: ranked-output equality test.

## 8. Risks and rollback

modernc.org/sqlite FTS5 compile flags — verify early (spike test before committing to the path). Trigger bugs can desync index — add consistency check (`030`).

Rollback: `git stash` or revert the single commit for this issue. Nothing here
touches the wire protocol unless the header says *(breaking)* — and none of the
early-phase issues do.

## 9. Follow-ups (do NOT do in this issue)

`038` (ranking improvements ride on the fast prefilter).

## 10. Touch-and-learn notes

- Read the linked files first, top to bottom, before editing.
- Note one thing that surprised you in the commit message.
- If the step takes > L (3+ days) estimate, stop and split it — open a follow-up note.
