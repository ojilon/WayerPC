# 013 Error taxonomy cleanup

| Field | Value |
|-------|-------|
| Status | `todo` (change to `doing` / `done` as you work) |
| Priority | P1 |
| Size | S (<1 day) |
| Area | protocol |
| Phase | Week 2 — protocol correctness |
| Related | `006`, `091` |

## 1. Why this matters

`ERROR <token>` strings grew organically: `file_not_found`, `file_access_denied`, `invalid_command`, `unknown_protocol_command.`, `upload_too_large`, `write_failed`, `upload_incomplete`, plus a free-text variant. The trailing period on one, `ERROR:` prefix on another — clients must fuzzy-match.

Read this file, then touch the code. This issue is deliberately scoped so you can
finish it in one sitting and see the result in the running app.

## 2. Current behavior

Scattered across `protocol.go:23,31,41,79,89,96,114-117,120-122,128,137-141,154-167,181,195`. Status codes (`StatusAskOK=0…`) are logged server-side but never sent. No doc table.

Key files:

- `app.go` (Wails bridge) + `internal/` packages — see issue body for exact paths.

## 3. Problems / symptoms

- Token spellings drifted: `unknown_protocol_command.` has a trailing period, `ERROR: Failed to obtain directory…` uses a colon-prefix plus sentence, the rest are `ERROR snake_case`.
- `Status*` codes (0/3/4/1/2/−1) are log-only, never on the wire — two taxonomies, neither documented.
- Clients must substring-match replies; a new token that almost matches an old one misclassifies silently.

## 4. Proposal (what "good" looks like)

Freeze existing tokens (compat) and publish the canonical table: token → meaning → client action → hello-proto version. Normalize new code to `ERROR snake_case` with no trailing period. Add `ERROR auth_required` + `ERROR busy` reservations for `010`/`011`.

## 5. Step-by-step implementation

1. Inventory every `sendLine(conn, "ERROR…")` into a table in code comment + README.
2. Fix the two anomalies for NEW paths only (`unknown_protocol_command.` period, `ERROR: Failed…` prefix) — keep old tokens accepted.
3. Add client-action column (retry / pick-from-MATCHES / abort).
4. Conformance test asserts the table (no undocumented ERROR).
5. Document in README Protocol section.

Work in small commits. After each step, run the relevant check:

```powershell
go build ./...; if ($?) { go test ./... }
cmd /c "pnpm.cmd --dir frontend build"
```

## 6. Acceptance criteria

- [ ] Table covers 100% of ERROR sends; CI fails on new undocumented token.
- [ ] Zero behavior change for existing clients.
- [ ] README table merged.

## 7. Tests to add / update

Grep-test in `091` script: capture ERRORs, diff against table.

## 8. Risks and rollback

Temptation to 'fix' old tokens — don't. Freeze + document, normalize only new ones.

Rollback: `git stash` or revert the single commit for this issue. Nothing here
touches the wire protocol unless the header says *(breaking)* — and none of the
early-phase issues do.

## 9. Follow-ups (do NOT do in this issue)

`006` (hello advertises taxonomy version), Android error-string mapping.

## 10. Touch-and-learn notes

- Read the linked files first, top to bottom, before editing.
- Note one thing that surprised you in the commit message.
- If the step takes > S (<1 day) estimate, stop and split it — open a follow-up note.
