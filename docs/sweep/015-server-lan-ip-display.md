# 015 Show the LAN IP, not 0.0.0.0

| Field | Value |
|-------|-------|
| Status | `todo` (change to `doing` / `done` as you work) |
| Priority | P1 |
| Size | S (<1 day) |
| Area | backend |
| Phase | Week 3 — backend reliability |
| Related | `016`, `069` |

## 1. Why this matters

`Status.Addr` reports the bind address (`server.go:113`, `config.DefaultHost=0.0.0.0`). `0.0.0.0:5000` is useless to type into a phone. Biggest phone-UX win per line of code in the whole sweep.

Read this file, then touch the code. This issue is deliberately scoped so you can
finish it in one sitting and see the result in the running app.

## 2. Current behavior

`Addr()` formats `host:port` from config; Console prints `Listening on ${st.server.addr}` (`console.ts:70`). No interface enumeration anywhere. Users must run `ipconfig` themselves.

Key files:

- `app.go` (Wails bridge) + `internal/` packages — see issue body for exact paths.

## 3. Problems / symptoms

- Onboarding fails at the first step: 'what address do I type?'.
- Hotspot subnet changes per session; stale addresses linger in screenshots.
- QR (`016`) and phone panel (`069`) both depend on this.

## 4. Proposal (what "good" looks like)

Enumerate interfaces at status time: first private IPv4 (192.168/10/172.16), fallback to hostname, last resort bind addr. Show `that:5000` in Console hero + Settings server card. Refresh on each `GetStatus` (cheap) or on network-change event.

## 5. Step-by-step implementation

1. Add `LanIP()` helper (new file `internal/server/netip.go`): iterate `net.InterfaceAddrs`, prefer private IPv4, skip loopback/down.
2. Extend `Stats` with `lanAddr` (keep `addr` for compat) or compute display-side in `app.go:GetStatus`.
3. Update `console.ts:70` + `settings.ts:67` to show LAN IP prominently, bind addr muted.
4. Handle no-network case gracefully (`—`, not crash).
5. Test with loopback-only CI (assert non-empty fallback, no panic).

Work in small commits. After each step, run the relevant check:

```powershell
go build ./...; if ($?) { go test ./... }
cmd /c "pnpm.cmd --dir frontend build"
```

## 6. Acceptance criteria

- [ ] Console shows `192.168.x.x:5000` on hotspot; `0.0.0.0` gone from hero.
- [ ] No-network machine shows graceful placeholder.
- [ ] Settings and Console agree.

## 7. Tests to add / update

Unit test for private-IP preference with fake addrs (injectable func).

## 8. Risks and rollback

Multiple NICs (WiFi + hotspot + VPN) — picking 'first private' may pick wrong one. Mitigation: list all candidates in tooltip/`069` panel; hotspot subnet heuristic later.

Rollback: `git stash` or revert the single commit for this issue. Nothing here
touches the wire protocol unless the header says *(breaking)* — and none of the
early-phase issues do.

## 9. Follow-ups (do NOT do in this issue)

`016` (QR), `069` (phone panel lists all IPs).

## 10. Touch-and-learn notes

- Read the linked files first, top to bottom, before editing.
- Note one thing that surprised you in the commit message.
- If the step takes > S (<1 day) estimate, stop and split it — open a follow-up note.
