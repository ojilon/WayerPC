# 056 Console: copy-logs + download-logs buttons

| Field | Value |
|-------|-------|
| Status | `todo` (change to `doing` / `done` as you work) |
| Priority | P2 |
| Size | S (<1 day) |
| Area | frontend |
| Phase | Week 1 — frontend quick wins |
| Related | `025`, `091` |

## 1. Why this matters

The field debugging loop needed the user's log and had no one-click way to get it. Filter + pause + clear exist (`console.ts:36-44`); copy doesn't. Smallest UX change with direct debugging payoff.

Read this file, then touch the code. This issue is deliberately scoped so you can
finish it in one sitting and see the result in the running app.

## 2. Current behavior

`c-filter/c-pause/c-clear` wired (`console.ts:91-101`); `GetLogs/ClearLogs` bridged (`app.go:202-215`). No clipboard, no download. Toasts confirm actions elsewhere (`main.ts:14-25`).

Key files:

- `app.go` (Wails bridge) + `internal/` packages — see issue body for exact paths.

## 3. Problems / symptoms

- Bug reports arrive as screenshots of logs (unsearchable).
- No way to capture post-restart history (needs `025` too).
- Pause+filter state doesn't transfer to the copied text (should it? decide).

## 4. Proposal (what "good" looks like)

Add Copy (filtered? full? — offer both: copies current view incl. filter) + Download (.log file, Wails save dialog or browser blob fallback). Toast confirms with line count. Respect pause state (copy what's shown).

## 5. Step-by-step implementation

1. Copy button: gather `logEl` lines (or `api.getLogs` + apply filter) → `navigator.clipboard.writeText` with execCommand fallback.
2. Download: `api.getLogs` → blob/file save (Wails dialog if available, else anchor download).
3. Toast 'Copied N lines' / errors surfaced (clipboard perms).
4. Works in browser-preview mode (`dev-mock.ts`) too.
5. Document in bug-report template ('paste Console export').

Work in small commits. After each step, run the relevant check:

```powershell
go build ./...; if ($?) { go test ./... }
cmd /c "pnpm.cmd --dir frontend build"
```

## 6. Acceptance criteria

- [ ] One click copies the filtered view; toast shows count.
- [ ] Download produces a .log that opens in Notepad.
- [ ] Clipboard-denied shows guidance, not silence.

## 7. Tests to add / update

Manual matrix (Wails + browser preview); filter-applies-to-copy test on pure func.

## 8. Risks and rollback

Clipboard API perms in WebView2 — provide download fallback always.

Rollback: `git stash` or revert the single commit for this issue. Nothing here
touches the wire protocol unless the header says *(breaking)* — and none of the
early-phase issues do.

## 9. Follow-ups (do NOT do in this issue)

`025` (download includes persisted history), issue template referencing the export.

## 10. Touch-and-learn notes

- Read the linked files first, top to bottom, before editing.
- Note one thing that surprised you in the commit message.
- If the step takes > S (<1 day) estimate, stop and split it — open a follow-up note.
