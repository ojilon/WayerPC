# 083 Zig plan 3/10: file-walk + filter port

| Field | Value |
|-------|-------|
| Status | `todo` (change to `doing` / `done` as you work) |
| Priority | P2 |
| Size | M (1-3 days) |
| Area | zig |
| Phase | Week 7 — Zig track |
| Related | `081`, `040`, `089` |

## 1. Why this matters

Walks (`SearchRoots` `search.go:104-173`, `AddFolder` `catalog.go:120-137`, `DiskUsage` `storage.go:125-137`, `CountFiles` `064`) dominate Import/count/usage latency. Zig's explicit allocators + fewer per-entry allocs could cut walk time — second port after scorer proves the pipeline.

Read this file, then touch the code. This issue is deliberately scoped so you can
finish it in one sitting and see the result in the running app.

## 2. Current behavior

Go walks use `WalkDir` + per-entry closures + `SkipRel` segment splits (`search.go:40-50`) + stat calls. Depth/loop guards arrive via `040`. No shared walker: three call sites, three loops.

Key files:

- `app.go` (Wails bridge) + `internal/` packages — see issue body for exact paths.

## 3. Problems / symptoms

- Three loops to port vs one — port the WRONG one and gains don't transfer.
- OS APIs differ (Windows reparse points, long paths `\\?\`) — Zig std.fs parity work.
- Callback/closure boundary per entry is costly — needs batch/event design.

## 4. Proposal (what "good" looks like)

First unify Go walkers behind one `internal/walk` iterator (with skips `039`, depth+loop guards `040`, progress `037`, cancel `036`), THEN port that single iterator to Zig (C-ABI: resume-token iterator or batch-of-N-paths out-buffer). Walk stays Go-orchestrated; Zig accelerates the hot filter loop only if `089` numbers justify.

## 5. Step-by-step implementation

1. Unify Go: `internal/walk` with options (roots, skips, depth, kinds) + tests (fixtures incl. loops `040`).
2. Bench Go walker on 50k fixture (baseline for `089`).
3. Zig spike: `walk_filter` batch fn (dir fd + pattern → matches out-buffer); compare on same fixture.
4. Decide: full port vs 'Zig filter callback only' vs stay-Go (document with numbers).
5. If porting: parity tests (same tree → same hit sets, ordering normalized).

Work in small commits. After each step, run the relevant check:

```powershell
go build ./...; if ($?) { go test ./... }
cmd /c "pnpm.cmd --dir frontend build"
```

## 6. Acceptance criteria

- [ ] Single Go walker used by all 4 call sites (or documented why not).
- [ ] Zig spike numbers recorded; explicit port/stay decision.
- [ ] No behavior change in flag-off path (golden tree fixtures).

## 7. Tests to add / update

Walker golden fixtures (tree → expected hits incl. skips/loops/depth).

## 8. Risks and rollback

Windows long-path + junction semantics in Zig std — spike must include a Windows fixture run, not just Linux numbers.

Rollback: `git stash` or revert the single commit for this issue. Nothing here
touches the wire protocol unless the header says *(breaking)* — and none of the
early-phase issues do.

## 9. Follow-ups (do NOT do in this issue)

`086` (builds the walker if ported), `040` guards must exist first.

## 10. Touch-and-learn notes

- Read the linked files first, top to bottom, before editing.
- Note one thing that surprised you in the commit message.
- If the step takes > M (1-3 days) estimate, stop and split it — open a follow-up note.
