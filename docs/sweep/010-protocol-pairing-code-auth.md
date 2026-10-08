# 010 Pairing code auth (/auth)

| Field | Value |
|-------|-------|
| Status | `todo` (change to `doing` / `done` as you work) |
| Priority | P2 |
| Size | S (<1 day) |
| Area | protocol/security |
| Phase | Week 5 — security |
| Related | `080`, `016` |

## 1. Why this matters

Hotspots are open networks: anyone joined can push files today. A 6-digit code shown in Console, sent once per install (`/auth <code>`), is a big trust win for small protocol cost.

Read this file, then touch the code. This issue is deliberately scoped so you can
finish it in one sitting and see the result in the running app.

## 2. Current behavior

No auth anywhere: `serveConn` accepts all commands from any peer (`server.go:147-161`). Console (`frontend/src/views/console.ts`) has Start/Stop but no secret. `SUGGESTIONS-v0.2.1.md` §1 marks this P2/S breaking.

Key files:

- `app.go` (Wails bridge) + `internal/` packages — see issue body for exact paths.

## 3. Problems / symptoms

- Any device on the hotspot can `/upload` into `received/` and `/ask` for cataloged files — first connect equals full access.
- No pairing ceremony, no per-device identity, no revocation: a lost phone cannot be un-paired.
- QR/manual-IP onboarding (`015`/`016`) hands out the address with no accompanying secret.

## 4. Proposal (what "good" looks like)

Server generates/stores a 6-digit code (persist in catalog `settings` table); Console displays it + QR (`016`); phone sends `/auth <code>` once, server remembers the device token. Gate file commands on authed state. Keep `/hello`+`/ping` open pre-auth.

## 5. Step-by-step implementation

1. Add `pairing_code` to settings kv (`catalog.go:SetSetting/GetSetting`).
2. Add `/auth <code>` branch; track authed state per conn (simple map or token).
3. Show code in Console hero + Settings; add Regenerate button.
4. Tests: unauthed `/ask` → ERROR auth_required; authed flows pass.
5. Coordinate Android: store token, send on connect.
6. Decide breaking posture: warn-only mode first (log, don't block), enforce later.

Work in small commits. After each step, run the relevant check:

```powershell
go build ./...; if ($?) { go test ./... }
cmd /c "pnpm.cmd --dir frontend build"
```

## 6. Acceptance criteria

- [ ] Stranger without code cannot upload (gets clear ERROR).
- [ ] Paired phone works across restarts without re-entering.
- [ ] Code regeneration invalidates old tokens.

## 7. Tests to add / update

Auth matrix tests; manual hotspot test with two phones.

## 8. Risks and rollback

Lockout risk if code lost — provide Regenerate + local-only reset path. Breaking: roll out warn-then-enforce.

Rollback: `git stash` or revert the single commit for this issue. Nothing here
touches the wire protocol unless the header says *(breaking)* — and none of the
early-phase issues do.

## 9. Follow-ups (do NOT do in this issue)

`080` (trust model doc), `016` (QR carries the code).

## 10. Touch-and-learn notes

- Read the linked files first, top to bottom, before editing.
- Note one thing that surprised you in the commit message.
- If the step takes > S (<1 day) estimate, stop and split it — open a follow-up note.
