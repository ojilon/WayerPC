# 021 Idle vs stall timeout tuning + docs

| Field | Value |
|-------|-------|
| Status | `todo` (change to `doing` / `done` as you work) |
| Priority | P2 |
| Size | S (<1 day) |
| Area | backend |
| Phase | Week 3 — backend reliability |
| Related | `007`, `017` |

## 1. Why this matters

Three timeouts interact: 300s idle, 500ms settle, 60s write (`server.go:38-53`). Users can't tell 'slow but alive' from 'dead'. This issue documents + tunes them and makes stalls loud.

Read this file, then touch the code. This issue is deliberately scoped so you can
finish it in one sitting and see the result in the running app.

## 2. Current behavior

Comments explain each timeout well, but values are consts, behavior is log-only (`[TIMEOUT]`, `[DISCONNECT]`), and the frontend shows nothing. Slow-but-progressing transfers refresh deadlines per chunk (good) — undocumented in UI.

Key files:

- `app.go` (Wails bridge) + `internal/` packages — see issue body for exact paths.

## 3. Problems / symptoms

- 500ms settle adds latency to every newline-less command (fine) but is invisible.
- 60s write timeout can kill a stalled-but-recoverable hotspot transfer with a bare log.
- No UI distinction idle/stalled/active.

## 4. Proposal (what "good" looks like)

Keep values; add loudness: distinct log tags `[IDLE-TIMEOUT]` vs `[STALL]`; expose `lastActivity` in `Stats`; Console shows 'idle 4m' vs 'receiving…'. Make settle/write/idle config-overridable for tests + power users. Document the matrix in code + README.

## 5. Step-by-step implementation

1. Split timeout log lines with distinct tags + durations.
2. Track `lastActivityUnix` per server (updated per chunk + per command).
3. Surface in `GetStatus` → Console subtitle (`017` rate area).
4. Add config overrides (env or config keys, defaulting to current consts).
5. Write the timeout matrix doc (table: timeout, value, what it guards, user-visible effect).
6. Test: idle conn closed at deadline with correct tag (short override).

Work in small commits. After each step, run the relevant check:

```powershell
go build ./...; if ($?) { go test ./... }
cmd /c "pnpm.cmd --dir frontend build"
```

## 6. Acceptance criteria

- [ ] Log distinguishes idle-close from mid-transfer stall.
- [ ] Console shows time-since-activity.
- [ ] Matrix doc merged; values unchanged by default.

## 7. Tests to add / update

Timeout-tag assertions with short overrides.

## 8. Risks and rollback

Changing defaults would alter field behavior — don't. Loudness only.

Rollback: `git stash` or revert the single commit for this issue. Nothing here
touches the wire protocol unless the header says *(breaking)* — and none of the
early-phase issues do.

## 9. Follow-ups (do NOT do in this issue)

`017` (UI uses lastActivity), `076` (structured logs carry timeout fields).

## 10. Touch-and-learn notes

- Read the linked files first, top to bottom, before editing.
- Note one thing that surprised you in the commit message.
- If the step takes > S (<1 day) estimate, stop and split it — open a follow-up note.
