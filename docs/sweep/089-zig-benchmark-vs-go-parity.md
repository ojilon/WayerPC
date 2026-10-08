# 089 Zig plan 9/10: benchmark gates + parity harness

| Field | Value |
|-------|-------|
| Status | `todo` (change to `doing` / `done` as you work) |
| Priority | P1 |
| Size | M (1-3 days) |
| Area | zig/tests |
| Phase | Week 8 — Zig track |
| Related | `081`, `095` |

## 1. Why this matters

`081` set the bar (≥2x faster or ≥5x less-alloc on one candidate) — this builds the harness that judges it: same-machine benches, allocation counters, and the parity oracle all three ports share. No harness, no honest decision.

Read this file, then touch the code. This issue is deliberately scoped so you can
finish it in one sitting and see the result in the running app.

## 2. Current behavior

`095` catalog bench (Go-side) exists or is planned; no Zig harness; no alloc-counting habit. Bench numbers so far live in commit messages (fine) but aren't comparable across machines/runs.

Key files:

- `app.go` (Wails bridge) + `internal/` packages — see issue body for exact paths.

## 3. Problems / symptoms

- Apples-to-oranges benches (different fixtures, warm vs cold, debug vs ReleaseFast vs ReleaseSmall).
- Parity checked ad-hoc per port instead of once centrally.
- No 'stay Go' exit criteria written before results (bias toward shipping the port).

## 4. Proposal (what "good" looks like)

Harness: `bench/` fixtures (100k names, 50k tree, 1GB file — generated, gitignored, seeded), `bench.sh` (Go bench + Zig `ReleaseFast` bench, 5 runs, median, machine info header), parity oracle (Go output = truth, Zig must match within epsilon/exact as specified per port), gate table per port (ship bar + stay-Go bar). Record everything; decide per port independently.

## 5. Step-by-step implementation

1. Build fixture generators (seeded, deterministic).
2. Bench scripts (both sides, ReleaseFast + also ReleaseSmall note for size).
3. Parity oracle runner (goldens from `082`/`083`/`084` executed in one command).
4. Gate table filled per port (numbers + ship/stay verdict + date).
5. Commit the harness + results (numbers in repo, not just chat).
6. Feed verdicts into `090` flag defaults.

Work in small commits. After each step, run the relevant check:

```powershell
go build ./...; if ($?) { go test ./... }
cmd /c "pnpm.cmd --dir frontend build"
```

## 6. Acceptance criteria

- [ ] One command reproduces all numbers on a fresh clone (with zig installed).
- [ ] Each port has a dated ship/stay verdict.
- [ ] Stay-Go verdicts documented as success (no stigma).

## 7. Tests to add / update

Harness self-test (tiny fixtures, both sides agree).

## 8. Risks and rollback

Benchmark theater (optimizing the harness, not the app) — fixtures must mirror real shapes (real filename distributions, real tree depths from YOUR library).

Rollback: `git stash` or revert the single commit for this issue. Nothing here
touches the wire protocol unless the header says *(breaking)* — and none of the
early-phase issues do.

## 9. Follow-ups (do NOT do in this issue)

`090` (verdicts set flag defaults), `095` (Go-side numbers shared).

## 10. Touch-and-learn notes

- Read the linked files first, top to bottom, before editing.
- Note one thing that surprised you in the commit message.
- If the step takes > M (1-3 days) estimate, stop and split it — open a follow-up note.
