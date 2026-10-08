# 051 Reveal-in-Explorer fallbacks

| Field | Value |
|-------|-------|
| Status | `todo` (change to `doing` / `done` as you work) |
| Priority | P2 |
| Size | S (<1 day) |
| Area | storage/frontend |
| Phase | Week 4 — frontend depth |
| Related | `049` |

## 1. Why this matters

Reveal (`storage.go:216-230`, `app.go:486-491`) shell-outs to explorer/open/xdg-open and returns 'ok' or raw err. On missing files, Linux without xdg-open, or Snap confinement it fails silently or with raw exec text.

Read this file, then touch the code. This issue is deliberately scoped so you can
finish it in one sitting and see the result in the running app.

## 2. Current behavior

`exec.Command(...).Start()` per OS; no existence pre-check; `library.ts:79,109` toasts only non-ok. Missing-on-disk rows (`032`) offer Reveal that can't work. No 'open parent folder' fallback.

Key files:

- `app.go` (Wails bridge) + `internal/` packages — see issue body for exact paths.

## 3. Problems / symptoms

- Dead button on missing files (ghost rows).
- Linux error text leaks exec details.
- No fallback: reveal fails → nothing (should open parent or show path).

## 4. Proposal (what "good" looks like)

Pre-check existence; disable Reveal on missing rows (badge already says missing); on exec failure fall back to opening the parent dir, then to copying the path to clipboard with guidance. Map errors via `049` humanizer.

## 5. Step-by-step implementation

1. `RevealInExplorer`: stat first → typed `ErrMissing` (UI disables button instead of calling).
2. Fallback chain: select → open-parent → clipboard+toast.
3. `library.ts`: disable Reveal when `exists===false`; show parent-open for missing.
4. Test: missing path → typed error, no process spawn.
5. Manual matrix: Windows file/folder, Linux xdg present/absent.

Work in small commits. After each step, run the relevant check:

```powershell
go build ./...; if ($?) { go test ./... }
cmd /c "pnpm.cmd --dir frontend build"
```

## 6. Acceptance criteria

- [ ] Missing rows have no dead Reveal button.
- [ ] Broken-reveal still lands the user in the parent folder.
- [ ] No raw exec text reaches a toast.

## 7. Tests to add / update

Missing-path unit test; fallback test with fake exe in PATH.

## 8. Risks and rollback

Clipboard fallback needs Wails runtime access — degrade to 'path shown' if unavailable.

Rollback: `git stash` or revert the single commit for this issue. Nothing here
touches the wire protocol unless the header says *(breaking)* — and none of the
early-phase issues do.

## 9. Follow-ups (do NOT do in this issue)

`032` (missing UX owns the disabled state).

## 10. Touch-and-learn notes

- Read the linked files first, top to bottom, before editing.
- Note one thing that surprised you in the commit message.
- If the step takes > S (<1 day) estimate, stop and split it — open a follow-up note.
