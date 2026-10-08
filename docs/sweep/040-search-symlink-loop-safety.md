# 040 Symlink loop + reparse-point safety

| Field | Value |
|-------|-------|
| Status | `todo` (change to `doing` / `done` as you work) |
| Priority | P1 |
| Size | S (<1 day) |
| Area | search |
| Phase | Week 3 — search depth |
| Related | `039` |

## 1. Why this matters

`filepath.WalkDir` doesn't follow symlinks by default — but junctions/reparse points on Windows + user-created symlinks can still cycle or explode the walk (OneDrive, Dropbox, `node_modules` nesting). Today there's no visited-inode guard and no depth cap.

Read this file, then touch the code. This issue is deliberately scoped so you can
finish it in one sitting and see the result in the running app.

## 2. Current behavior

`SearchRoots` (`search.go:118-164`) and `AddFolder` (`catalog.go:125-136`) walk raw with no depth limit, no symlink resolution, no visited-set. `SkipRel` may skip some system junctions but not user loops. OneDrive `\?\` paths untested.

Key files:

- `app.go` (Wails bridge) + `internal/` packages — see issue body for exact paths.

## 3. Problems / symptoms

- Infinite or exponential walks on crafted/cyclic trees.
- Double-counting the same file via two links.
- OneDrive/Cloud placeholder files trigger downloads on stat? (check behavior).

## 4. Proposal (what "good" looks like)

Add depth cap (e.g. 32) + visited-dir guard (device+inode on unix; file-index on Windows via `os.SameFile`-ish or path-canonicalization) + don't-follow-symlinked-dirs by default (record link target instead). Log skipped loops at debug.

## 5. Step-by-step implementation

1. Add `maxDepth=32` to both walks (search + AddFolder).
2. Track visited dirs (canonical path via `EvalSymlinks` when cheap; inode map where available).
3. Symlinked dirs: record the link itself as a hit (search) / skip-descend with log.
4. Fixture test: cyclic symlink tree terminates <2s with correct hits.
5. Document OneDrive placeholder behavior found during testing.

Work in small commits. After each step, run the relevant check:

```powershell
go build ./...; if ($?) { go test ./... }
cmd /c "pnpm.cmd --dir frontend build"
```

## 6. Acceptance criteria

- [ ] Cyclic fixture terminates; no duplicates from the loop.
- [ ] Depth-40 nesting stops at 32 with a debug log.
- [ ] Normal trees unaffected (regression suite green).

## 7. Tests to add / update

Symlink-loop fixture test (skip on filesystems without symlink perms, with clear skip message).

## 8. Risks and rollback

Windows symlink/Junction APIs differ — keep guard best-effort + depth cap as the hard guarantee.

Rollback: `git stash` or revert the single commit for this issue. Nothing here
touches the wire protocol unless the header says *(breaking)* — and none of the
early-phase issues do.

## 9. Follow-ups (do NOT do in this issue)

Catalog `AddFolder` shares the guard (same helper).

## 10. Touch-and-learn notes

- Read the linked files first, top to bottom, before editing.
- Note one thing that surprised you in the commit message.
- If the step takes > S (<1 day) estimate, stop and split it — open a follow-up note.
