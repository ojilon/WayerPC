# 043 Indexed filename cache (SQLite FTS)

| Field | Value |
|-------|-------|
| Status | `todo` (change to `doing` / `done` as you work) |
| Priority | P2 |
| Size | L (3+ days) |
| Area | search/backend |
| Phase | Week 8 — advanced (defer) |
| Related | `027`, `042` |

## 1. Why this matters

Every Import search re-walks whole drives — minutes, every time. A persistent filename index (SQLite FTS over last-known tree + background refresh) turns repeat searches into milliseconds. This is the 'internal file search stuff' endgame — and the most complex, so it goes last.

Read this file, then touch the code. This issue is deliberately scoped so you can
finish it in one sitting and see the result in the running app.

## 2. Current behavior

No index: `UserRoots` (`search.go:57-82`) + full walk per query. `027` adds FTS for the *catalog* (imported paths) — this extends the idea to the *whole user tree*. Watcher (`024`) covers `shared/` only.

Key files:

- `app.go` (Wails bridge) + `internal/` packages — see issue body for exact paths.

## 3. Problems / symptoms

- Repeat searches pay full walk cost.
- Stale results vs fresh walks tradeoff unexamined.
- Background indexing burns IO if unbounded — needs budget + toggle.

## 4. Proposal (what "good" looks like)

DEFERRED design: `search_index` table (path, name, size, mtime); indexer walks roots on idle/schedule (or on demand 'Build index'); queries hit FTS first, fall back to live walk when index stale/missing; staleness badge in UI. Decide after `027` proves FTS ops. This issue = schema + manual-build path only; background scheduling is the follow-up.

## 5. Step-by-step implementation

1. Read `027` outcome; reuse its FTS pattern.
2. Schema: `search_index` + triggers or batch rebuild (prefer rebuild-by-root for simplicity).
3. `BuildIndex(roots)` with progress (`037` pattern) + cancel (`036`).
4. Query path: FTS first (flag `fromIndex`), live-walk fallback.
5. UI: 'Rebuild index' button + staleness line ('indexed 2h ago').
6. Measure: indexed vs walk on 50k fixture; document IO cost.

Work in small commits. After each step, run the relevant check:

```powershell
go build ./...; if ($?) { go test ./... }
cmd /c "pnpm.cmd --dir frontend build"
```

## 6. Acceptance criteria

- [ ] Indexed repeat query <200ms on 50k fixture.
- [ ] Stale index clearly labeled; fallback works with index deleted.
- [ ] Index toggle + rebuild UI present.

## 7. Tests to add / update

Index build/query/staleness tests on fixtures.

## 8. Risks and rollback

Biggest complexity in the sweep — timebox strictly; ship manual-rebuild before any auto-schedule.

Rollback: `git stash` or revert the single commit for this issue. Nothing here
touches the wire protocol unless the header says *(breaking)* — and none of the
early-phase issues do.

## 9. Follow-ups (do NOT do in this issue)

Auto-refresh scheduler (separate issue); `042` content index (only after this proves out).

## 10. Touch-and-learn notes

- Read the linked files first, top to bottom, before editing.
- Note one thing that surprised you in the commit message.
- If the step takes > L (3+ days) estimate, stop and split it — open a follow-up note.
