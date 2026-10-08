# 031 Catalog backup and restore

| Field | Value |
|-------|-------|
| Status | `todo` (change to `doing` / `done` as you work) |
| Priority | P2 |
| Size | M (1-3 days) |
| Area | backend/frontend |
| Phase | Week 5 — hardening |
| Related | `030`, `047` |

## 1. Why this matters

The catalog IS the library (paths, settings). One bad relocate or disk-full write and months of imports vanish. SQLite backup API makes safe copies trivial — but nobody wired it.

Read this file, then touch the code. This issue is deliberately scoped so you can
finish it in one sitting and see the result in the running app.

## 2. Current behavior

No backup. `Relocate` (`storage.go:141-169`) moves the whole tree including the DB with no snapshot. `MigrateLegacy` reads the old DB without copying it first. Settings has no export.

Key files:

- `app.go` (Wails bridge) + `internal/` packages — see issue body for exact paths.

## 3. Problems / symptoms

- The catalog is months of curation with zero copies: one bad relocate or disk-full write loses it.
- `Relocate` moves the live DB with no snapshot; `MigrateLegacy` reads the legacy DB without copying it first.
- No documented recovery: nobody can answer "where is my backup?" — because there is none.

## 4. Proposal (what "good" looks like)

`App.CatalogBackup(destPath)` via SQLite `VACUUM INTO` (or file copy after checkpoint); auto-backup before `Relocate` + before `MigrateLegacy`; Settings shows 'Backup catalog' + 'Restore…' (restore = close, replace, reopen). Keep last 3 auto-backups.

## 5. Step-by-step implementation

1. Implement backup (prefer `VACUUM INTO 'file'` — consistent snapshot, version-dependent; fallback: checkpoint + copy).
2. Hook: auto-backup to `Data/backups/` before relocate/migrate (prune >3).
3. Settings UI: Backup now (file picker) + Restore (confirm + reopen via `openRoot`).
4. Test: backup → drop table → restore → rows back.
5. Document recovery recipe in README troubleshooting.

Work in small commits. After each step, run the relevant check:

```powershell
go build ./...; if ($?) { go test ./... }
cmd /c "pnpm.cmd --dir frontend build"
```

## 6. Acceptance criteria

- [ ] One-click backup produces a valid DB openable by sqlite3.
- [ ] Relocate leaves a timestamped backup behind.
- [ ] Restore round-trip verified in test.

## 7. Tests to add / update

Backup/restore round-trip test; prune-old-backups test.

## 8. Risks and rollback

Restore replaces live DB — confirm dialog + auto pre-restore backup. Disk-full during backup: fail cleanly, never truncate live DB.

Rollback: `git stash` or revert the single commit for this issue. Nothing here
touches the wire protocol unless the header says *(breaking)* — and none of the
early-phase issues do.

## 9. Follow-ups (do NOT do in this issue)

`047` (relocate uses this), `099` (note backup location in release notes).

## 10. Touch-and-learn notes

- Read the linked files first, top to bottom, before editing.
- Note one thing that surprised you in the commit message.
- If the step takes > M (1-3 days) estimate, stop and split it — open a follow-up note.
