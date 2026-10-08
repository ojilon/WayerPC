# 037 Search progress events

| Field | Value |
|-------|-------|
| Status | `todo` (change to `doing` / `done` as you work) |
| Priority | P1 |
| Size | M (1-3 days) |
| Area | search/frontend |
| Phase | Week 3 — search depth |
| Related | `036`, `065` |

## 1. Why this matters

Long walks give zero feedback (`import.ts:88` status is static 'Searching for…' until done). Users can't tell 10% from 90%. The backend already has the hook point (per-directory callback in the walk).

Read this file, then touch the code. This issue is deliberately scoped so you can
finish it in one sitting and see the result in the running app.

## 2. Current behavior

`SearchRoots` has no progress callback; `app.go:logf/emit` supports `runtime.EventsEmit` (`app.go:194-199`) with `log`/`stats`/`catalog-changed` — adding `search-progress` follows the same pattern. No determinate progress anywhere in Import.

Key files:

- `app.go` (Wails bridge) + `internal/` packages — see issue body for exact paths.

## 3. Problems / symptoms

- 'Is it stuck?' → user kills healthy searches.
- No dirs-visited / hits-so-far signal.
- Indeterminate spinner is the best the UI could do today (and doesn't).

## 4. Proposal (what "good" looks like)

Stream `search-progress {visited, hits, currentDir}` events (throttled ~4/s); Import status bar shows '…1,240 folders · 37 hits · D:\Photos' + determinate-ish bar (visited vs estimated) or indeterminate + counts. Reuse for `065` import progress pattern.

## 5. Step-by-step implementation

1. Add `Progress func(visited, hits int, dir string)` param (or channel) to walk; throttle emits to 250ms.
2. Emit `search-progress` from `App` (new event name; document alongside `log`/`stats`).
3. `import.ts`: subscribe on search start, unsubscribe on done/cancel; render counts + current dir (truncated).
4. Cap event rate; ensure no emit after cancel (`036`).
5. Mock in `dev-mock.ts` (fake progressive events).

Work in small commits. After each step, run the relevant check:

```powershell
go build ./...; if ($?) { go test ./... }
cmd /c "pnpm.cmd --dir frontend build"
```

## 6. Acceptance criteria

- [ ] 30s search shows live counts + current folder.
- [ ] Event rate ≤5/s (no UI flood).
- [ ] Cancel stops events immediately.

## 7. Tests to add / update

Progress-callback test (counts monotonic); frontend mock-event smoke.

## 8. Risks and rollback

Event flood on fast SSDs — throttle server-side, not in UI.

Rollback: `git stash` or revert the single commit for this issue. Nothing here
touches the wire protocol unless the header says *(breaking)* — and none of the
early-phase issues do.

## 9. Follow-ups (do NOT do in this issue)

`065` (same pattern for imports), `093` (mock must include the new event).

## 10. Touch-and-learn notes

- Read the linked files first, top to bottom, before editing.
- Note one thing that surprised you in the commit message.
- If the step takes > M (1-3 days) estimate, stop and split it — open a follow-up note.
