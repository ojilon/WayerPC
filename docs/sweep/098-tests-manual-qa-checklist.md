# 098 Manual QA checklist (phone + hotspot, 30 min)

| Field | Value |
|-------|-------|
| Status | `todo` (change to `doing` / `done` as you work) |
| Priority | P1 |
| Size | S (<1 day) |
| Area | tests |
| Phase | Week 6 — tests |
| Related | `091`, `099` |

## 1. Why this matters

Automated gates (`091`/`094`) can't hold a phone on a flaky hotspot. A written 30-minute script (fresh install → onboard → upload 3 sizes → ask found/matches/missing → kill-mid-upload → port conflict → relocate → uninstall-keep) makes releases repeatable and delegable.

Read this file, then touch the code. This issue is deliberately scoped so you can
finish it in one sitting and see the result in the running app.

## 2. Current behavior

No checklist; QA is memory + chat history. `SUGGESTIONS` §5 has pieces (conformance, smoke) but no human script. Release testing varies per release.

Key files:

- `app.go` (Wails bridge) + `internal/` packages — see issue body for exact paths.

## 3. Problems / symptoms

- 'Did anyone test resume on a real hotspot?' — unknown.
- Regressed onboarding (`063`) ships because nobody fresh-installs.
- Two-phone + scan-neighbor cases never exercised.

## 4. Proposal (what "good" looks like)

Checklist file (this issue writes `tests/QA-CHECKLIST.md`, not here): setup (2 phones? 1 + laptop-hotspot), 25 steps with expected results + log lines to grep (`[ASK OK]`, `[UPLOAD OK]`, `[TIMEOUT]`), device matrix (Android versions), sign-off line (tester, date, build). Run it once yourself against current build (baseline), fix the checklist, then require it per release (`099`).

## 5. Step-by-step implementation

1. Write the checklist (25 steps, 30-min budget, copy-paste commands incl. `ncat`/conformance shortcuts).
2. Run it on the current build; record baseline pass/fail.
3. Fix checklist ambiguities found during the run.
4. Require sign-off in `099` release process.
5. Store signed checklists per release (e.g. `tests/qa/vX.Y.Z.md` copy).

Work in small commits. After each step, run the relevant check:

```powershell
go build ./...; if ($?) { go test ./... }
cmd /c "pnpm.cmd --dir frontend build"
```

## 6. Acceptance criteria

- [ ] Checklist exists, timed ≤30 min on a real run.
- [ ] Baseline run recorded (pass/fail per step).
- [ ] `099` references it as a release gate.

## 7. Tests to add / update

Human-run. Track as release artifact, not CI.

## 8. Risks and rollback

Checklist rot (UI steps stale after `063`/`066`) — owner per release refreshes steps before running.

Rollback: `git stash` or revert the single commit for this issue. Nothing here
touches the wire protocol unless the header says *(breaking)* — and none of the
early-phase issues do.

## 9. Follow-ups (do NOT do in this issue)

`099` (gate), Android-repo QA mirror note.

## 10. Touch-and-learn notes

- Read the linked files first, top to bottom, before editing.
- Note one thing that surprised you in the commit message.
- If the step takes > S (<1 day) estimate, stop and split it — open a follow-up note.
