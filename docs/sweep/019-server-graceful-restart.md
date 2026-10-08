# 019 Graceful server restart on config change

| Field | Value |
|-------|-------|
| Status | `todo` (change to `doing` / `done` as you work) |
| Priority | P1 |
| Size | S (<1 day) |
| Area | backend |
| Phase | Week 3 — backend reliability |
| Related | `052`, `018` |

## 1. Why this matters

Changing host/port/data-root today needs mental gymnastics: `Attach` hot-swaps backends (`server.go:119-124`) but listener bind never changes without stop+start racing active transfers. Settings needs one safe Restart button.

Read this file, then touch the code. This issue is deliberately scoped so you can
finish it in one sitting and see the result in the running app.

## 2. Current behavior

`Stop()` closes listener; conns drain (`server.go:163-174`). `StartServer`/`StopServer` (`app.go:253-281`) are separate calls with no sequencing; `openRoot` starts once. No 'restart' primitive; rapid stop→start can hit TIME_WAIT/EADDRINUSE.

Key files:

- `app.go` (Wails bridge) + `internal/` packages — see issue body for exact paths.

## 3. Problems / symptoms

- Port change strands active transfers or fails to rebind.
- Double-Start races (button spam) spawn confusing states.
- No single backend method the UI can call for 'apply + restart'.

## 4. Proposal (what "good" looks like)

Add `Server.Restart(newHost, newPort)` (stop, rebind with retry, keep stats) + `App.RestartServer()` bridge. Serialize with mutex; refuse concurrent restarts; drain with deadline; surface final state via `GetStatus`.

## 5. Step-by-step implementation

1. Add `Restart` to `Server` with stop→bind-retry (3x, 200ms backoff)→serve loop.
2. Add `App.RestartServer(host, port)` calling it + emitting `stats`.
3. Disable Start/Stop buttons mid-restart in `console.ts`.
4. Test: restart with active upload → old conn drains, new listener answers hello.
5. Wire to `052` port editor + `015` IP refresh.

Work in small commits. After each step, run the relevant check:

```powershell
go build ./...; if ($?) { go test ./... }
cmd /c "pnpm.cmd --dir frontend build"
```

## 6. Acceptance criteria

- [ ] Restart during idle rebinds <1s with no error.
- [ ] Restart during upload: upload either completes or fails cleanly (no hang).
- [ ] Button spam doesn't wedge the server.

## 7. Tests to add / update

Restart integration test (start → occupy conn → restart → hello on new port).

## 8. Risks and rollback

Drain semantics: document 'running connections drain, new conns use new bind'.

Rollback: `git stash` or revert the single commit for this issue. Nothing here
touches the wire protocol unless the header says *(breaking)* — and none of the
early-phase issues do.

## 9. Follow-ups (do NOT do in this issue)

`052` (UI caller), `078` (restart counter metric).

## 10. Touch-and-learn notes

- Read the linked files first, top to bottom, before editing.
- Note one thing that surprised you in the commit message.
- If the step takes > S (<1 day) estimate, stop and split it — open a follow-up note.
