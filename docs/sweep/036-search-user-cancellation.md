# 036 Cancellable user search

| Field | Value |
|-------|-------|
| Status | `todo` (change to `doing` / `done` as you work) |
| Priority | P1 |
| Size | S (<1 day) |
| Area | search |
| Phase | Week 3 — search depth |
| Related | `037`, `064` |

## 1. Why this matters

User-storage walks (`search.go:104-173`, whole drives D:–Z:) take minutes. No cancel exists: Import view `search()` (`import.ts:85-100`) awaits one promise; user stares or kills the app. 'Internal file search stuff' starts here.

Read this file, then touch the code. This issue is deliberately scoped so you can
finish it in one sitting and see the result in the running app.

## 2. Current behavior

`SearchRoots` walks synchronously to `limit` with no context; `App.SearchUser` (`app.go:286-295`) has no cancellation token. Frontend `busy` flag blocks re-search but can't abort the in-flight walk. `readyDrives` walks entire drives.

Key files:

- `app.go` (Wails bridge) + `internal/` packages — see issue body for exact paths.

## 3. Problems / symptoms

- Accidental 'e' query (matches everything) locks the UI path for minutes.
- No way to refine the query mid-walk.
- Goroutine + file handles held pointlessly after user navigates away.

## 4. Proposal (what "good" looks like)

Thread `context.Context` from Wails through `SearchUser`→`SearchRoots`: check `ctx.Done()` per directory (every ~64 entries); return partial hits with `cancelled=true`. Frontend: Search button becomes Cancel while busy; navigating away cancels.

## 5. Step-by-step implementation

1. Add `SearchRootsCtx(ctx, roots, query, limit)`; keep old wrapper for tests.
2. Check ctx every directory visit + every 64 files (cheap counter).
3. Bridge: `App.SearchUser` uses request ctx (Wails provides per-call ctx? if not, add cancel func map keyed by call — simplest: `App.CancelSearch()` + stored cancel).
4. `import.ts`: busy→Cancel button swap; show 'cancelled — N partial hits'.
5. Test: cancel mid-walk on big temp tree → fast return, partial hits.

Work in small commits. After each step, run the relevant check:

```powershell
go build ./...; if ($?) { go test ./... }
cmd /c "pnpm.cmd --dir frontend build"
```

## 6. Acceptance criteria

- [ ] 60s walk cancellable in <300ms with partial results.
- [ ] Cancel button + Esc both work.
- [ ] No goroutine/handle leak (test with -race + fd check).

## 7. Tests to add / update

Cancellation test (slow fixture, cancel at 100ms).

## 8. Risks and rollback

Wails ctx semantics per call — verify; fallback to explicit Cancel method is fine.

Rollback: `git stash` or revert the single commit for this issue. Nothing here
touches the wire protocol unless the header says *(breaking)* — and none of the
early-phase issues do.

## 9. Follow-ups (do NOT do in this issue)

`037` (progress rides the same walk loop), `043` (index makes cancel rarer).

## 10. Touch-and-learn notes

- Read the linked files first, top to bottom, before editing.
- Note one thing that surprised you in the commit message.
- If the step takes > S (<1 day) estimate, stop and split it — open a follow-up note.
