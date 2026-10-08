# 011 Per-IP connection cap

| Field | Value |
|-------|-------|
| Status | `todo` (change to `doing` / `done` as you work) |
| Priority | P2 |
| Size | S (<1 day) |
| Area | protocol |
| Phase | Week 2 — protocol correctness |
| Related | `020`, `079` |

## 1. Why this matters

A neighbor port-scanning the hotspot spawns unbounded goroutines (`go s.serveConn(conn)` per Accept, `server.go:159`). One bad actor can exhaust the PC.

Read this file, then touch the code. This issue is deliberately scoped so you can
finish it in one sitting and see the result in the running app.

## 2. Current behavior

No cap: Accept → `active++` → goroutine, unbounded. `Stats.Active` counts but nothing enforces. No per-IP tracking at all.

Key files:

- `app.go` (Wails bridge) + `internal/` packages — see issue body for exact paths.

## 3. Problems / symptoms

- One port-scan spawns unbounded `serveConn` goroutines plus log spam; legit phones starve.
- `Active` is counted but nothing enforces it; no per-IP view exists for diagnosis.
- Half-open connections linger until the 300s idle deadline, holding goroutines and log volume.

## 4. Proposal (what "good" looks like)

Cap concurrent conns per IP (e.g. 8) with `ERROR busy` reply + close. Global cap too (e.g. 64). Log at warn level when tripping so scans are visible in Console.

## 5. Step-by-step implementation

1. Add `map[string]int` per-IP counter guarded by `mu` (increment on accept, decrement on close).
2. Extract IP via `conn.RemoteAddr()` parsing (handle IPv6 zones).
3. On exceed: `sendLine(conn, "ERROR busy")`, close, log `[LIMIT]`.
4. Make limits consts (or config keys for `052`).
5. Test: 9 rapid conns from 127.0.0.1 → 9th gets ERROR busy.
6. Show `Active` + limit in Console hero (`015` synergy).

Work in small commits. After each step, run the relevant check:

```powershell
go build ./...; if ($?) { go test ./... }
cmd /c "pnpm.cmd --dir frontend build"
```

## 6. Acceptance criteria

- [ ] 9th concurrent conn from one IP refused with `ERROR busy`.
- [ ] Legit sequential use unaffected.
- [ ] Log line identifies the capped IP.

## 7. Tests to add / update

Conn-flood unit test with `net.Pipe` or loopback dials.

## 8. Risks and rollback

NAT hotspot: all phones may share one gateway IP in weird tethering setups — pick 8, not 2, and make it configurable.

Rollback: `git stash` or revert the single commit for this issue. Nothing here
touches the wire protocol unless the header says *(breaking)* — and none of the
early-phase issues do.

## 9. Follow-ups (do NOT do in this issue)

`020` (global transfer limits), `079` (audit).

## 10. Touch-and-learn notes

- Read the linked files first, top to bottom, before editing.
- Note one thing that surprised you in the commit message.
- If the step takes > S (<1 day) estimate, stop and split it — open a follow-up note.
