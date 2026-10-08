# 001 How to use this sweep (read me first)

| Field | Value |
|-------|-------|
| Status | `todo` (change to `doing` / `done` as you work) |
| Priority | P0 |
| Size | S (<2h) |
| Area | meta |
| Phase | Week 1 — orientation |
| Related | `002`, `003`, `100` |

## 1. Why this matters

You asked for ~100 single-case files you read one at a time over a month or two. This file defines the loop: read one md, make the change, verify, commit, move on. Without this discipline the sweep sprawls.

Read this file, then touch the code. This issue is deliberately scoped so you can
finish it in one sitting and see the result in the running app.

## 2. Current behavior

Today `docs/` has `ARCHITECTURE.md` (stale: describes the old Python/C++ layout, not `internal/`), `NATIVE.md`, `PACKAGING.md`, `SUGGESTIONS-v0.2.1.md` (the prioritized menu this sweep is built from). There is no per-issue workflow doc.

Key files:

- `app.go` (Wails bridge) + `internal/` packages — see issue body for exact paths.

## 3. Problems / symptoms

- No defined order: easy to start with a hard Zig port before learning the Go code.
- No done-definition per issue: 'improved search' never ends.
- Stale `ARCHITECTURE.md` misleads every new reading.

## 4. Proposal (what "good" looks like)

Adopt the loop used by every file in this folder: Status header, scoped steps, acceptance checkboxes, verify commands. Work in numeric order within each phase, but skip freely — the Phase field tells you what is safe early (Weeks 1-2: UI + small backend) vs late (Weeks 6-8: Zig, FTS5, benchmarks).

## 5. Step-by-step implementation

1. Read `002-roadmap-overview-and-order.md` for the 8-week arc.
2. Read `003-coding-conventions-for-this-sweep.md` for commit/build rules.
3. Fix `docs/ARCHITECTURE.md` staleness as your first commit (one paragraph pointing at `internal/` + `app.go`).
4. Pick the lowest-numbered `todo` file in the current week's phase and flip it to `doing`.
5. Implement only its Steps section; run its verify commands.
6. Commit with message `sweep(<num>): <title>` and flip header to `done`.

Work in small commits. After each step, run the relevant check:

```powershell
go build ./...; if ($?) { go test ./... }
cmd /c "pnpm.cmd --dir frontend build"
```

## 6. Acceptance criteria

- [ ] You can describe the 8-week arc in 3 sentences.
- [ ] `docs/ARCHITECTURE.md` no longer describes `backend/*.py` as current.
- [ ] First real issue commit follows the `sweep(NNN):` prefix.

## 7. Tests to add / update

No code tests. Verify: `Get-ChildItem docs/sweep/*.md | Measure-Object` returns 100.

## 8. Risks and rollback

Near zero. Doc-only change. Risk is process drift — re-read this file whenever an issue balloons past its size estimate.

Rollback: `git stash` or revert the single commit for this issue. Nothing here
touches the wire protocol unless the header says *(breaking)* — and none of the
early-phase issues do.

## 9. Follow-ups (do NOT do in this issue)

`002` (full ordering), `099` (changelog habit that records each sweep commit).

## 10. Touch-and-learn notes

- Read the linked files first, top to bottom, before editing.
- Note one thing that surprised you in the commit message.
- If the step takes > S (<2h) estimate, stop and split it — open a follow-up note.
