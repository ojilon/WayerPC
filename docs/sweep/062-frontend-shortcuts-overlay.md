# 062 Shortcuts overlay + focus-visible audit

| Field | Value |
|-------|-------|
| Status | `todo` (change to `doing` / `done` as you work) |
| Priority | P2 |
| Size | S (<1 day) |
| Area | frontend |
| Phase | Week 2 — frontend |
| Related | `067` |

## 1. Why this matters

`Ctrl+1..4` view switching exists (`main.ts:114-122`) but is undiscoverable. A `?` cheat-sheet + focus-visible audit makes keyboard use real for almost free.

Read this file, then touch the code. This issue is deliberately scoped so you can
finish it in one sitting and see the result in the running app.

## 2. Current behavior

Key handler only; no help UI, no focus styles audit, no per-view shortcuts (Enter-to-search exists implicitly in import). `titlebar` buttons lack titles except min/max/close have some.

Key files:

- `app.go` (Wails bridge) + `internal/` packages — see issue body for exact paths.

## 3. Problems / symptoms

- Power users never find Ctrl+1..4.
- Tab navigation invisible (focus ring missing/suppressed?).
- No documented per-view keys (/, Enter, Esc-cancel `036`).

## 4. Proposal (what "good" looks like)

`?` (or Ctrl+/) opens a cheat-sheet dialog listing global + contextual shortcuts; add per-view keys where cheap (Import: `/` focuses search, Esc cancels per `036`); ensure `:focus-visible` rings everywhere; document in dialog, not elsewhere.

## 5. Step-by-step implementation

1. Shortcuts dialog (reuse `confirm-overlay` dialog pattern) with grouped table.
2. Wire `?`/Ctrl+/ toggle; Esc closes (unless a search-cancel claims it — define precedence).
3. CSS `:focus-visible` audit: tab through all 4 views, fix invisible foci.
4. Add `/`-focus-search in Import; document Ctrl+1..4 + new keys.
5. Titles/aria-labels on icon-ish buttons.

Work in small commits. After each step, run the relevant check:

```powershell
go build ./...; if ($?) { go test ./... }
cmd /c "pnpm.cmd --dir frontend build"
```

## 6. Acceptance criteria

- [ ] `?` opens the sheet from any view; Esc closes.
- [ ] Full keyboard run: switch views, search, import, reveal — no mouse.
- [ ] Focus always visible when tabbing.

## 7. Tests to add / update

Manual keyboard run script (write it down in the file after verifying).

## 8. Risks and rollback

Shortcut collisions with WebView/IME — keep set tiny (Ctrl+1..4, /, ?, Esc, Enter).

Rollback: `git stash` or revert the single commit for this issue. Nothing here
touches the wire protocol unless the header says *(breaking)* — and none of the
early-phase issues do.

## 9. Follow-ups (do NOT do in this issue)

`067` (full a11y audit builds on this).

## 10. Touch-and-learn notes

- Read the linked files first, top to bottom, before editing.
- Note one thing that surprised you in the commit message.
- If the step takes > S (<1 day) estimate, stop and split it — open a follow-up note.
