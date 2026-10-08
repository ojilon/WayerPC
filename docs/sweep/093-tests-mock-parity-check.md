# 093 Frontend mock parity check

| Field | Value |
|-------|-------|
| Status | `todo` (change to `doing` / `done` as you work) |
| Priority | P1 |
| Size | S (<1 day) |
| Area | tests/frontend |
| Phase | Week 2 — tests |
| Related | `037`, `065` |

## 1. Why this matters

`frontend/src/dev-mock.ts` drifts from `app.go` silently: every new bridge method (`GetConnectQR` `016`, `CountFiles` `064`, `GetMetrics` `078`) needs a mock, and every new event (`search-progress` `037`, `import-progress` `065`) needs a mock emitter. Drift = browser-preview lies.

Read this file, then touch the code. This issue is deliberately scoped so you can
finish it in one sitting and see the result in the running app.

## 2. Current behavior

`api.ts` `call()` falls back to `mockCall` (non-Wails); `onEvent` falls back to `mockOn`. No check that mock covers all `api.*` methods + events. New methods routinely land mock-less (verify by diffing).

Key files:

- `app.go` (Wails bridge) + `internal/` packages — see issue body for exact paths.

## 3. Problems / symptoms

- Preview-tested UI breaks in Wails (or vice versa).
- Event-driven features (`037`/`065`) untestable in preview.
- Drift discovered by humans, late.

## 4. Proposal (what "good" looks like)

CI check (node script, no deps): parse `api.ts` exported `api = {...}` method names + `onEvent(` call sites' event names across `views/`; assert each exists in `dev-mock.ts` (names + arg counts); fail with the missing list. Or generate the mock skeleton from `api.ts` (preferred if simple). Run in `094` CI + document for hand-runs.

## 5. Step-by-step implementation

1. Write `scripts/check-mock.mjs` (regex parse both files; list api methods + event names).
2. Assert coverage; print actionable missing lines.
3. Add missing mocks found (baseline green).
4. Wire into CI (`094`) + `pnpm --dir frontend check` script alias.
5. Rule: every bridge/event PR updates the mock in the same commit (note in `003`).

Work in small commits. After each step, run the relevant check:

```powershell
go build ./...; if ($?) { go test ./... }
cmd /c "pnpm.cmd --dir frontend build"
```

## 6. Acceptance criteria

- [ ] Check green on current tree (after filling gaps).
- [ ] Removing a mock method fails the check with its name.
- [ ] New-event guideline documented (mock emitter required).

## 7. Tests to add / update

Self-test: fixtures with intentional drift → check fails correctly.

## 8. Risks and rollback

Regex parsing of TS is brittle — keep the api table shape stable; upgrade to wailsjs-generate later if available.

Rollback: `git stash` or revert the single commit for this issue. Nothing here
touches the wire protocol unless the header says *(breaking)* — and none of the
early-phase issues do.

## 9. Follow-ups (do NOT do in this issue)

`094` (runs it), every event-adding issue (`037`/`065`) obeys it.

## 10. Touch-and-learn notes

- Read the linked files first, top to bottom, before editing.
- Note one thing that surprised you in the commit message.
- If the step takes > S (<1 day) estimate, stop and split it — open a follow-up note.
