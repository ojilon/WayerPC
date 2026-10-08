# 068 Empty states that teach (all four views)

| Field | Value |
|-------|-------|
| Status | `todo` (change to `doing` / `done` as you work) |
| Priority | P2 |
| Size | S (<1 day) |
| Area | frontend |
| Phase | Week 2 — frontend |
| Related | `063` |

## 1. Why this matters

Empty states today are one-liners ('No results yet.' `import.ts:46`, 'Nothing here yet.' `library.ts:105`, 'Catalog is empty…' `library.ts:75`). Each is a teaching moment wasted: the user with an empty screen most needs the next action.

Read this file, then touch the code. This issue is deliberately scoped so you can
finish it in one sitting and see the result in the running app.

## 2. Current behavior

Plain `.empty` divs, no actions, no illustrations, no next-step links. Onboarding banners (`063`) cover first-run globally but not per-view emptiness later (e.g. pruned catalog, fresh received/).

Key files:

- `app.go` (Wails bridge) + `internal/` packages — see issue body for exact paths.

## 3. Problems / symptoms

- 'Catalog is empty' without 'Go to Import' button.
- 'Nothing here yet' in Received without 'how uploads get here' hint.
- No visual warmth; app feels unfinished when new.

## 4. Proposal (what "good" looks like)

Actionable empties: icon (CSS/unicode, no asset pipeline) + one-line why + primary action button (Import→goes to Import; Received→'how to send from phone' popover reusing `063` copy; Console empty log→'start server' or 'copy logs' `056`). Keep zero new deps.

## 5. Step-by-step implementation

1. Rewrite the 4 empty states with why+action (Import, Received, Catalog, Shared, log-empty edge).
2. Wire actions to existing handlers (view switch, pickers, how-to).
3. Keep strings in i18n catalog if `061` landed (else inline + note to migrate).
4. Screenshot each for release notes.
5. No animation beyond CSS fade (respect `067` reduced-motion).

Work in small commits. After each step, run the relevant check:

```powershell
go build ./...; if ($?) { go test ./... }
cmd /c "pnpm.cmd --dir frontend build"
```

## 6. Acceptance criteria

- [ ] Every empty state has a working next-action button.
- [ ] Zero dead-ends: no empty screen without guidance.
- [ ] Copy consistent with onboarding (`063`).

## 7. Tests to add / update

Manual screenshot matrix; no logic tests (static UI).

## 8. Risks and rollback

Illustration scope creep — unicode/CSS only, no commissioned art, no binary assets.

Rollback: `git stash` or revert the single commit for this issue. Nothing here
touches the wire protocol unless the header says *(breaking)* — and none of the
early-phase issues do.

## 9. Follow-ups (do NOT do in this issue)

Phone-side empty states (Android repo note).

## 10. Touch-and-learn notes

- Read the linked files first, top to bottom, before editing.
- Note one thing that surprised you in the commit message.
- If the step takes > S (<1 day) estimate, stop and split it — open a follow-up note.
