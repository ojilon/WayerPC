# 014 Command limits and filename sanitization

| Field | Value |
|-------|-------|
| Status | `todo` (change to `doing` / `done` as you work) |
| Priority | P1 |
| Size | S (<1 day) |
| Area | protocol/security |
| Phase | Week 2 — protocol correctness |
| Related | `079`, `092` |

## 1. Why this matters

`maxLine = 64*1024` (`server.go:54`) with no per-verb cap; filenames take `filepath.Base` (`protocol.go:126`) which is good, but `..`, absolute paths, reserved Windows names (`CON`, `NUL`), and trailing-dot behaviors deserve explicit tests.

Read this file, then touch the code. This issue is deliberately scoped so you can
finish it in one sitting and see the result in the running app.

## 2. Current behavior

`filepath.Base` strips directories — solid. But overlong queries hit the catalog linear scan (`catalog.go:FindRelated`) with attacker-controlled cost; empty/`.`/`/` names are checked (`protocol.go:127`) while `...`, `CON`, `foo.` are not. No separator for max query length vs max filename length.

Key files:

- `app.go` (Wails bridge) + `internal/` packages — see issue body for exact paths.

## 3. Problems / symptoms

- 64 KiB `maxLine` with no per-verb cap: a 64KB `/upload` query runs the full fuzzy scan — attacker-cheap, user-useless.
- `CON`/`NUL`/`AUX`, trailing-dot/space names, and `...` are accepted into `received/` — Windows surprises at open/reveal time.
- `..`/absolute/UNC inputs rely solely on `filepath.Base`; correct but unproven — no test says so.

## 4. Proposal (what "good" looks like)

Document + test the sanitizer: Base + reject list (empty, `.`, `/`, reserved device names, trailing space/dot on Windows). Cap query length (e.g. 512 chars) before scoring. Fuzz it (`092`).

## 5. Step-by-step implementation

1. Extract `sanitizeFilename()` helper from inline `filepath.Base` logic.
2. Add reject list + Windows-reserved names (case-insensitive).
3. Cap search-query length before `FindForRequest`.
4. Table-driven unit tests incl. `../x`, `C:\y`, `CON`, `nul.txt`, 70KB name.
5. Fuzz with `092` harness.

Work in small commits. After each step, run the relevant check:

```powershell
go build ./...; if ($?) { go test ./... }
cmd /c "pnpm.cmd --dir frontend build"
```

## 6. Acceptance criteria

- [ ] Reserved/traversal names rejected with `ERROR invalid_upload_command`.
- [ ] 64KB command doesn't OOM (bounded alloc).
- [ ] Query cap documented.

## 7. Tests to add / update

New `protocol_sanitize_test.go`; fuzz corpus entry.

## 8. Risks and rollback

Over-blocking legit names (e.g. `...and more.txt`) — Base keeps them; only exact reserved names rejected.

Rollback: `git stash` or revert the single commit for this issue. Nothing here
touches the wire protocol unless the header says *(breaking)* — and none of the
early-phase issues do.

## 9. Follow-ups (do NOT do in this issue)

`079` (full audit), `092` (fuzz).

## 10. Touch-and-learn notes

- Read the linked files first, top to bottom, before editing.
- Note one thing that surprised you in the commit message.
- If the step takes > S (<1 day) estimate, stop and split it — open a follow-up note.
