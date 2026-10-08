# 050 Files sorting modes that match the catalog

| Field | Value |
|-------|-------|
| Status | `todo` (change to `doing` / `done` as you work) |
| Priority | P2 |
| Size | S (<1 day) |
| Area | storage/frontend |
| Phase | Week 4 — frontend depth |
| Related | `034`, `028` |

## 1. Why this matters

`ListFiles` (`storage.go:79-122`) sorts files-before-folders then by name; the UI re-sorts by date/size/name (`library.ts:144-151`) using string-compared dates. Two sorters, neither total, disagreeing with catalog ordering (`034`).

Read this file, then touch the code. This issue is deliberately scoped so you can
finish it in one sitting and see the result in the running app.

## 2. Current behavior

Backend name-sort; frontend `sortRows` with `localeCompare` on ISO date strings and numeric size. Dirs excluded from display (`library.ts:100` filters `!f.isDir`) but included in payload. No 'folders first' toggle; no stable tiebreak.

Key files:

- `app.go` (Wails bridge) + `internal/` packages — see issue body for exact paths.

## 3. Problems / symptoms

- Same-folder listing order differs between Received/Shared/Catalog segments.
- Date ties (same-second uploads) shuffle randomly.
- Payload includes dirs the UI drops (wasted bridge bytes).

## 4. Proposal (what "good" looks like)

Single documented order: explicit sort param (`name/size/date`, asc/desc) executed backend-side with `(key, id/name)` tiebreak; UI becomes a thin control. Option to include/exclude dirs. Keep frontend `sortRows` as fallback for mock mode only.

## 5. Step-by-step implementation

1. Add `ListFilesSorted(dir, sort, desc, includeDirs)`; keep `ListFiles` wrapper (default current behavior).
2. Parse real `ModTime` (already `time.Time` in `FileInfo`) — stop string-comparing dates anywhere.
3. UI sort control sends param instead of re-sorting (mock implements same). 
4. Fixture test: ties broken deterministically across 3 runs.
5. Measure bridge payload before/after (dirs excluded).

Work in small commits. After each step, run the relevant check:

```powershell
go build ./...; if ($?) { go test ./... }
cmd /c "pnpm.cmd --dir frontend build"
```

## 6. Acceptance criteria

- [ ] All three segments share one ordering semantic.
- [ ] Tie order stable across restarts.
- [ ] Date sort uses timestamps, not strings.

## 7. Tests to add / update

Sort-matrix tests (name/size/date × asc/desc + ties).

## 8. Risks and rollback

Changing default order surprises existing users — keep default (name, files-first) identical; only add determinism.

Rollback: `git stash` or revert the single commit for this issue. Nothing here
touches the wire protocol unless the header says *(breaking)* — and none of the
early-phase issues do.

## 9. Follow-ups (do NOT do in this issue)

`028` (catalog pager ordering uses the same tiebreak rule).

## 10. Touch-and-learn notes

- Read the linked files first, top to bottom, before editing.
- Note one thing that surprised you in the commit message.
- If the step takes > S (<1 day) estimate, stop and split it — open a follow-up note.
