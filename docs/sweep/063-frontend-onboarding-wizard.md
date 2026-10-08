# 063 First-run onboarding wizard

| Field | Value |
|-------|-------|
| Status | `todo` (change to `doing` / `done` as you work) |
| Priority | P1 |
| Size | M (1-3 days) |
| Area | frontend |
| Phase | Week 4 — frontend depth |
| Related | `048`, `015`, `061` |

## 1. Why this matters

First run today = Console banner + Settings hint ('No data folder yet', `console.ts:77-81`, `settings.ts:54-60`). A 3-step wizard (1 data folder with space check, 2 hotspot how-to with LAN IP, 3 test the link) turns abandonment into success.

Read this file, then touch the code. This issue is deliberately scoped so you can
finish it in one sitting and see the result in the running app.

## 2. Current behavior

Onboarding is two banners + `SetDataRoot` flow (`app.go:442-459`, `settings.ts:70-81`). No stepper, no progress, no 'test connection' action, no QR (`016`). `GetStorageInfo.onboarded` is the only state.

Key files:

- `app.go` (Wails bridge) + `internal/` packages — see issue body for exact paths.

## 3. Problems / symptoms

- New users land on an OFFLINE Console with no path forward.
- Data-root choice has no space/fs guidance (`048`).
- 'Did I set it up right?' unanswerable without the phone in hand.

## 4. Proposal (what "good" looks like)

Wizard overlay on `!onboarded`: Step 1 folder (picker + `048` space/fs line + Create), Step 2 connect (LAN IP `015` + QR `016` if landed + hotspot instructions), Step 3 verify (server running + 'Send /hello from phone' checklist + open Console). Skippable, resumable, never blocks re-entry to Settings.

## 5. Step-by-step implementation

1. Wizard component (3 steps, Back/Next/Skip) shown when `!onboarded` (keep banners as fallback).
2. Step 1 embeds folder picker + preflight line (reuse `047`/`048` helpers).
3. Step 2 shows IP (+QR if `016` done, else text) + 3-line hotspot recipe.
4. Step 3: live server state + 'connection seen' indicator (total>0 or hello log) + Finish → Console.
5. Persist `onboarding_seen` so it doesn't nag; Settings 'Replay onboarding' link.

Work in small commits. After each step, run the relevant check:

```powershell
go build ./...; if ($?) { go test ./... }
cmd /c "pnpm.cmd --dir frontend build"
```

## 6. Acceptance criteria

- [ ] Fresh profile completes setup without visiting docs.
- [ ] Skip path leaves app in today's banner state (no worse).
- [ ] Wizard never appears for onboarded users.

## 7. Tests to add / update

Onboarded/onboarding-seen matrix tests on state logic; manual fresh-profile run.

## 8. Risks and rollback

Wizard rot (screenshots/steps stale after protocol changes) — keep copy minimal, link to README for details.

Rollback: `git stash` or revert the single commit for this issue. Nothing here
touches the wire protocol unless the header says *(breaking)* — and none of the
early-phase issues do.

## 9. Follow-ups (do NOT do in this issue)

`016` (QR inside wizard), `061` (wizard copy via i18n).

## 10. Touch-and-learn notes

- Read the linked files first, top to bottom, before editing.
- Note one thing that surprised you in the commit message.
- If the step takes > M (1-3 days) estimate, stop and split it — open a follow-up note.
