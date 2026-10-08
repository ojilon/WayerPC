# 033 Catalog schema: hash + metadata columns

| Field | Value |
|-------|-------|
| Status | `todo` (change to `doing` / `done` as you work) |
| Priority | P2 |
| Size | M (1-3 days) |
| Area | backend |
| Phase | Week 6 — advanced backend |
| Related | `026`, `009` |

## 1. Why this matters

`026` dedup, `009` richer MATCHES, and future thumbnails all need columns that don't exist: `sha256`, `modtime`, `mime/kind`. Doing the migration once, cleanly, unlocks all three.

Read this file, then touch the code. This issue is deliberately scoped so you can
finish it in one sitting and see the result in the running app.

## 2. Current behavior

Schema (`catalog.go:23-36`): id/path/name/size/imported_at + settings kv. `Entry` (`catalog.go:39-48`) mirrors it. `MigrateLegacy` assumes the old 4-column shape. No migration framework — schema is `CREATE IF NOT EXISTS` only.

Key files:

- `app.go` (Wails bridge) + `internal/` packages — see issue body for exact paths.

## 3. Problems / symptoms

- Every feature adds its own ALTER; without a version table they collide.
- Backfill (hash 100k files) has no progress story.
- Downgrade path (new DB → old binary) unconsidered.

## 4. Proposal (what "good" looks like)

Add `PRAGMA user_version` migration chain: v1 baseline → v2 `sha256` → v3 `modtime/mime`. Nullable columns, idempotent `ALTER`s, version stamp. Backfill lazily (on access) + explicit 'Enrich catalog' job later. This issue = framework + columns, NOT the backfill UI.

## 5. Step-by-step implementation

1. Add `user_version` get/set + ordered migration funcs in `catalog.go`.
2. v2: `sha256 TEXT`, index; v3: `modtime TEXT`, `kind TEXT`.
3. Extend `Entry` + List/Find scans (NULL-tolerant).
4. Test: fresh open → version=latest; old v1 DB file → migrates, data intact.
5. Keep `MigrateLegacy` working against pre-migration legacy files.
6. Document downgrade note (new columns ignored by old builds, forward-only).

Work in small commits. After each step, run the relevant check:

```powershell
go build ./...; if ($?) { go test ./... }
cmd /c "pnpm.cmd --dir frontend build"
```

## 6. Acceptance criteria

- [ ] Old DB opens, migrates, all rows preserved, version stamped.
- [ ] Fresh DB has all columns.
- [ ] `026`/`009` can proceed without further schema work.

## 7. Tests to add / update

Migration tests from hand-built v1 sqlite files.

## 8. Risks and rollback

ALTER on 100k-row tables locks briefly — run at open, log duration. modernc.org/sqlite `user_version` semantics: verify.

Rollback: `git stash` or revert the single commit for this issue. Nothing here
touches the wire protocol unless the header says *(breaking)* — and none of the
early-phase issues do.

## 9. Follow-ups (do NOT do in this issue)

`026` (fills sha256), `009` (reads size/modtime), thumbnail kind detection.

## 10. Touch-and-learn notes

- Read the linked files first, top to bottom, before editing.
- Note one thing that surprised you in the commit message.
- If the step takes > M (1-3 days) estimate, stop and split it — open a follow-up note.
