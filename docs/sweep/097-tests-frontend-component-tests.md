# 097 Frontend component tests (vitest, light touch)

| Field | Value |
|-------|-------|
| Status | `todo` (change to `doing` / `done` as you work) |
| Priority | P2 |
| Size | M (1-3 days) |
| Area | tests/frontend |
| Phase | Week 7 — tests |
| Related | `093` |

## 1. Why this matters

Frontend has zero unit tests (verify: no `*.test.ts`). Pure helpers (`fmtMB`, `fmtDate` `api.ts:146-155`, `sortRows` `library.ts:144-151`, `escapeHtml`, history rotation `045`, delta math `017`) are testable without a DOM; view-mount tests need happy-dom and pay more. Light touch: test the pure parts, mock-mount the rest once.

Read this file, then touch the code. This issue is deliberately scoped so you can
finish it in one sitting and see the result in the running app.

## 2. Current behavior

No test runner in `frontend/` (check `package.json`); `dev-mock.ts` doubles as manual harness. `pnpm build` is the only gate. `093` checks mock shape, not behavior.

Key files:

- `app.go` (Wails bridge) + `internal/` packages — see issue body for exact paths.

## 3. Problems / symptoms

- Sort/regressions (`050` ordering, `034` dates) caught by eye only.
- Refactors (i18n `061` string moves) break silently.
- No harness for delta/rate math (`017`) or history rotation (`045`).

## 4. Proposal (what "good" looks like)

Add vitest + a handful of suites: fmt/escape/sort/history/delta pure tests + ONE mount smoke per view (happy-dom, mock api, assert no-throw + key selectors present). Keep <10 files; run in `094` CI alongside `093`. Don't chase full DOM coverage — the conformance + manual QA (`098`) carry the views.

## 5. Step-by-step implementation

1. `pnpm --dir frontend add -D vitest happy-dom` (or jsdom — pick one, document).
2. Suites for the pure helpers first (instant value).
3. One smoke per view (mount with mock, assert headings/buttons exist).
4. `package.json` scripts: `test`, `check` (tsc --noEmit), wire to CI.
5. i18n `061` keys test (every key used exists — if i18n landed).

Work in small commits. After each step, run the relevant check:

```powershell
go build ./...; if ($?) { go test ./... }
cmd /c "pnpm.cmd --dir frontend build"
```

## 6. Acceptance criteria

- [ ] `pnpm --dir frontend test` green, <30s.
- [ ] Pure-helper suites cover the 5 named areas.
- [ ] 4 view smokes mount without throwing.

## 7. Tests to add / update

The suites. Snapshot tests: avoid (brittle against intentional UI churn).

## 8. Risks and rollback

DOM-test brittleness vs UI churn — keep smokes shallow (presence, not pixels). happy-dom vs WebView2 gaps documented (smoke ≠ proof).

Rollback: `git stash` or revert the single commit for this issue. Nothing here
touches the wire protocol unless the header says *(breaking)* — and none of the
early-phase issues do.

## 9. Follow-ups (do NOT do in this issue)

`098` (manual QA remains the view gate), `094` (runs this).

## 10. Touch-and-learn notes

- Read the linked files first, top to bottom, before editing.
- Note one thing that surprised you in the commit message.
- If the step takes > M (1-3 days) estimate, stop and split it — open a follow-up note.
