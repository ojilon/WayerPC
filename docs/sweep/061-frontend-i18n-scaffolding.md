# 061 i18n scaffolding (extract strings, t() helper)

| Field | Value |
|-------|-------|
| Status | `todo` (change to `doing` / `done` as you work) |
| Priority | P2 |
| Size | M (1-3 days) |
| Area | frontend |
| Phase | Week 5 — frontend depth |
| Related | `063` |

## 1. Why this matters

Strings live inline per view ('Import library', 'No matches in user storage.', confirm texts). Before they spread further, extract to `src/i18n/en.ts` + `t()` so a second language (or even consistent English edits) is a data change, not a hunt.

Read this file, then touch the code. This issue is deliberately scoped so you can
finish it in one sitting and see the result in the running app.

## 2. Current behavior

`import.ts`, `library.ts`, `settings.ts`, `console.ts`, `main.ts` all inline English. No i18n file, no helper, no locale detection. Confirm/toast texts hardest to find (scattered call sites).

Key files:

- `app.go` (Wails bridge) + `internal/` packages — see issue body for exact paths.

## 3. Problems / symptoms

- Typo fix needs 5-file grep.
- Translators can't work (no catalog of strings).
- Plurals ('1 file' vs '2 files') already inconsistent.

## 4. Proposal (what "good" looks like)

Scaffolding only (this issue): `en.ts` dictionary + `t(key, vars)` + per-view migration of visible strings (no behavior change); plural helper `nt()`; locale stub (en only, `navigator.language` detected + logged for later). No second language yet — prove the pipeline with `en` + a `de`-stub of 10 keys? (decide; stub optional).

## 5. Step-by-step implementation

1. Create `src/i18n/en.ts` (keys `import.title`, `files.empty`, …) + `t()` with `{var}` interpolation + `nt()` plural.
2. Migrate one view fully (`settings.ts` smallest) as the pattern; then the other three.
3. Lint rule or checklist: no new inline user-visible strings (document; enforce by review, not tooling yet).
4. Fallback: missing key → English key name logged (never blank).
5. Keep `dev-mock.ts` strings migrated too.

Work in small commits. After each step, run the relevant check:

```powershell
go build ./...; if ($?) { go test ./... }
cmd /c "pnpm.cmd --dir frontend build"
```

## 6. Acceptance criteria

- [ ] Zero user-visible inline strings in the 4 views (grep audit).
- [ ] Language switch stub present (en locked, structure ready).
- [ ] Plurals correct for 0/1/N in migrated views.

## 7. Tests to add / update

`t()` interpolation/plural/fallback unit tests; grep-check in CI later (`094`).

## 8. Risks and rollback

Key-churn (renaming keys breaks nothing but diffs) — freeze key naming convention early (`view.element`).

Rollback: `git stash` or revert the single commit for this issue. Nothing here
touches the wire protocol unless the header says *(breaking)* — and none of the
early-phase issues do.

## 9. Follow-ups (do NOT do in this issue)

Second-language contribution guide (future); `063` onboarding copy uses the catalog.

## 10. Touch-and-learn notes

- Read the linked files first, top to bottom, before editing.
- Note one thing that surprised you in the commit message.
- If the step takes > M (1-3 days) estimate, stop and split it — open a follow-up note.
