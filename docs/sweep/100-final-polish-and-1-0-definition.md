# 100 Final polish + what 1.0 means

| Field | Value |
|-------|-------|
| Status | `todo` (change to `doing` / `done` as you work) |
| Priority | P0 |
| Size | S (<1 day) |
| Area | meta/release |
| Phase | Week 8 — the finish line |
| Related | `002`, `099` |

## 1. Why this matters

A two-month sweep needs a definition of done, or issue 100 feels like issue 1. This is the 1.0 gate: which files MUST be `done` (P0s + gates), what 'polish' means concretely (no OFFLINE-without-reason, no dead buttons, both themes readable, changelog current), and what explicitly slips to 1.1.

Read this file, then touch the code. This issue is deliberately scoped so you can
finish it in one sitting and see the result in the running app.

## 2. Current behavior

No 1.0 criteria; `version.json` at 0.2.x; `SUGGESTIONS` P0 list is the closest thing to a gate (`004` READY-newline, `005` checksum, `022` disk-cache, `070` EnVar, `091` conformance). `100` turns that into a sign-off sheet.

Key files:

- `app.go` (Wails bridge) + `internal/` packages — see issue body for exact paths.

## 3. Problems / symptoms

- Endless sweep (P2s expand to fill all time).
- 'Polish' without a list = vibes.
- 1.1 scope leaks into 1.0 pressure.

## 4. Proposal (what "good" looks like)

Gate: all P0 files done (`004`,`005`,`022`,`070`,`091`) + `098` QA signed on the release build + `099` notes current + `094` CI green + no `todo` P1 in Weeks 1-3 phases. Polish list: empty states (`068`), shortcuts sheet (`062`), onboarding (`063`) OR documented defer, light theme (`060`) OR documented defer. 1.1 parking lot: FTS5 (`027`), index (`043`), content search (`042`), one-click update (follow-up of `071`) unless already done. Sign, tag v1.0.0, celebrate.

## 5. Step-by-step implementation

1. Score the sweep: count done/doing/todo per priority (one command over headers).
2. Close or defer every P0; file 1.1 issues for deferred P1/P2 (link from here).
3. Polish pass: fresh-profile run (`063` script) + keyboard run (`062`) + both themes (`060`) + dead-button hunt (missing-row actions `032`/`051`/`058`).
4. QA sign (`098`) + changelog freeze (`099`) + hash publish (`073`).
5. Tag `v1.0.0` (version.json → sync → build → tag) and write the 1.1 headnotes.

Work in small commits. After each step, run the relevant check:

```powershell
go build ./...; if ($?) { go test ./... }
cmd /c "pnpm.cmd --dir frontend build"
```

## 6. Acceptance criteria

- [ ] P0s all `done` with commit refs.
- [ ] QA sign-off filed for the tagged build.
- [ ] 1.1 lot listed with owners (even if owner = 'future you').

## 7. Tests to add / update

Gates are human + CI (`094` green on the tag, `091` green against the tag exe).

## 8. Risks and rollback

Perfectionism — the defer list is part of done. Ship 1.0 with known P2s deferred and SAY so in the notes.

Rollback: `git stash` or revert the single commit for this issue. Nothing here
touches the wire protocol unless the header says *(breaking)* — and none of the
early-phase issues do.

## 9. Follow-ups (do NOT do in this issue)

1.1 kickoff: top of the lot is usually `027`/`008`/`010` — pick after a break.

## 10. Touch-and-learn notes

- Read the linked files first, top to bottom, before editing.
- Note one thing that surprised you in the commit message.
- If the step takes > S (<1 day) estimate, stop and split it — open a follow-up note.
