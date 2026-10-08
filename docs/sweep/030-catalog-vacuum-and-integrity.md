# 030 Vacuum + integrity check routine

| Field | Value |
|-------|-------|
| Status | `todo` (change to `doing` / `done` as you work) |
| Priority | P2 |
| Size | S (<1 day) |
| Area | backend |
| Phase | Week 5 — hardening |
| Related | `023`, `031` |

## 1. Why this matters

SQLite grows with churn (deletes, re-imports, migrations). No `VACUUM`, no `PRAGMA integrity_check` anywhere — corruption (rare) would surface as bizarre `/ask` misses with no diagnosis.

Read this file, then touch the code. This issue is deliberately scoped so you can
finish it in one sitting and see the result in the running app.

## 2. Current behavior

`Open` runs schema only (`catalog.go:59-86`). No maintenance entry point. Settings shows `dbPath` (`settings.ts:51`) but no health. `MigrateLegacy` touches old DBs without checking them.

Key files:

- `app.go` (Wails bridge) + `internal/` packages — see issue body for exact paths.

## 3. Problems / symptoms

- DB bloat over months of import/remove cycles.
- No early warning for corruption (disk-full writes, unclean shutdowns).
- Users can't answer 'is my catalog healthy?'.

## 4. Proposal (what "good" looks like)

Add `DB.Maintenance()` (`VACUUM` + `integrity_check` + stats) exposed as `App.CatalogMaintenance()`; run integrity (cheap, read-only) on open, VACUUM only on demand (button in Settings: 'Optimize catalog'). Log results.

## 5. Step-by-step implementation

1. Add `Maintenance()` returning `{sizeBefore, sizeAfter, integrity}`.
2. Call `integrity_check` (not VACUUM) in `Open`; log warning on failure.
3. Settings button + log line; guard concurrent runs with mutex.
4. Test: fragment fixture (insert+delete 1k) → VACUUM shrinks file.
5. Document when to press the button (rarely).

Work in small commits. After each step, run the relevant check:

```powershell
go build ./...; if ($?) { go test ./... }
cmd /c "pnpm.cmd --dir frontend build"
```

## 6. Acceptance criteria

- [ ] Corrupt fixture → open logs a clear warning (not a panic).
- [ ] Fragmented DB shrinks after maintenance.
- [ ] Button exists in Settings with result toast.

## 7. Tests to add / update

Integrity-fail test (crafted bad page? or mock); vacuum-shrinks test.

## 8. Risks and rollback

VACUUM rewrites the whole file — needs free space ~DB size; warn if disk low (`048`). Never auto-VACUUM on startup.

Rollback: `git stash` or revert the single commit for this issue. Nothing here
touches the wire protocol unless the header says *(breaking)* — and none of the
early-phase issues do.

## 9. Follow-ups (do NOT do in this issue)

`031` (backup before vacuum on big DBs).

## 10. Touch-and-learn notes

- Read the linked files first, top to bottom, before editing.
- Note one thing that surprised you in the commit message.
- If the step takes > S (<1 day) estimate, stop and split it — open a follow-up note.
