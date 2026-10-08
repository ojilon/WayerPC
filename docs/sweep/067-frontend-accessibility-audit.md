# 067 Accessibility audit (contrast, focus, labels, motion)

| Field | Value |
|-------|-------|
| Status | `todo` (change to `doing` / `done` as you work) |
| Priority | P2 |
| Size | M (1-3 days) |
| Area | frontend |
| Phase | Week 5 — frontend depth |
| Related | `060`, `062` |

## 1. Why this matters

Frameless custom chrome (titlebar drag, icon-ish `_/▢/✕` buttons `main.ts:51-55`) + toast-only feedback + no aria review = keyboard/screen-reader users struggle. Do a bounded audit, not a rewrite.

Read this file, then touch the code. This issue is deliberately scoped so you can
finish it in one sitting and see the result in the running app.

## 2. Current behavior

No aria roles beyond `role=tablist` on segments (`library.ts:20`); log has `aria-live=polite` (good, `console.ts:40`); buttons mostly text (good); focus styles unaudited; no reduced-motion respect; no skip-link.

Key files:

- `app.go` (Wails bridge) + `internal/` packages — see issue body for exact paths.

## 3. Problems / symptoms

- Unknown screen-reader story for log + results lists.
- Focus rings possibly invisible on dark tokens.
- Toast-only errors (`057`) invisible to SR without live regions.

## 4. Proposal (what "good" looks like)

Bounded pass, not a rewrite: label nav/segments/results (roles + `aria-selected` + result-count announcements), visible `:focus-visible` rings in both themes (`060`), `prefers-reduced-motion` disables toast/log animation, toast errors get an assertive live region in the drawer (`057`) while info stays polite — and whatever remains is written down as explicit follow-ups, never silently "done".

## 5. Step-by-step implementation

1. Keyboard-only run script (from `062`) extended with SR spot-check (Narrator/NVDA free). 
2. Add missing labels/roles (nav, seg tabs with aria-selected, results count announcements).
3. `:focus-visible` + contrast fixes for both themes.
4. `prefers-reduced-motion` media query kills animations.
5. Toast errors → assertive live region in drawer; info stays polite.
6. Record remaining gaps explicitly (no silent 'done').

Work in small commits. After each step, run the relevant check:

```powershell
go build ./...; if ($?) { go test ./... }
cmd /c "pnpm.cmd --dir frontend build"
```

## 6. Acceptance criteria

- [ ] Keyboard-only full-flow passes (import→files→settings).
- [ ] No information conveyed by color alone (badges get text, not just red).
- [ ] Reduced-motion honored; audit notes filed as follow-ups.

## 7. Tests to add / update

Axe-core spot run in browser preview (optional but nice); manual SR script recorded.

## 8. Risks and rollback

Retrofitting full ARIA perfectly is a rabbit hole — bound it: labels+focus+motion now, deeper SR work later.

Rollback: `git stash` or revert the single commit for this issue. Nothing here
touches the wire protocol unless the header says *(breaking)* — and none of the
early-phase issues do.

## 9. Follow-ups (do NOT do in this issue)

Full SR pass (future); `060` theme contrast tokens revisited with numbers.

## 10. Touch-and-learn notes

- Read the linked files first, top to bottom, before editing.
- Note one thing that surprised you in the commit message.
- If the step takes > M (1-3 days) estimate, stop and split it — open a follow-up note.
