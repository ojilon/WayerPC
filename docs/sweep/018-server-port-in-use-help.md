# 018 Port-in-use help

| Field | Value |
|-------|-------|
| Status | `todo` (change to `doing` / `done` as you work) |
| Priority | P1 |
| Size | S (<1 day) |
| Area | backend/frontend |
| Phase | Week 3 — backend reliability |
| Related | `052` |

## 1. Why this matters

Bind failure today just logs (`server.go:137-139`, `app.go:142-144`). Users see OFFLINE with no next action; the fix (change port in config file by hand) is undiscoverable.

Read this file, then touch the code. This issue is deliberately scoped so you can
finish it in one sitting and see the result in the running app.

## 2. Current behavior

`Start()` returns the raw `net.Listen` error; `StartServer` (`app.go:253-269`) returns 'starting' or nothing useful. No EADDRINUSE detection, no owning-process hint, no next-port offer. Host/port are config-file-only (`052`).

Key files:

- `app.go` (Wails bridge) + `internal/` packages — see issue body for exact paths.

## 3. Problems / symptoms

- Second instance / stale process → cryptic log, silent OFFLINE.
- Non-technical users cannot recover alone.
- Settings has no port field to act on the advice.

## 4. Proposal (what "good" looks like)

Detect EADDRINUSE (syscall errno unwrap), log a helpful line (port N busy — close the other WayerPC or pick N+1 in Settings), surface a banner in Console + Settings with a one-click 'Use next port' button (persists via `052`).

## 5. Step-by-step implementation

1. Add `isAddrInUse(err)` helper (errors.As + syscall.EADDRINUSE, Windows WSAEADDRINUSE).
2. Return typed error from `Start()`; `app.go` maps to banner state in `GetStatus`.
3. Console banner (`console.ts:18`) + Settings action apply `port+1` → save → restart server.
4. Test: occupy :5000 with dummy listener, assert helpful status.
5. Document in BUILD.md troubleshooting.

Work in small commits. After each step, run the relevant check:

```powershell
go build ./...; if ($?) { go test ./... }
cmd /c "pnpm.cmd --dir frontend build"
```

## 6. Acceptance criteria

- [ ] Port conflict shows 'Port 5000 busy…' banner, not bare OFFLINE.
- [ ] One-click next-port recovers without editing files.
- [ ] Normal bind path unchanged.

## 7. Tests to add / update

`TestPortInUseHelp` with occupied port.

## 8. Risks and rollback

Windows errno mapping quirks — test on Windows CI (`094`).

Rollback: `git stash` or revert the single commit for this issue. Nothing here
touches the wire protocol unless the header says *(breaking)* — and none of the
early-phase issues do.

## 9. Follow-ups (do NOT do in this issue)

`052` (host/port UI this button needs).

## 10. Touch-and-learn notes

- Read the linked files first, top to bottom, before editing.
- Note one thing that surprised you in the commit message.
- If the step takes > S (<1 day) estimate, stop and split it — open a follow-up note.
