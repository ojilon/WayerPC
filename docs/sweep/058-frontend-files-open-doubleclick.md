# 058 Files: double-click to open + Enter key

| Field | Value |
|-------|-------|
| Status | `todo` (change to `doing` / `done` as you work) |
| Priority | P2 |
| Size | S (<1 day) |
| Area | frontend |
| Phase | Week 2 — frontend |
| Related | `051`, `059` |

## 1. Why this matters

Files view rows (`library.ts:44-55`) offer only Reveal + Remove. Opening needs Explorer → navigate → double-click. Default-app open is one bridge call away (`RevealInExplorer` sibling).

Read this file, then touch the code. This issue is deliberately scoped so you can
finish it in one sitting and see the result in the running app.

## 2. Current behavior

`action()` helper (`library.ts:57-66`) wires buttons; no dblclick, no keyboard, no Open. Backend can already launch (`exec.Command` in `RevealInExplorer`); opening is `rundll32`/shellexec variant.

Key files:

- `app.go` (Wails bridge) + `internal/` packages — see issue body for exact paths.

## 3. Problems / symptoms

- Reveal-then-open is two steps for the common case.
- Keyboard users can't operate the list at all (`067`).
- Missing files (`032`) would attempt open and fail raw.

## 4. Proposal (what "good" looks like)

Double-click / Enter opens with the default app (new `App.OpenFile` bridge: validated open, missing → typed error, no arbitrary-exec). Single-click selects (pairs with `046` preview later). Missing rows can't be opened (disabled + tooltip). Confirm nothing — opening is non-destructive.

## 5. Step-by-step implementation

1. Backend `OpenFile(path)`: stat (missing → error), OS open (`rundll32 url.dll,FileProtocolHandler` / `open` / `xdg-open`), log `[OPEN]`.
2. Frontend: dblclick + Enter + explicit Open button per row.
3. Missing rows: button disabled, tooltip 'missing on disk'.
4. Security note: open only (no args, no shell); validate path is file.
5. Test open with mock opener (injectable func) — never spawn apps in CI.

Work in small commits. After each step, run the relevant check:

```powershell
go build ./...; if ($?) { go test ./... }
cmd /c "pnpm.cmd --dir frontend build"
```

## 6. Acceptance criteria

- [ ] Double-click opens a .txt in Notepad (manual Windows verify).
- [ ] Missing file shows disabled Open with tooltip.
- [ ] No shell-injection (args never concatenated; review + test with `a&b.txt` name).

## 7. Tests to add / update

Opener tests with fake exec; traversal-name safety test.

## 8. Risks and rollback

Auto-open is also auto-execute for .exe — warn once ('Open runs files — only open what you trust') or restrict Open to non-executable kinds; decide explicitly.

Rollback: `git stash` or revert the single commit for this issue. Nothing here
touches the wire protocol unless the header says *(breaking)* — and none of the
early-phase issues do.

## 9. Follow-ups (do NOT do in this issue)

`059` (thumbnails make open-targets identifiable), `067` (keyboard map documents Enter).

## 10. Touch-and-learn notes

- Read the linked files first, top to bottom, before editing.
- Note one thing that surprised you in the commit message.
- If the step takes > S (<1 day) estimate, stop and split it — open a follow-up note.
