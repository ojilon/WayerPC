# 041 Extension + size filters for import search

| Field | Value |
|-------|-------|
| Status | `todo` (change to `doing` / `done` as you work) |
| Priority | P2 |
| Size | S (<1 day) |
| Area | search/frontend |
| Phase | Week 4 — search depth |
| Related | `046`, `050` |

## 1. Why this matters

'photo' matching 4,000 files including `.dll`s and 10GB ISOs is noise. Two dropdowns (type: any/image/video/doc/audio; min/max size) cut Import results to what the user actually wants to register.

Read this file, then touch the code. This issue is deliberately scoped so you can
finish it in one sitting and see the result in the running app.

## 2. Current behavior

`SearchUser(query, limit)` (`app.go:286-295`) takes no filters; `Hit` (`search.go:16-18`) has name+path only (no size). Frontend (`import.ts`) has no filter UI. `FileInfo` has sizes but that's the Files view, not search.

Key files:

- `app.go` (Wails bridge) + `internal/` packages — see issue body for exact paths.

## 3. Problems / symptoms

- Noisy results; best hits truncated by the 200-limit (`038`).
- Huge files surprise-imported with no size context.
- No size signal means `038` ranking can't prefer sensibly-sized hits.

## 4. Proposal (what "good" looks like)

Add `Filter{Exts, MinSize, MaxSize, Kind}` to search; extend `Hit` with `size` (stat during walk — already paying for `DirEntry`); Import view filter row (kind select + max-size input). Server-side filtering (not client) so the 200-limit applies post-filter.

## 5. Step-by-step implementation

1. Extend `Hit` with `Size int64` (+ `IsDir`? decide, default file-only toggle).
2. `SearchRootsFiltered(roots, query, limit, filter)`; keep old signature wrapping it.
3. `import.ts`: filter row UI; persist last filter in setting kv.
4. Show size per hit (reuse `fmtMB`).
5. Tests: kind=image returns only image exts; max-size excludes ISO fixture.

Work in small commits. After each step, run the relevant check:

```powershell
go build ./...; if ($?) { go test ./... }
cmd /c "pnpm.cmd --dir frontend build"
```

## 6. Acceptance criteria

- [ ] Filtered search returns ⊆ unfiltered, all matching filter.
- [ ] Sizes shown per hit.
- [ ] Filter persists across view switches (session).

## 7. Tests to add / update

Filter unit tests on fixture tree with known sizes/exts.

## 8. Risks and rollback

Stat-per-file slows walks slightly — measure; `DirEntry.Info()` is usually cached/cheap.

Rollback: `git stash` or revert the single commit for this issue. Nothing here
touches the wire protocol unless the header says *(breaking)* — and none of the
early-phase issues do.

## 9. Follow-ups (do NOT do in this issue)

`046` (preview uses size/kind), `050` (Files sorting parity).

## 10. Touch-and-learn notes

- Read the linked files first, top to bottom, before editing.
- Note one thing that surprised you in the commit message.
- If the step takes > S (<1 day) estimate, stop and split it — open a follow-up note.
