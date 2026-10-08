# 069 Connected-phone / link-health panel

| Field | Value |
|-------|-------|
| Status | `todo` (change to `doing` / `done` as you work) |
| Priority | P2 |
| Size | M (1-3 days) |
| Area | frontend/backend |
| Phase | Week 5 — frontend depth |
| Related | `015`, `017`, `006` |

## 1. Why this matters

Console shows server-centric stats (active/total/bytes) but nothing per-phone: which device is connected, how healthy is the link, what did it last do? Field debugging ('is my phone even seen?') needs a device view.

Read this file, then touch the code. This issue is deliberately scoped so you can
finish it in one sitting and see the result in the running app.

## 2. Current behavior

`Snapshot()` has active/total but no peer list (`server.go:176-190`); `serveConn` logs `[CONNECT]` with remote addr (`server.go:211`) but keeps no table. Hello (`006`) would give client versions — unused without a panel. LAN IP (`015`) + rate (`017`) feed it.

Key files:

- `app.go` (Wails bridge) + `internal/` packages — see issue body for exact paths.

## 3. Problems / symptoms

- 'Active links: 2' — which two? phone vs port-scan (`011`)?
- No per-peer last-command / bytes / RTT.
- Support can't ask 'what does the panel say?' — it doesn't exist.

## 4. Proposal (what "good" looks like)

Peer table: track per-conn `{ip, connectedAt, lastCmd, bytesUp/Down, helloVer?}` in Server (bounded 32, evict oldest); `Stats.peers[]` in Snapshot; Console panel lists peers with rate (`017` math), version (`006`), and kick button (close conn). Scan-looking peers (many short conns, `011` trips) flagged subtly.

## 5. Step-by-step implementation

1. Add peer registry in `Server` (mutex-guarded map conn→info, updated per command + chunk).
2. Extend `Snapshot()` (compat: new field, old readers ignore).
3. Console panel UI (table, auto-update via `stats` event, kick button → new `App.KickPeer(ip)`).
4. Hello version column appears when `006` landed (else '—').
5. Cap + evict; ensure -race clean.
6. Privacy: IPs are LAN-local; note in UI ('visible on your hotspot only').

Work in small commits. After each step, run the relevant check:

```powershell
go build ./...; if ($?) { go test ./... }
cmd /c "pnpm.cmd --dir frontend build"
```

## 6. Acceptance criteria

- [ ] Two phones show as two rows with live rates.
- [ ] Kick disconnects the right peer (test with loopback pair).
- [ ] Port-scan burst doesn't grow the table unboundedly.

## 7. Tests to add / update

Peer registry tests (add/update/evict/kick); -race on parallel conns.

## 8. Risks and rollback

Per-chunk updates contend the mutex — update counters atomically-ish (batch per 256KB or per second, not per 32KB write).

Rollback: `git stash` or revert the single commit for this issue. Nothing here
touches the wire protocol unless the header says *(breaking)* — and none of the
early-phase issues do.

## 9. Follow-ups (do NOT do in this issue)

Per-device pairing state display (`010`); bandwidth történelem (`078`).

## 10. Touch-and-learn notes

- Read the linked files first, top to bottom, before editing.
- Note one thing that surprised you in the commit message.
- If the step takes > M (1-3 days) estimate, stop and split it — open a follow-up note.
