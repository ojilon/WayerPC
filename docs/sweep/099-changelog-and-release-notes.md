# 099 CHANGELOG habit + release notes process

| Field | Value |
|-------|-------|
| Status | `todo` (change to `doing` / `done` as you work) |
| Priority | P2 |
| Size | S (<1 day) |
| Area | release |
| Phase | Week 1 — start now, keep always |
| Related | `100`, `071` |

## 1. Why this matters

`SUGGESTIONS-v0.2.1.md` §5 asks for Keep-a-Changelog; nothing exists yet (verify: no `CHANGELOG.md` at root). 100 sweep commits without a changelog = release notes written from memory, missing the fixes users need (`004`-`011` protocol notes are user-critical).

Read this file, then touch the code. This issue is deliberately scoped so you can
finish it in one sitting and see the result in the running app.

## 2. Current behavior

Version in `version.json` (Go+frontend+installer derive via `sync-version.mjs`); `SUGGESTIONS` is a menu, not a log. Release process lives in BUILD.md fragments + memory.

Key files:

- `app.go` (Wails bridge) + `internal/` packages — see issue body for exact paths.

## 3. Problems / symptoms

- Users can't tell if their bug is fixed without reading commits.
- Protocol changes lack a user-facing record (Android coordination needs it).
- Release cut is improvised each time.

## 4. Proposal (what "good" looks like)

Start `CHANGELOG.md` (Keep-a-Changelog: Added/Fixed/Changed + Protocol subsection per release); every sweep commit adds one line under Unreleased (enforced by review rule `003`); release process = version bump (`version.json`) → QA sign (`098`) → CI green (`094`) → hash publish (`073`) → notes from changelog. Tag v0.2.0 now (rewrite baseline) if not yet tagged.

## 5. Step-by-step implementation

1. Create `CHANGELOG.md` with Unreleased + v0.2.0 (rewrite) sections; backfill from `SUGGESTIONS` fixed-bugs + git log.
2. Add 'Protocol' subsection convention (client-visible changes only).
3. `003` amendment: sweep commits touch the changelog (one line).
4. Release checklist section (version bump order: `version.json` → sync script → build → tag).
5. `071` feed reads these notes (link contract).

Work in small commits. After each step, run the relevant check:

```powershell
go build ./...; if ($?) { go test ./... }
cmd /c "pnpm.cmd --dir frontend build"
```

## 6. Acceptance criteria

- [ ] CHANGELOG.md exists with Unreleased + v0.2.0.
- [ ] 5 sweep commits each add a line (habit proven).
- [ ] Release checklist merged (points at `098` + `094` + `073`).

## 7. Tests to add / update

None (doc). Verify by doing one version-bump dry run (no tag).

## 8. Risks and rollback

Changelog conflicts on parallel branches — one-line entries, resolve trivially; never gate a fix on changelog bikeshed.

Rollback: `git stash` or revert the single commit for this issue. Nothing here
touches the wire protocol unless the header says *(breaking)* — and none of the
early-phase issues do.

## 9. Follow-ups (do NOT do in this issue)

`100` (1.0 notes come from here), `071` (feed links here).

## 10. Touch-and-learn notes

- Read the linked files first, top to bottom, before editing.
- Note one thing that surprised you in the commit message.
- If the step takes > S (<1 day) estimate, stop and split it — open a follow-up note.
