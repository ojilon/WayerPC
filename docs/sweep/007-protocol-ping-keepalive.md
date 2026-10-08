# 007 /ping keepalive probe

| Field | Value |
|-------|-------|
| Status | `todo` (change to `doing` / `done` as you work) |
| Priority | P1 |
| Size | S (<1 day) |
| Area | protocol |
| Phase | Week 2 — protocol correctness |
| Related | `006`, `017` |

## 1. Why this matters

The phone discovers a dead/stale socket only after committing to a 200MB upload. A cheap `/ping` → `PONG` lets it probe a reused socket first.

Read this file, then touch the code. This issue is deliberately scoped so you can
finish it in one sitting and see the result in the running app.

## 2. Current behavior

No keepalive command. `serveConn` (`server.go:204-260`) sets 300s idle + 60s write deadlines; a half-dead peer is detected only when a real command's reply fails (`fatal` path). Console shows no per-peer liveness.

Key files:

- `app.go` (Wails bridge) + `internal/` packages — see issue body for exact paths.

## 3. Problems / symptoms

- Wasted user time starting doomed transfers.
- Reconnect storms after hotspot hiccups with no lightweight probe.
- No latency signal for the speed/ETA feature (`017`).

## 4. Proposal (what "good" looks like)

Add `/ping [<nonce>]` → `PONG [<nonce>]`. Stateless, no logging above debug. Phone sends before big uploads and on reconnect; measures RTT for future ETA math.

## 5. Step-by-step implementation

1. Add `ping` branch in `handleCommand`; echo nonce if present, else bare `PONG`.
2. Keep it out of `[ASK OK]`-style info logs (debug only) to avoid log spam.
3. Test: `/ping`, `/ping abc123`, `/ping` with no newline (settle path).
4. Document alongside `006` hello.
5. Android: probe before uploads >50MB; surface 'server unreachable' early.
6. Optionally expose `lastRTT` in `Snapshot()` for `017`.

Work in small commits. After each step, run the relevant check:

```powershell
go build ./...; if ($?) { go test ./... }
cmd /c "pnpm.cmd --dir frontend build"
```

## 6. Acceptance criteria

- [ ] `/ping 42` → `PONG 42` within one RTT.
- [ ] No info-log line per ping (log stays quiet).
- [ ] Stale-socket detection demo: kill server, ping fails fast.

## 7. Tests to add / update

`TestPingEcho`; conformance ping case.

## 8. Risks and rollback

None protocol-wise. Log-spam risk handled by debug-level logging.

Rollback: `git stash` or revert the single commit for this issue. Nothing here
touches the wire protocol unless the header says *(breaking)* — and none of the
early-phase issues do.

## 9. Follow-ups (do NOT do in this issue)

`017` (speed+ETA uses RTT), `069` (phone panel shows liveness).

## 10. Touch-and-learn notes

- Read the linked files first, top to bottom, before editing.
- Note one thing that surprised you in the commit message.
- If the step takes > S (<1 day) estimate, stop and split it — open a follow-up note.
