# 071 In-app update check (GitHub Releases feed)

| Field | Value |
|-------|-------|
| Status | `todo` (change to `doing` / `done` as you work) |
| Priority | P1 |
| Size | M (1-3 days) |
| Area | installer/frontend |
| Phase | Week 6 — release |
| Related | `099` |

## 1. Why this matters

Manual re-installs rot the install base: protocol fixes (`004`-`009`) only help if users run them. A start-up version check (compare + download link, later one-click) closes the loop.

Read this file, then touch the code. This issue is deliberately scoped so you can
finish it in one sitting and see the result in the running app.

## 2. Current behavior

Version from `version.json` → Go + frontend + installer (`sync-version.mjs`, `internal/version/version.go`). No feed check, no 'new version' UI, no download path. Settings About (`settings.ts:33-37`) shows version statically.

Key files:

- `app.go` (Wails bridge) + `internal/` packages — see issue body for exact paths.

## 3. Problems / symptoms

- Users run stale builds with known hotspot bugs.
- No channel (stable vs pre) concept.
- One-click update is a big build — needs a link-only first step.

## 4. Proposal (what "good" looks like)

Phase 1 (this issue): on start (debounced, weekly, opt-out), fetch GitHub Releases latest tag, semver-compare, show 'Update available: vX.Y.Z — download' banner in Settings + Console (link only). No auto-download yet. Respect offline (silent) + metered-hotspot caution (check only, tiny payload).

## 5. Step-by-step implementation

1. `App.CheckUpdates()` (net client, 10s timeout, cache last-check in settings kv).
2. Settings banner + Console notice with release-notes link.
3. Opt-out toggle; manual 'Check now' button.
4. Tests with local HTTP stub (no live GitHub in CI).
5. Document channels (tag scheme `v*` stable) + `099` feeds the notes link.

Work in small commits. After each step, run the relevant check:

```powershell
go build ./...; if ($?) { go test ./... }
cmd /c "pnpm.cmd --dir frontend build"
```

## 6. Acceptance criteria

- [ ] Stale build shows banner within one start (stubbed feed).
- [ ] Offline start shows nothing (no error toast).
- [ ] Opt-out persists; manual check works.

## 7. Tests to add / update

Feed-parse + compare tests (semver edge: rc, patch, older).

## 8. Risks and rollback

False positives from draft/prerelease tags — filter to non-draft, non-prerelease.

Rollback: `git stash` or revert the single commit for this issue. Nothing here
touches the wire protocol unless the header says *(breaking)* — and none of the
early-phase issues do.

## 9. Follow-ups (do NOT do in this issue)

One-click download+install (separate, bigger issue); MSIX/Store story if ever.

## 10. Touch-and-learn notes

- Read the linked files first, top to bottom, before editing.
- Note one thing that surprised you in the commit message.
- If the step takes > M (1-3 days) estimate, stop and split it — open a follow-up note.
