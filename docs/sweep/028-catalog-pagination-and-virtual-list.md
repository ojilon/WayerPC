# 028 Catalog pagination + virtualized list

| Field | Value |
|-------|-------|
| Status | `todo` (change to `doing` / `done` as you work) |
| Priority | P2 |
| Size | M (1-3 days) |
| Area | backend/frontend |
| Phase | Week 4 — frontend depth |
| Related | `009`, `059` |

## 1. Why this matters

`CatalogList` (`app.go:315-330`) returns ALL rows; the Files view renders ALL rows (`library.ts:68-92`). At 10k imports the UI freezes; at 100k it dies. The backend and the DOM both need paging.

Read this file, then touch the code. This issue is deliberately scoped so you can
finish it in one sitting and see the result in the running app.

## 2. Current behavior

`cat.List()` → full slice → JSON over Wails → `innerHTML` per row. No limit/offset, no search-within-catalog, no virtualization. Sort happens client-side (`sortRows`).

Key files:

- `app.go` (Wails bridge) + `internal/` packages — see issue body for exact paths.

## 3. Problems / symptoms

- UI jank/OOM on big catalogs — the exact libraries `027` targets.
- Wails JSON bridge chokes on huge payloads.
- No 'search my imports' — only phone-side `/ask` searches.

## 4. Proposal (what "good" looks like)

Backend: `CatalogListPaged(limit, offset, query, sort)`; frontend: paged fetch + simple virtualization (render visible window) or at minimum page controls (100/page). Add catalog-local search box (substring first, FTS later via `027`).

## 5. Step-by-step implementation

1. Add paged query to `catalog.DB` (`LIMIT/OFFSET` + optional `LIKE` + ORDER BY).
2. Add `App.CatalogListPaged` bridge (keep old `CatalogList` for compat/mock).
3. Files view: pager (Prev/Next, total count) + search input; 100 rows/page default.
4. Measure: 50k-row catalog renders page 1 in <300ms.
5. Update `dev-mock.ts` with paged mock.
6. Keep `catalog-changed` → refresh current page.

Work in small commits. After each step, run the relevant check:

```powershell
go build ./...; if ($?) { go test ./... }
cmd /c "pnpm.cmd --dir frontend build"
```

## 6. Acceptance criteria

- [ ] 50k catalog: Files view interactive, page turn <300ms.
- [ ] Search-within-catalog returns in <200ms (LIKE path).
- [ ] Old callers unaffected.

## 7. Tests to add / update

Paged-query tests (offsets, ordering, LIKE); frontend smoke with mock 10k rows.

## 8. Risks and rollback

Sorting consistency between pages (ORDER BY must be total) — include id tiebreak.

Rollback: `git stash` or revert the single commit for this issue. Nothing here
touches the wire protocol unless the header says *(breaking)* — and none of the
early-phase issues do.

## 9. Follow-ups (do NOT do in this issue)

`027` (FTS powers the search box), `059` (thumbnails per page).

## 10. Touch-and-learn notes

- Read the linked files first, top to bottom, before editing.
- Note one thing that surprised you in the commit message.
- If the step takes > M (1-3 days) estimate, stop and split it — open a follow-up note.
