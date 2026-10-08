# 060 Light theme toggle (respect OS theme)

| Field | Value |
|-------|-------|
| Status | `todo` (change to `doing` / `done` as you work) |
| Priority | P2 |
| Size | S (<1 day) |
| Area | frontend |
| Phase | Week 4 — frontend depth |
| Related | `067` |

## 1. Why this matters

Dark-only (`main.go:56` forces `windows.Dark`; `styles.css` dark tokens). Daytime hotspot field use + accessibility (`067`) want a light option; CSS variables already isolate the palette (verify).

Read this file, then touch the code. This issue is deliberately scoped so you can
finish it in one sitting and see the result in the running app.

## 2. Current behavior

No toggle; no `prefers-color-scheme` query; no persisted theme. `BackgroundColour` (15,20,25) + `windows.Dark` hardcode dark chrome. Titlebar/sidebar palette in `styles.css`.

Key files:

- `app.go` (Wails bridge) + `internal/` packages — see issue body for exact paths.

## 3. Problems / symptoms

- Bright-room illegibility for some users.
- OS-theme users get a mismatched app.
- Hardcoded WebView chrome vs CSS could disagree (titlebar).

## 4. Proposal (what "good" looks like)

Add `data-theme=dark|light` on root + light token set; toggle in Settings (System/Dark/Light; System = media query, default Dark to preserve current look); persist in settings kv; update Wails window theme where the API allows at runtime (else document restart-only for chrome).

## 5. Step-by-step implementation

1. Audit `styles.css` variables; extract light set (contrast-check grays + accent).
2. Toggle UI in Settings + `matchMedia('(prefers-color-scheme: light)')` listener for System.
3. Persist choice; apply before first paint (avoid flash).
4. Manual contrast pass (text 4.5:1) + focus rings visible in both (`067`).
5. Screenshot both for the release notes (`099`).

Work in small commits. After each step, run the relevant check:

```powershell
go build ./...; if ($?) { go test ./... }
cmd /c "pnpm.cmd --dir frontend build"
```

## 6. Acceptance criteria

- [ ] Toggle switches instantly, persists across restarts.
- [ ] System mode follows OS change live (or documents restart if API-bound).
- [ ] No unreadable text in either theme (spot-check all 4 views).

## 7. Tests to add / update

Manual visual matrix; token-presence test (both themes define every var).

## 8. Risks and rollback

Third-party/WebView chrome theming limits on Windows — document what can't change at runtime.

Rollback: `git stash` or revert the single commit for this issue. Nothing here
touches the wire protocol unless the header says *(breaking)* — and none of the
early-phase issues do.

## 9. Follow-ups (do NOT do in this issue)

`067` (focus/contrast audit covers both themes).

## 10. Touch-and-learn notes

- Read the linked files first, top to bottom, before editing.
- Note one thing that surprised you in the commit message.
- If the step takes > S (<1 day) estimate, stop and split it — open a follow-up note.
