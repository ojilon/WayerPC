# 096 Backend unit coverage: fill the gaps per package

| Field | Value |
|-------|-------|
| Status | `todo` (change to `doing` / `done` as you work) |
| Priority | P1 |
| Size | M (1-3 days) |
| Area | tests/backend |
| Phase | Week 6 — tests |
| Related | `091`, `092` |

## 1. Why this matters

Per-package tests exist (`config_test`, `catalog_test`, `search_test`, `server_test`, `storage_test`, `version_test`) but thin on the paths users hit: relocate crash (`047`), prune (`032`), sanitize (`014`), timeout tags (`021`), busy-retry (`023`). Coverage as a habit, not a number chase.

Read this file, then touch the code. This issue is deliberately scoped so you can
finish it in one sitting and see the result in the running app.

## 2. Current behavior

Unknown coverage (run `go test -cover ./...` to find out — first step). Likely gaps: error paths (unwritable, full disk), concurrent paths (-race), migration paths (legacy DB, `033` versions), platform branches (Windows-only reveal/open).

Key files:

- `app.go` (Wails bridge) + `internal/` packages — see issue body for exact paths.

## 3. Problems / symptoms

- Happy-path tests; field failures live in error paths.
- No coverage gate → new code lands untested (`082` extraction needs a guard).
- Platform-specific code tested only on one OS.

## 4. Proposal (what "good" looks like)

Measure, then fill top-5 gaps by risk (not by %): relocate-verify, prune-missing, sanitize-matrix, busy-retry, timeout-tags. Table-driven style (match existing tests' voice). Modest gate: new exported funcs need tests (review rule `003`, not a CI % gate yet).

## 5. Step-by-step implementation

1. Run coverage per package; paste table into the commit message.
2. Fill the 5 gaps (one commit each if large).
3. Add `-race` to the local test command + document (`003` update).
4. Skip-gracefully where OS-bound (with reason strings, not silent skips).
5. Re-measure; record delta (no target % — trend matters).

Work in small commits. After each step, run the relevant check:

```powershell
go build ./...; if ($?) { go test ./... }
cmd /c "pnpm.cmd --dir frontend build"
```

## 6. Acceptance criteria

- [ ] Coverage table before/after in the commit(s).
- [ ] The 5 named gaps have tests.
- [ ] `go test -race ./...` green.

## 7. Tests to add / update

The new tests. Keep each <50 lines, table-driven.

## 8. Risks and rollback

Coverage theater (assert-nothing tests) — each test must fail if the code is reverted (mutation spot-check 2-3).

Rollback: `git stash` or revert the single commit for this issue. Nothing here
touches the wire protocol unless the header says *(breaking)* — and none of the
early-phase issues do.

## 9. Follow-ups (do NOT do in this issue)

`098` (manual QA covers what units can't), `089` (bench discipline shared).

## 10. Touch-and-learn notes

- Read the linked files first, top to bottom, before editing.
- Note one thing that surprised you in the commit message.
- If the step takes > M (1-3 days) estimate, stop and split it — open a follow-up note.
