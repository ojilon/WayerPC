# 023 SQLite busy-timeout + retry

| Field | Value |
|-------|-------|
| Status | `todo` (change to `doing` / `done` as you work) |
| Priority | P1 |
| Size | S (<1 day) |
| Area | backend |
| Phase | Week 3 — backend reliability |
| Related | `030` |

## 1. Why this matters

WAL is on (`catalog.go:67-71`) but no `busy_timeout` is set. A phone upload racing a UI import can hit `SQLITE_BUSY` — a transient error that currently surfaces as a failure instead of a 5s wait.

Read this file, then touch the code. This issue is deliberately scoped so you can
finish it in one sitting and see the result in the running app.

## 2. Current behavior

`Open` applies `journal_mode=WAL`, `synchronous=NORMAL`, `foreign_keys=ON` — no `busy_timeout`. `DB.mu` serializes Go-side writers, but the upload path (`cat.AddPath` after rename) and UI path share the handle across goroutines; busy can still occur on lock upgrades.

Key files:

- `app.go` (Wails bridge) + `internal/` packages — see issue body for exact paths.

## 3. Problems / symptoms

- Flaky `write failed` / import skipped under concurrent load.
- Users retry manually what a 50ms wait would have fixed.
- No retry: single attempt, error bubbles to UI/phone.

## 4. Proposal (what "good" looks like)

Set `PRAGMA busy_timeout=5000` in `catalog.Open` + retry-once on `SQLITE_BUSY` in `AddPath`/`SetSetting`. Log retries at debug. Small, standard, high-value.

## 5. Step-by-step implementation

1. Add `PRAGMA busy_timeout=5000;` to the pragma list (`catalog.go:67-71`).
2. Add retry helper: on `sqlite3: database is locked` error, sleep 100ms, retry once.
3. Apply to `AddPath`, `AddFolder` inner loop, `SetSetting`, `Remove`.
4. Stress test: parallel `AddPath` x50 + upload-simulated `AddPath` → zero BUSY failures.
5. Document in `doc/06-data-sqlite.md` (or its successor).

Work in small commits. After each step, run the relevant check:

```powershell
go build ./...; if ($?) { go test ./... }
cmd /c "pnpm.cmd --dir frontend build"
```

## 6. Acceptance criteria

- [ ] 50-way parallel import shows zero `database is locked` errors.
- [ ] Pragma verified via `PRAGMA busy_timeout` query in test.
- [ ] No perf regression on single-thread import.

## 7. Tests to add / update

Concurrency stress test with `-race`.

## 8. Risks and rollback

Hides genuine lock contention past 5s — acceptable; longer waits would block the phone. Log retries so masking is visible.

Rollback: `git stash` or revert the single commit for this issue. Nothing here
touches the wire protocol unless the header says *(breaking)* — and none of the
early-phase issues do.

## 9. Follow-ups (do NOT do in this issue)

`030` (integrity), `095` (benchmark under contention).

## 10. Touch-and-learn notes

- Read the linked files first, top to bottom, before editing.
- Note one thing that surprised you in the commit message.
- If the step takes > S (<1 day) estimate, stop and split it — open a follow-up note.
