# 022 Cache DiskUsage (stop walking the tree every second)

| Field | Value |
|-------|-------|
| Status | `todo` (change to `doing` / `done` as you work) |
| Priority | P0 |
| Size | S (<1 day) |
| Area | backend |
| Phase | Week 1 — backend reliability |
| Related | `017` |

## 1. Why this matters

`GetStatus()` walks the whole data tree on every 1s stats tick (`app.go:164-175` → `store.DiskUsage()` → full `WalkDir`, `storage.go:125-137`) plus Console polling every 2s (`console.ts:113`). Big libraries burn real CPU for a number that barely changes.

Read this file, then touch the code. This issue is deliberately scoped so you can
finish it in one sitting and see the result in the running app.

## 2. Current behavior

`DiskUsage()` sums every file under root per call. Called from `GetStatus` (`app.go:242`) and `GetStorageInfo` (`app.go:434`). No cache, no invalidation — O(tree) per second.

Key files:

- `app.go` (Wails bridge) + `internal/` packages — see issue body for exact paths.

## 3. Problems / symptoms

- CPU + disk churn on every tick; laptop fans spin on 100k-file libraries.
- Duplicated walks: statsLoop (1s) + Console poll (2s) + Settings paint.
- Mobile-hotspot machines are often low-power — waste matters.

## 4. Proposal (what "good" looks like)

Cache the total in `Store` (or `App`); invalidate on upload/import/remove/relocate; refresh async (debounced, e.g. ≤1/5s) instead of per-tick. Show 'calculating…' only on first run. Trivial, saves real CPU.

## 5. Step-by-step implementation

1. Add `cachedUsage int64` + `usageMu` + `usageDirty` to `Store` (or App-level cache).
2. Invalidate in: `handleUpload` success (`protocol.go:170-174`), `CatalogAddPaths/AddFolder/Remove` (`app.go:333-391`), `Relocate` (`storage.go:141-169`).
3. Background refresh (singleflight: concurrent Gets share one walk).
4. Keep `DiskUsage()` API; add `CachedDiskUsage()` for hot paths.
5. Benchmark before/after on `.data/` stub + a 10k-file fixture.
6. Console: no visible change except lower CPU (verify in Task Manager).

Work in small commits. After each step, run the relevant check:

```powershell
go build ./...; if ($?) { go test ./... }
cmd /c "pnpm.cmd --dir frontend build"
```

## 6. Acceptance criteria

- [ ] 1s tick causes zero filesystem walks when idle.
- [ ] After upload/import, usage updates within 5s.
- [ ] Benchmark numbers recorded in the commit message.

## 7. Tests to add / update

`TestDiskUsageCacheInvalidation` (add file → dirty → refresh returns new total).

## 8. Risks and rollback

Stale reads if external tools modify the tree — mitigate with 30s max-age refresh.

Rollback: `git stash` or revert the single commit for this issue. Nothing here
touches the wire protocol unless the header says *(breaking)* — and none of the
early-phase issues do.

## 9. Follow-ups (do NOT do in this issue)

`078` (usage history sparkline later).

## 10. Touch-and-learn notes

- Read the linked files first, top to bottom, before editing.
- Note one thing that surprised you in the commit message.
- If the step takes > S (<1 day) estimate, stop and split it — open a follow-up note.
