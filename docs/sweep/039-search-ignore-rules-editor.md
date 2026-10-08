# 039 Search ignore-rules editor

| Field | Value |
|-------|-------|
| Status | `todo` (change to `doing` / `done` as you work) |
| Priority | P2 |
| Size | M (1-3 days) |
| Area | search/frontend |
| Phase | Week 4 — search depth |
| Related | `040` |

## 1. Why this matters

`skipSegments` (`search.go:24-37`) is a hardcoded block-list: windows, program files, appdata, wayerpc, proc… Users with `D:\Windows-ISOs` or dev folders can't tune it; `my-windows-themes` is correctly kept (segment match) but nobody can tell that from the UI.

Read this file, then touch the code. This issue is deliberately scoped so you can
finish it in one sitting and see the result in the running app.

## 2. Current behavior

Map literal, case-insensitive segment match (`SkipRel`, `search.go:40-50`). No config, no UI, no per-drive rules. Tests cover segments (`search_test.go`) but users can't see them.

Key files:

- `app.go` (Wails bridge) + `internal/` packages — see issue body for exact paths.

## 3. Problems / symptoms

- False skips with no recourse (e.g. folder literally named `dev` on an external drive).
- No way to add personal skips (`node_modules`, `.git`, `__pycache__`).
- Block-list drift: code change needed for every new rule.

## 4. Proposal (what "good" looks like)

Persist custom rules (extra skip segments + allow-overrides) in settings kv; Import view gets 'Search filters…' collapsible showing effective rules with add/remove; `SkipRel` consults defaults + custom. Ship with sensible dev-folder defaults added (`node_modules`, `.git`, `.venv`, `__pycache__`).

## 5. Step-by-step implementation

1. Settings kv keys `search.skip_extra`, `search.allow` (JSON arrays).
2. `SkipRel` → `SkipRelWith(custom)`; keep old func as default wrapper.
3. Import view filter editor (list + add/remove + reset-to-defaults).
4. Add dev-folder defaults to base map (measure walk-speed win).
5. Tests: custom skip honored; allow-override wins over default.

Work in small commits. After each step, run the relevant check:

```powershell
go build ./...; if ($?) { go test ./... }
cmd /c "pnpm.cmd --dir frontend build"
```

## 6. Acceptance criteria

- [ ] User can exclude `node_modules` and see faster searches.
- [ ] `D:\dev\photo.jpg` findable after allow-override (documents the tradeoff).
- [ ] Rules persist across restarts.

## 7. Tests to add / update

Rule-precedence unit tests; persistence round-trip.

## 8. Risks and rollback

Over-skipping hides user files — UI must show effective rule count and offer reset. Never skip user home root itself (`036` roots logic).

Rollback: `git stash` or revert the single commit for this issue. Nothing here
touches the wire protocol unless the header says *(breaking)* — and none of the
early-phase issues do.

## 9. Follow-ups (do NOT do in this issue)

`040` (symlink safety composes with skips).

## 10. Touch-and-learn notes

- Read the linked files first, top to bottom, before editing.
- Note one thing that surprised you in the commit message.
- If the step takes > M (1-3 days) estimate, stop and split it — open a follow-up note.
