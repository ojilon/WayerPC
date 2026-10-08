# 003 Coding conventions for this sweep

| Field | Value |
|-------|-------|
| Status | `todo` (change to `doing` / `done` as you work) |
| Priority | P0 |
| Size | S (<2h) |
| Area | meta |
| Phase | Week 1 — orientation |
| Related | `001`, `002` |

## 1. Why this matters

AI wrote most of this codebase; you are now sweeping through by hand. Small consistent rules (commit prefix, verify commands, no drive-by refactors) keep 100 touches from conflicting.

Read this file, then touch the code. This issue is deliberately scoped so you can
finish it in one sitting and see the result in the running app.

## 2. Current behavior

`main.go:25-33` (flags), `app.go` (Wails bridge, ~514 lines), `internal/*` (pure-Go backend), `frontend/src/*.ts` (no framework, view functions). Tests exist per package (`*_test.go`) but coverage is thin. `BUILD.md` documents build; nothing documents commit style.

Key files:

- `app.go` (Wails bridge) + `internal/` packages — see issue body for exact paths.

## 3. Problems / symptoms

- Mixed concerns risk: a search fix that also reformats `catalog.go` is unreviewable.
- No commit convention: history is hard to bisect later.
- Frontend has no typecheck step documented; TS errors surface only at `pnpm build`.

## 4. Proposal (what "good" looks like)

Rules: one issue = one commit (`sweep(NNN): title`); run `go build ./...`, `go test ./...`, `pnpm --dir frontend build` before every commit; no refactors outside the issue's Steps; update the issue header Status in the same commit.

## 5. Step-by-step implementation

1. Adopt commit template `sweep(NNN): <imperative title>`; add co-authors only if AI assists.
2. Before each commit run: `go vet ./...`, `go test ./internal/...`, `pnpm --dir frontend exec tsc --noEmit` (add if missing).
3. Keep diffs < ~250 lines; split larger work and note the split in Follow-ups.
4. Leave `// sweep(NNN):` comments only where behavior is subtle (timeouts, framing).
5. Never commit `.data/`, `build/`, `out/` artifacts — check `git status` first.
6. Record each finished issue in `099` changelog line.

Work in small commits. After each step, run the relevant check:

```powershell
go build ./...; if ($?) { go test ./... }
cmd /c "pnpm.cmd --dir frontend build"
```

## 6. Acceptance criteria

- [ ] `git log --oneline -5` shows `sweep(NNN):` prefixes after Week 1.
- [ ] No commit mixes two issue numbers.
- [ ] Verify commands for Go + frontend are known by heart.

## 7. Tests to add / update

Process-only. Optional: add `git log --oneline -10` check to your pre-commit habit.

## 8. Risks and rollback

Overhead fatigue. If the ritual slips, shrink it to build+test, never skip verification entirely.

Rollback: `git stash` or revert the single commit for this issue. Nothing here
touches the wire protocol unless the header says *(breaking)* — and none of the
early-phase issues do.

## 9. Follow-ups (do NOT do in this issue)

`093` (mock parity), `094` (CI smoke that automates this ritual).

## 10. Touch-and-learn notes

- Read the linked files first, top to bottom, before editing.
- Note one thing that surprised you in the commit message.
- If the step takes > S (<2h) estimate, stop and split it — open a follow-up note.
