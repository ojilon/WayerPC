# 017 Transfer speed + ETA in Console

| Field | Value |
|-------|-------|
| Status | `todo` (change to `doing` / `done` as you work) |
| Priority | P1 |
| Size | S (<1 day) |
| Area | backend/frontend |
| Phase | Week 3 — backend reliability |
| Related | `007`, `069` |

## 1. Why this matters

Big uploads look hung: bytes counters move but no rate. Bytes-delta per stats tick is almost free given `statsLoop` (`app.go:164-175`) already polls every second.

Read this file, then touch the code. This issue is deliberately scoped so you can
finish it in one sitting and see the result in the running app.

## 2. Current behavior

`Stats` has cumulative `bytesSent/bytesReceived` (`server.go:63-73`); Console shows totals in MB (`console.ts:73-74`). No deltas, no per-transfer state, no ETA. `sendFile`/`recvExact` chunks (32KiB) don't report progress.

Key files:

- `app.go` (Wails bridge) + `internal/` packages — see issue body for exact paths.

## 3. Problems / symptoms

- Users kill healthy slow transfers thinking they're stuck.
- No signal to distinguish 'slow hotspot' from 'stalled'.
- Resume (`008`) needs rate context to be understandable.

## 4. Proposal (what "good" looks like)

Compute per-second deltas client-side (Console keeps last sample) or server-side (add `sentPerSec/recvPerSec` to `Stats`). Show `↑/↓ MB/s` + ETA when a transfer is active (detect via active>0 + delta>0). Almost-free version first; per-transfer progress events later (`065`).

## 5. Step-by-step implementation

1. Decide: client-side delta (zero backend change, preferred for this issue).
2. In `console.ts` keep last `bytesSent/bytesReceived` + timestamp; render rate next to totals.
3. Show ETA only when size known (uploads: from `/upload <size>` — needs active-transfer size exposure; else rate-only).
4. Smooth with 3-sample average to avoid jitter.
5. Test manually: throttled upload shows stable rate.

Work in small commits. After each step, run the relevant check:

```powershell
go build ./...; if ($?) { go test ./... }
cmd /c "pnpm.cmd --dir frontend build"
```

## 6. Acceptance criteria

- [ ] Active upload shows `↓ 3.2 MB/s`; idle shows nothing extra.
- [ ] No backend protocol change.
- [ ] Rate drops to zero visibly on stall (then `021` timeout fires).

## 7. Tests to add / update

Manual + optional frontend delta unit test (pure function).

## 8. Risks and rollback

Jitter on fast small files — hide rate for sub-second transfers.

Rollback: `git stash` or revert the single commit for this issue. Nothing here
touches the wire protocol unless the header says *(breaking)* — and none of the
early-phase issues do.

## 9. Follow-ups (do NOT do in this issue)

`065` (per-transfer progress bars), `078` (metrics history).

## 10. Touch-and-learn notes

- Read the linked files first, top to bottom, before editing.
- Note one thing that surprised you in the commit message.
- If the step takes > S (<1 day) estimate, stop and split it — open a follow-up note.
