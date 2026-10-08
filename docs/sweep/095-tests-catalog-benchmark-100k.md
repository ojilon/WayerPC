# 095 Catalog benchmark: seed 100k, measure FindRelated

| Field | Value |
|-------|-------|
| Status | `todo` (change to `doing` / `done` as you work) |
| Priority | P2 |
| Size | M (1-3 days) |
| Area | tests/backend |
| Phase | Week 6 — tests before optimize |
| Related | `027`, `089` |

## 1. Why this matters

`027` FTS5 and `089` Zig gates both need numbers: `FindRelated` p50/p99 at 1k/10k/100k with the CURRENT code. Decide cutoff/index policy with data, not vibes. This MUST precede `027` (gate noted there).

Read this file, then touch the code. This issue is deliberately scoped so you can
finish it in one sitting and see the result in the running app.

## 2. Current behavior

No bench files for catalog (verify `catalog_test.go` contents); `sortByScore` insertion sort unmeasured; `DefaultCutoff=0.2` unvalidated at scale. Fixture generation (100k rows) doesn't exist.

Key files:

- `app.go` (Wails bridge) + `internal/` packages — see issue body for exact paths.

## 3. Problems / symptoms

- 'Sloggy past 100k' is a guess with no p99.
- FTS5 (`027`) scope can't be sized without the knee location.
- Regression risk: future ranking tweaks (`038`) with no perf guard.

## 4. Proposal (what "good" looks like)

Bench harness: `BenchmarkFindRelated/{1k,10k,100k}` (seeded names: realistic distribution — extensions, dates, typos) measuring p50/p99 + allocs; query set (exact, substring, typo, no-match); record machine + Go version in output; commit results table in this file's PR. Add a CI perf-guard (warn, not fail, on >2x regression) only after 3 stable runs.

## 5. Step-by-step implementation

1. Fixture generator (seeded rand, realistic names; gitignored DBs, committed generator).
2. Bench queries (fixed set) × sizes; `-benchtime=100x -count=5` for stability.
3. Record: p50/p99, ns/op, allocs/op, machine spec.
4. Publish table (this file updated with numbers) → `027` go/no-go.
5. Keep a `SLOW` guard test (100k p99 < Xms generous bound — fail only on egregious regression).

Work in small commits. After each step, run the relevant check:

```powershell
go build ./...; if ($?) { go test ./... }
cmd /c "pnpm.cmd --dir frontend build"
```

## 6. Acceptance criteria

- [ ] Numbers table merged (1k/10k/100k × 4 query kinds).
- [ ] `027` decision references this table explicitly.
- [ ] Guard test green with 10x headroom (no flaky CI).

## 7. Tests to add / update

Bench + guard. No behavior change (measure-only allowed to close).

## 8. Risks and rollback

Fixture realism determines usefulness — include real-world name shapes (IMG_, invoice, report-final, unicode, long). Machine variance — record, don't compare across machines.

Rollback: `git stash` or revert the single commit for this issue. Nothing here
touches the wire protocol unless the header says *(breaking)* — and none of the
early-phase issues do.

## 9. Follow-ups (do NOT do in this issue)

`027` (consumes), `089` (reuses fixtures for Zig scorer bench).

## 10. Touch-and-learn notes

- Read the linked files first, top to bottom, before editing.
- Note one thing that surprised you in the commit message.
- If the step takes > M (1-3 days) estimate, stop and split it — open a follow-up note.
