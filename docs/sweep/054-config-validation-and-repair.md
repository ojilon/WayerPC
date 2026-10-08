# 054 Config validation + repair on startup

| Field | Value |
|-------|-------|
| Status | `todo` (change to `doing` / `done` as you work) |
| Priority | P1 |
| Size | S (<1 day) |
| Area | config |
| Phase | Week 3 — backend reliability |
| Related | `052`, `049` |

## 1. Why this matters

`Load()` (`config.go:85-116`) is lenient by design (missing/corrupt → defaults, nil error). Lenient + silent = users with a corrupt `config.json` or a deleted data-root get onboarding emptiness with no explanation. Validate, report, offer repair.

Read this file, then touch the code. This issue is deliberately scoped so you can
finish it in one sitting and see the result in the running app.

## 2. Current behavior

Corrupt JSON → defaults silently; unknown keys ignored; `DataRoot` pointing at a deleted folder flows to `openRoot` which `MkdirAll`s it back (recreating a deleted tree silently!) or fails raw. No 'config problem' state exists.

Key files:

- `app.go` (Wails bridge) + `internal/` packages — see issue body for exact paths.

## 3. Problems / symptoms

- Silent recreation of a folder the user deliberately deleted.
- Half-written config (crash during Save — non-atomic `WriteFile`) → silent defaults.
- No backup of last-good config.

## 4. Proposal (what "good" looks like)

Validate on load: JSON parse errors, unknown version keys, missing data-root, unwritable root → structured `ConfigIssue[]` surfaced in Settings banner with one-click repairs (re-create / pick new / restore backup). Make `Save` atomic (write tmp + rename) + keep `config.json.bak`.

## 5. Step-by-step implementation

1. `Load` returns `(Config, []Issue)` (keep err nil for compat; issues drive UI).
2. Checks: parseable, port range, root exists+writable (probe, `049`).
3. Atomic save: tmp+rename; keep one `.bak`.
4. Settings banner lists issues + repair buttons (reuse `052` fields + data-root picker).
5. Test: corrupt JSON → banner, not silence; deleted root → 're-create or pick' prompt.

Work in small commits. After each step, run the relevant check:

```powershell
go build ./...; if ($?) { go test ./... }
cmd /c "pnpm.cmd --dir frontend build"
```

## 6. Acceptance criteria

- [ ] Corrupt config produces a named banner with repairs (not silent defaults).
- [ ] Crash during save can't leave half-written config (atomicity test by fault injection or code review + test).
- [ ] Last-good `.bak` exists after every save.

## 7. Tests to add / update

Corrupt/missing-root test matrix; atomic-save test (write failure leaves original intact).

## 8. Risks and rollback

Over-prompting on first run (no config = normal, not an issue) — distinguish 'fresh' from 'broken'.

Rollback: `git stash` or revert the single commit for this issue. Nothing here
touches the wire protocol unless the header says *(breaking)* — and none of the
early-phase issues do.

## 9. Follow-ups (do NOT do in this issue)

`052` (repair UI fields), `031` (backup philosophy shared).

## 10. Touch-and-learn notes

- Read the linked files first, top to bottom, before editing.
- Note one thing that surprised you in the commit message.
- If the step takes > S (<1 day) estimate, stop and split it — open a follow-up note.
