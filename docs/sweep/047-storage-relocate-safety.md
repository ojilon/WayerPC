# 047 Relocate safety: preflight + backup + resume

| Field | Value |
|-------|-------|
| Status | `todo` (change to `doing` / `done` as you work) |
| Priority | P1 |
| Size | M (1-3 days) |
| Area | storage |
| Phase | Week 4 — storage+config |
| Related | `031`, `048` |

## 1. Why this matters

`Relocate` (`storage.go:141-169`) + `SetDataRoot` (`app.go:442-459`) move the whole tree — cross-drive via copy+swap with `RemoveAll(newRoot)` mid-flight. A crash/power-loss there eats both copies. This is the scariest code in the app.

Read this file, then touch the code. This issue is deliberately scoped so you can
finish it in one sitting and see the result in the running app.

## 2. Current behavior

Same-drive `Rename` (atomic-ish, good); cross-drive: copy to `newRoot.relocate-tmp`, `RemoveAll(newRoot)`, `Rename(tmp,newRoot)`. No preflight (space? perms?), no backup (`031`), no resume marker, no verification pass. `samePath` is Abs-compare only (case-sensitive on Windows).

Key files:

- `app.go` (Wails bridge) + `internal/` packages — see issue body for exact paths.

## 3. Problems / symptoms

- `RemoveAll(newRoot)` destroys existing destination data before the swap succeeds.
- No free-space check → half-copied tree on full disk.
- Interrupted relocate leaves `.relocate-tmp` + two half-trees, no recovery doc.

## 4. Proposal (what "good" looks like)

Preflight (dest writable? space ≥ 1.2x source? not inside source?), auto-backup catalog (`031`), copy→verify (size/count match)→swap, crash-marker file (`RELOCATE_IN_PROGRESS`) with startup recovery prompt, dry-run byte count for confirm dialog ('Move 12.4 GB to D:\…?').

## 5. Step-by-step implementation

1. Add `Preflight(newRoot)` returning human-readable blockers (space via syscall, writability probe, nesting check).
2. Settings confirm dialog shows source size + dest free + warning on cross-drive.
3. Write marker before copy, remove after; on `openRoot`, detect stale marker → banner with 'Recover / Start fresh'.
4. Verify pass: walk both trees, compare count+bytes (hash later, `026`).
5. Reuse `sameDir` case-insensitive compare (`helpers.go:27-34`) in `samePath`.
6. Test: full-disk simulation (tiny quota fixture), crash-marker recovery, nested-dest rejection.

Work in small commits. After each step, run the relevant check:

```powershell
go build ./...; if ($?) { go test ./... }
cmd /c "pnpm.cmd --dir frontend build"
```

## 6. Acceptance criteria

- [ ] Full-disk relocate aborts BEFORE deleting anything, with a clear message.
- [ ] Crash mid-relocate → startup banner, no silent half-state.
- [ ] Samsung-T5-style cross-drive move verified in test with fixtures.

## 7. Tests to add / update

`TestRelocatePreflightBlocks`, `TestRelocateCrashMarker`, cross-drive fixture move.

## 8. Risks and rollback

Highest-blast-radius code — keep diffs reviewable, test on a scratch drive letter, never on your real library first.

Rollback: `git stash` or revert the single commit for this issue. Nothing here
touches the wire protocol unless the header says *(breaking)* — and none of the
early-phase issues do.

## 9. Follow-ups (do NOT do in this issue)

`031` (backup hook), `048` (space-check helper shared).

## 10. Touch-and-learn notes

- Read the linked files first, top to bottom, before editing.
- Note one thing that surprised you in the commit message.
- If the step takes > M (1-3 days) estimate, stop and split it — open a follow-up note.
