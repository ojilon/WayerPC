# 025 Persist recent logs across restarts

| Field | Value |
|-------|-------|
| Status | `todo` (change to `doing` / `done` as you work) |
| Priority | P2 |
| Size | M (1-3 days) |
| Area | backend |
| Phase | Week 4 — backend depth |
| Related | `076`, `077` |

## 1. Why this matters

Bug reports die on restart: the in-memory ring (`app.go:72,179-192`, 2000 lines) vanishes. The READY-timeout field bug needed the user's log — make that frictionless and durable.

Read this file, then touch the code. This issue is deliberately scoped so you can
finish it in one sitting and see the result in the running app.

## 2. Current behavior

`logs []string` capped at 2000, served by `GetLogs`, cleared by `ClearLogs`. Nothing touches disk. Console replays on mount (`console.ts:111`) but only within a session.

Key files:

- `app.go` (Wails bridge) + `internal/` packages — see issue body for exact paths.

## 3. Problems / symptoms

- 'Send me your log' means 'reproduce the bug first'.
- Restart to update wipes the evidence.
- No rotation policy discussion at all.

## 4. Proposal (what "good" looks like)

Append each log line to `Data/logs.jsonl` (one JSON object per line: ts, level, msg); keep last ~2000 lines in memory as today; rotate by size (e.g. 5MB → `.1`). Console gains 'Download logs' (pairs with `056` copy button).

## 5. Step-by-step implementation

1. Add `internal/logstore` (or extend `app.go:logf`): tee to file under `store.DataDir()`.
2. Open/rotate on `openRoot`; handle write errors by degrading to memory-only (never crash for logs).
3. Load tail (last 500) on startup into the ring.
4. Size rotation: check on open + every N lines.
5. Frontend: download-as-file button in Console.
6. Privacy note: logs contain filenames — document before sharing.

Work in small commits. After each step, run the relevant check:

```powershell
go build ./...; if ($?) { go test ./... }
cmd /c "pnpm.cmd --dir frontend build"
```

## 6. Acceptance criteria

- [ ] Kill + restart → previous session's last lines present in Console.
- [ ] 10MB log spam rotates; disk bounded.
- [ ] Log-write failure never breaks transfers (test by read-only Data dir).

## 7. Tests to add / update

`TestLogPersistence` (log, reopen, tail matches).

## 8. Risks and rollback

Disk-full from logs — rotation + cap mitigates. Filenames in logs are PII — warn in UI before sharing.

Rollback: `git stash` or revert the single commit for this issue. Nothing here
touches the wire protocol unless the header says *(breaking)* — and none of the
early-phase issues do.

## 9. Follow-ups (do NOT do in this issue)

`056` (copy/download buttons), `076` (levels make persisted logs queryable).

## 10. Touch-and-learn notes

- Read the linked files first, top to bottom, before editing.
- Note one thing that surprised you in the commit message.
- If the step takes > M (1-3 days) estimate, stop and split it — open a follow-up note.
