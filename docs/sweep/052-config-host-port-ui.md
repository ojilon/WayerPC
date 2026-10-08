# 052 Host/port editing in Settings

| Field | Value |
|-------|-------|
| Status | `todo` (change to `doing` / `done` as you work) |
| Priority | P1 |
| Size | S (<1 day) |
| Area | config/frontend |
| Phase | Week 3 — backend reliability |
| Related | `018`, `019` |

## 1. Why this matters

Host/port are config-file-only (`config.go:Load/Save`, `settings.ts:28-31` shows addr read-only). Every port conflict (`018`), restart (`019`), or hotspot change needs Notepad + restart. This unlocks all three.

Read this file, then touch the code. This issue is deliberately scoped so you can
finish it in one sitting and see the result in the running app.

## 2. Current behavior

`Config{Host,Port}` (`config.go:33-39`) with validation on load (`config.go:103-107`); `Save` normalizes empties. UI never writes them. `App` holds `cfg` (`app.go:66`) but no setter bridge exists.

Key files:

- `app.go` (Wails bridge) + `internal/` packages — see issue body for exact paths.

## 3. Problems / symptoms

- Undiscoverable recovery from EADDRINUSE.
- Power users can't bind localhost-only for testing.
- Invalid manual edits (port 99999) silently revert to defaults — confusing.

## 4. Proposal (what "good" looks like)

Settings Server card: host + port fields, validate (1-65535, host IP/hostname or 0.0.0.0), Save → `RestartServer` (`019`) with confirm. Show effective LAN IP (`015`) next to the fields. Invalid input blocked inline with reasons.

## 5. Step-by-step implementation

1. Add `App.SetServerAddr(host, port)` (validate → save → restart via `019`).
2. Settings UI fields + validation messages + Apply button.
3. Confirm when server running ('Restart to apply? Active transfers drain.').
4. Persist via `config.Save`; reload path re-validates.
5. Test: bad port rejected; good port rebinds and survives restart.

Work in small commits. After each step, run the relevant check:

```powershell
go build ./...; if ($?) { go test ./... }
cmd /c "pnpm.cmd --dir frontend build"
```

## 6. Acceptance criteria

- [ ] Port change applies without touching files.
- [ ] Invalid input never written; message explains why.
- [ ] Restart honors drain semantics (`019`).

## 7. Tests to add / update

Setter validation tests; restart-on-apply integration test.

## 8. Risks and rollback

Fat-finger lockout (bind 1.2.3.4 that doesn't exist) — validate bind succeeds, auto-revert to last-good on failure.

Rollback: `git stash` or revert the single commit for this issue. Nothing here
touches the wire protocol unless the header says *(breaking)* — and none of the
early-phase issues do.

## 9. Follow-ups (do NOT do in this issue)

`019` (restart primitive), `055` (profiles reuse this).

## 10. Touch-and-learn notes

- Read the linked files first, top to bottom, before editing.
- Note one thing that surprised you in the commit message.
- If the step takes > S (<1 day) estimate, stop and split it — open a follow-up note.
