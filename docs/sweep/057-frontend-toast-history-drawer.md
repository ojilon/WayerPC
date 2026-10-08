# 057 Toast history drawer

| Field | Value |
|-------|-------|
| Status | `todo` (change to `doing` / `done` as you work) |
| Priority | P2 |
| Size | S (<1 day) |
| Area | frontend |
| Phase | Week 2 — frontend |
| Related | `076` |

## 1. Why this matters

Toasts vanish after 3.2s (`main.ts:21-24`). Errors ('Import failed: …', 'Could not move data') disappear before users read them — then they file 'nothing happened'.

Read this file, then touch the code. This issue is deliberately scoped so you can
finish it in one sitting and see the result in the running app.

## 2. Current behavior

`toast()` appends div, auto-removes; `confirm()` modal (`main.ts:27-40`). No history, no levels (info vs error look identical), no persistence. Callers: import results, settings moves, reveal errors.

Key files:

- `app.go` (Wails bridge) + `internal/` packages — see issue body for exact paths.

## 3. Problems / symptoms

- Error evidence evaporates.
- No distinction between FYI and failure.
- Repeated toasts (500-file watcher batch) would spam with no coalescing (`024`).

## 4. Proposal (what "good" looks like)

Toast store: keep last 50 (time+level+text); bell icon with count badge in titlebar/sidebar opens a drawer (list, dismiss-all, click-to-copy). Levels: info/success/error (color + icon). Coalesce identical bursts ('Imported… (x12)' or single batch line).

## 5. Step-by-step implementation

1. Extend `main.ts:toast(msg)` → `notify(msg, level)`; keep old signature as info wrapper.
2. History array + drawer UI (overlay panel, `confirm-overlay` pattern reuse).
3. Badge count (unread errors); opening marks read.
4. Update call sites to pass levels (errors → error).
5. Cap 50, errors pinned until dismissed (info auto-expire).

Work in small commits. After each step, run the relevant check:

```powershell
go build ./...; if ($?) { go test ./... }
cmd /c "pnpm.cmd --dir frontend build"
```

## 6. Acceptance criteria

- [ ] Error toast recoverable after it fades (drawer shows it).
- [ ] 12 rapid identical toasts coalesce (or cap visibly).
- [ ] Levels visually distinct; errors need explicit dismiss.

## 7. Tests to add / update

Store unit tests (cap, coalesce, levels); manual drawer walk.

## 8. Risks and rollback

Scope: drawer styling can balloon — reuse existing card/dialog CSS, no new design system.

Rollback: `git stash` or revert the single commit for this issue. Nothing here
touches the wire protocol unless the header says *(breaking)* — and none of the
early-phase issues do.

## 9. Follow-ups (do NOT do in this issue)

`076` (levels shared with backend log levels), `024` (batch notifications use it).

## 10. Touch-and-learn notes

- Read the linked files first, top to bottom, before editing.
- Note one thing that surprised you in the commit message.
- If the step takes > S (<1 day) estimate, stop and split it — open a follow-up note.
