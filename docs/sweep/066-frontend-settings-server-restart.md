# 066 Settings: server controls + restart button

| Field | Value |
|-------|-------|
| Status | `todo` (change to `doing` / `done` as you work) |
| Priority | P1 |
| Size | S (<1 day) |
| Area | frontend |
| Phase | Week 3 — frontend |
| Related | `019`, `052` |

## 1. Why this matters

Console owns Start/Stop (`console.ts:17-24`); Settings (where addr/port live, `settings.ts:27-32`) owns neither. Users edit server config where they can't restart, and restart where they can't edit. Co-locate.

Read this file, then touch the code. This issue is deliberately scoped so you can
finish it in one sitting and see the result in the running app.

## 2. Current behavior

Settings server card is read-only text (`settings.ts:62-67` paints addr+running). No buttons. `StartServer/StopServer` bridged (`app.go:253-281`); `RestartServer` arrives via `019`.

Key files:

- `app.go` (Wails bridge) + `internal/` packages — see issue body for exact paths.

## 3. Problems / symptoms

- Apply-port flow (`052`) dead-ends without restart control nearby.
- Running-state shown in two places, actionable in one.
- No 'restart to apply' affordance after any future server setting.

## 4. Proposal (what "good" looks like)

Settings server card gains Start/Stop/Restart buttons (same handlers as Console) + status dot reuse. Restart disabled mid-restart with spinner text. Both views reflect the same `GetStatus` state (no divergent polling).

## 5. Step-by-step implementation

1. Extract shared server-control snippet (or duplicate 15 lines deliberately — note the duplication + follow-up to componentize if a framework ever lands).
2. Wire to existing `api.startServer/stopServer` + new `restartServer` (`019`).
3. Disable-all during pending op; toast result.
4. Poll or subscribe consistent with Console (reuse `stats` event).
5. Manual: port change (`052`) → Restart here → Console reflects.

Work in small commits. After each step, run the relevant check:

```powershell
go build ./...; if ($?) { go test ./... }
cmd /c "pnpm.cmd --dir frontend build"
```

## 6. Acceptance criteria

- [ ] Port edit + restart completes without visiting Console.
- [ ] Buttons disable during ops; double-click safe.
- [ ] Status matches Console at all times.

## 7. Tests to add / update

Manual two-view consistency check; button-disable logic test if extracted.

## 8. Risks and rollback

Duplicated control code drifts — leave a `// keep in sync with console.ts` comment + follow-up.

Rollback: `git stash` or revert the single commit for this issue. Nothing here
touches the wire protocol unless the header says *(breaking)* — and none of the
early-phase issues do.

## 9. Follow-ups (do NOT do in this issue)

Shared-controls extraction (when frontend grows a component model).

## 10. Touch-and-learn notes

- Read the linked files first, top to bottom, before editing.
- Note one thing that surprised you in the commit message.
- If the step takes > S (<1 day) estimate, stop and split it — open a follow-up note.
