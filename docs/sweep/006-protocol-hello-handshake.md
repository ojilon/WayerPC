# 006 /hello handshake with versions

| Field | Value |
|-------|-------|
| Status | `todo` (change to `doing` / `done` as you work) |
| Priority | P1 |
| Size | M (1-3 days) |
| Area | protocol |
| Phase | Week 2 — protocol correctness |
| Related | `004`, `007`, `013` |

## 1. Why this matters

Version mismatches today surface as cryptic mid-transfer failures. A handshake (`/hello <client-version>` → `WAYERPC <server-version> <proto>`) turns them into one clear message at connect time.

Read this file, then touch the code. This issue is deliberately scoped so you can
finish it in one sitting and see the result in the running app.

## 2. Current behavior

`handleCommand` (`protocol.go:19-43`) knows `/ask` and `/upload` only; anything else is `ERROR unknown_protocol_command.` `version.Current()` (`internal/version/version.go:26-32`) embeds `version.json` but is never sent on the wire.

Key files:

- `app.go` (Wails bridge) + `internal/` packages — see issue body for exact paths.

## 3. Problems / symptoms

- Phone on old proto + PC on new proto fails deep in a transfer, not at hello.
- No way to gate breaking features (resume, richer MATCHES) on peer capability.
- Debugging field logs requires asking both sides for versions manually.

## 4. Proposal (what "good" looks like)

Add `/hello <client-version> [<proto>]` → `WAYERPC <server-version> <proto>`. Unknown today, answered tomorrow: old servers reply ERROR (client proceeds legacy-style); new clients tolerate ERROR. Backward compatible, no bump required to ship.

## 5. Step-by-step implementation

1. Add `hello` branch in `handleCommand` before `/ask`; reply via `sendLine`.
2. Include `version.Current().Version` + `.ProtocolVersion` in the reply.
3. Log hello at `[CONNECT]` level with client version (`server.go:204-212`).
4. Add test: hello → exact reply format; unknown command still ERROR.
5. Document in `README.md` Protocol + Android repo issue for client to send hello first.
6. Later: gate `008` resume and `009` richer MATCHES on negotiated proto.

Work in small commits. After each step, run the relevant check:

```powershell
go build ./...; if ($?) { go test ./... }
cmd /c "pnpm.cmd --dir frontend build"
```

## 6. Acceptance criteria

- [ ] `echo /hello | nc` returns `WAYERPC <ver> <proto>`.
- [ ] Old client (no hello) still works — handshake is optional.
- [ ] Connect log shows client version.

## 7. Tests to add / update

`TestHelloHandshake`, plus conformance-script hello case (`091`).

## 8. Risks and rollback

Minimal. Only new command; no existing path changes. Keep hello optional forever.

Rollback: `git stash` or revert the single commit for this issue. Nothing here
touches the wire protocol unless the header says *(breaking)* — and none of the
early-phase issues do.

## 9. Follow-ups (do NOT do in this issue)

`007` (ping), `008` (gated resume), `013` (error taxonomy reuses hello's version context).

## 10. Touch-and-learn notes

- Read the linked files first, top to bottom, before editing.
- Note one thing that surprised you in the commit message.
- If the step takes > M (1-3 days) estimate, stop and split it — open a follow-up note.
