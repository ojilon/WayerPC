# 049 Readable storage permission errors

| Field | Value |
|-------|-------|
| Status | `todo` (change to `doing` / `done` as you work) |
| Priority | P2 |
| Size | S (<1 day) |
| Area | storage/frontend |
| Phase | Week 5 — hardening |
| Related | `047` |

## 1. Why this matters

Storage failures speak errno: `storage: mkdir …: permission denied` in a log nobody reads, OFFLINE in the UI. Users need 'WayerPC can't write D:\… — run as admin? pick another folder' with one click to re-pick.

Read this file, then touch the code. This issue is deliberately scoped so you can
finish it in one sitting and see the result in the running app.

## 2. Current behavior

Errors propagate as `fmt.Errorf("storage: …: %w")` (`storage.go:44,150,158…`) into `openRoot` log (`app.go:101`) — accurate but raw. `ListFiles` swallows unreadable entries silently (`storage.go:92-94`, parity with Python). No actionable UI.

Key files:

- `app.go` (Wails bridge) + `internal/` packages — see issue body for exact paths.

## 3. Problems / symptoms

- Raw errno in UI/toast confuses; no next action.
- Silent skips hide real problems (failing disk reads as 'empty folder').
- No probe: app discovers unwritable root only after failing.

## 4. Proposal (what "good" looks like)

Map common storage errors to human guidance (EACCES→'no write permission…', ENOSPC→'disk full…', EROFS→'read-only…'); Settings + Console show guidance + 'Choose another folder' button; count-but-don't-hide skipped entries ('12 unreadable skipped'). Probe writability on root adopt (temp file create+delete).

## 5. Step-by-step implementation

1. Add `HumanizeStorageErr(err)` helper (errors.Is unwraps: Permission→guidance, NoSpace→guidance, else raw).
2. Use in `openRoot` failure path + `SetDataRoot` errors + relocate errors.
3. `ListFiles`: return `skippedUnreadable int` alongside (signature change — update callers + mock).
4. UI: skipped-count line in Files hint (`library.ts:104`).
5. Writability probe in `Init`/adopt flow.

Work in small commits. After each step, run the relevant check:

```powershell
go build ./...; if ($?) { go test ./... }
cmd /c "pnpm.cmd --dir frontend build"
```

## 6. Acceptance criteria

- [ ] Read-only-root adopt shows guidance + re-pick button (tested with ACL fixture).
- [ ] Unreadable entries counted, not hidden.
- [ ] No raw `mkdir …: permission denied` reaches a toast (mapped instead).

## 7. Tests to add / update

Humanize-matrix tests; skipped-count test with chmod-000 fixture (skip on Windows w/o ACL support, note why).

## 8. Risks and rollback

OS error-mapping differences — default to raw message when unmapped (never blank).

Rollback: `git stash` or revert the single commit for this issue. Nothing here
touches the wire protocol unless the header says *(breaking)* — and none of the
early-phase issues do.

## 9. Follow-ups (do NOT do in this issue)

Installer default-data-root perms (verify standard user can write it).

## 10. Touch-and-learn notes

- Read the linked files first, top to bottom, before editing.
- Note one thing that surprised you in the commit message.
- If the step takes > S (<1 day) estimate, stop and split it — open a follow-up note.
