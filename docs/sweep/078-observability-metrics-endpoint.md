# 078 Local metrics snapshot (no network server)

| Field | Value |
|-------|-------|
| Status | `todo` (change to `doing` / `done` as you work) |
| Priority | P2 |
| Size | M (1-3 days) |
| Area | backend |
| Phase | Week 7 — advanced |
| Related | `017`, `069` |

## 1. Why this matters

Support questions ('is it slow for everyone or just me?') need numbers: transfers/hour, error rates, p50 sizes, resume counts. No new network port (hotspot trust! `080`) — an in-app snapshot (JSON download + Console sparklines) is enough.

Read this file, then touch the code. This issue is deliberately scoped so you can
finish it in one sitting and see the result in the running app.

## 2. Current behavior

`Stats` cumulative counters (`server.go:63-73`) + Console totals. No history (rates are instant deltas `017`), no per-command counts, no error taxonomy counts (`013`), no export. `078` must not open a Prometheus port on an open hotspot — explicit non-goal.

Key files:

- `app.go` (Wails bridge) + `internal/` packages — see issue body for exact paths.

## 3. Problems / symptoms

- "Is it slow for everyone?" is unanswerable: instant counters only, no history.
- The error taxonomy (`013`) exists on paper but nothing counts occurrences — repeat offenders stay invisible.
- Bug reports carry anecdotes ("it felt slow") instead of a downloadable JSON snapshot.

## 4. Proposal (what "good" looks like)

In-memory ring (last 24h, 5-min buckets): `{transfers, bytesUp/Down, errorsByToken, resumes, avgRate}`; Console 'Health' card shows 24h sparkline (CSS bars, no chart lib) + 'Download JSON' (pairs with `056` log export for bug reports). Reset button. No listener, no telemetry upload — local only, documented loudly.

## 5. Step-by-step implementation

1. Metrics recorder in `Server` (bucketed counters, mutex-light: atomics or per-bucket mutex).
2. `App.GetMetrics()` bridge (JSON); Console Health card + download.
3. Count: per-command outcomes (reuse `013` tokens), resumes (`008`), kicks (`069`).
4. Cap memory (288 buckets fixed array, overwrite).
5. Test: simulate 100 transfers → buckets correct; restart → empty (documented ephemeral) or persisted? (decide: ephemeral first).

Work in small commits. After each step, run the relevant check:

```powershell
go build ./...; if ($?) { go test ./... }
cmd /c "pnpm.cmd --dir frontend build"
```

## 6. Acceptance criteria

- [ ] Bug report bundle = logs (`056`) + metrics JSON, both one-click.
- [ ] 24h sparkline renders with no deps.
- [ ] No new listening socket (verify: `netstat` during review).

## 7. Tests to add / update

Bucketing tests (boundary ticks); export-shape test.

## 8. Risks and rollback

Counter contention on hot path — keep increments lock-free-ish (atomic) and bucket rotation rare.

Rollback: `git stash` or revert the single commit for this issue. Nothing here
touches the wire protocol unless the header says *(breaking)* — and none of the
early-phase issues do.

## 9. Follow-ups (do NOT do in this issue)

Persisted history (only if users ask); Android-side metrics note.

## 10. Touch-and-learn notes

- Read the linked files first, top to bottom, before editing.
- Note one thing that surprised you in the commit message.
- If the step takes > M (1-3 days) estimate, stop and split it — open a follow-up note.
