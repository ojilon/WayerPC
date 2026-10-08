# 074 Start-with-Windows toggle

| Field | Value |
|-------|-------|
| Status | `todo` (change to `doing` / `done` as you work) |
| Priority | P2 |
| Size | S (<1 day) |
| Area | installer/frontend |
| Phase | Week 6 — release |
| Related | `055` |

## 1. Why this matters

Server-first users (family photo hub on a home PC) want WayerPC listening after reboot without logging in gymnastics. Registry Run key + Settings checkbox is the standard shape.

Read this file, then touch the code. This issue is deliberately scoped so you can
finish it in one sitting and see the result in the running app.

## 2. Current behavior

No autostart: no Run key writes, no Settings toggle, installer offers nothing. `main.go` has no `--minimized`/`--tray` flag (window always shows).

Key files:

- `app.go` (Wails bridge) + `internal/` packages — see issue body for exact paths.

## 3. Problems / symptoms

- Manual Startup-folder hacks per user.
- No minimized/tray start — autostart that pops a window every boot annoys.
- Per-profile autostart (`055`) undefined.

## 4. Proposal (what "good" looks like)

Settings checkbox ↔ `HKCU\Software\Microsoft\Windows\CurrentVersion\Run\WayerPC` (quoted exe path + `--minimized`? — add the flag too, window starts hidden/minimized). Installer offers the checkbox (default off). Uninstall removes the key. Document Task Manager → Startup side-effect.

## 5. Step-by-step implementation

1. Registry helpers (set/remove/query) + `App.GetAutostart/SetAutostart` bridge.
2. `--minimized` flag in `main.go` (start hidden or minimized; verify Wails option support).
3. Settings toggle + installer checkbox (NSIS section, default off).
4. Uninstall cleanup (`072` page notes it).
5. Test: toggle on → key present → reboot-sim (logon via VM) → running; toggle off → gone.

Work in small commits. After each step, run the relevant check:

```powershell
go build ./...; if ($?) { go test ./... }
cmd /c "pnpm.cmd --dir frontend build"
```

## 6. Acceptance criteria

- [ ] Toggle round-trips the registry key.
- [ ] Autostarted instance doesn't steal focus (minimized/hidden).
- [ ] Uninstall removes the key in both keep/remove-data modes.

## 7. Tests to add / update

Registry round-trip tests (Windows-only, skip elsewhere); manual VM reboot check.

## 8. Risks and rollback

AV/enterprise policy may block Run keys — surface write errors as guidance, not crash.

Rollback: `git stash` or revert the single commit for this issue. Nothing here
touches the wire protocol unless the header says *(breaking)* — and none of the
early-phase issues do.

## 9. Follow-ups (do NOT do in this issue)

Tray-icon mode (future — needs backend tray support); per-profile autostart rule.

## 10. Touch-and-learn notes

- Read the linked files first, top to bottom, before editing.
- Note one thing that surprised you in the commit message.
- If the step takes > S (<1 day) estimate, stop and split it — open a follow-up note.
