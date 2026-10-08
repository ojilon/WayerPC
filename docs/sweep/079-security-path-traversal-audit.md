# 079 Path-traversal + input audit (explicit pass)

| Field | Value |
|-------|-------|
| Status | `todo` (change to `doing` / `done` as you work) |
| Priority | P1 |
| Size | M (1-3 days) |
| Area | security |
| Phase | Week 5 — hardening |
| Related | `014`, `080` |

## 1. Why this matters

A hotspot-local server that writes files (`received/`), reads arbitrary cataloged paths, and shells to Explorer is exactly where traversal/expansion bugs bite. `014` fixes filenames; this audits everything else: `..`, symlinks, absolute paths, `.part` collisions, dialog-path trust.

Read this file, then touch the code. This issue is deliberately scoped so you can
finish it in one sitting and see the result in the running app.

## 2. Current behavior

`filepath.Base` on upload names (`protocol.go:126`) + `Abs` on catalog ops + `RevealInExplorer` exec with raw path + `SetDataRoot`/`PickFolder` trusting dialog output. Each is probably fine; none has a proof (test) next to it.

Key files:

- `app.go` (Wails bridge) + `internal/` packages — see issue body for exact paths.

## 3. Problems / symptoms

- No single audit artifact: 'we checked X, Y, Z on <date>'.
- `.part` + final-path TOCTOU (symlink swap between stat and rename?) unexamined.
- Dialog paths (attacker can't control, but a malicious shortcut could?) unvalidated.

## 4. Proposal (what "good" looks like)

Bounded audit, documented: enumerate every path input (wire, dialog, config, CLI `053`), assert containment (final paths must be within received//dataRoot as appropriate; catalog READS may be anywhere by design — document why that's OK + the auth story `010`/`080`), add regression tests per finding, file follow-ups for anything bigger.

## 5. Step-by-step implementation

1. List all path inputs → sinks (table in the commit/file).
2. Add containment asserts: upload final must be within received/ (Base already ensures; prove with test incl. `..`, abs, `~`).
3. `Reveal/Open` (`051`/`058`): refuse missing + log odd paths (NUL bytes, overlong).
4. `SetDataRoot`: refuse nesting inside received//shared? of another root + root-of-drive without confirm.
5. Write the audit note (date, scope, findings, wontfix-with-reason).
6. Re-run `092` fuzz corpus against the audit cases.

Work in small commits. After each step, run the relevant check:

```powershell
go build ./...; if ($?) { go test ./... }
cmd /c "pnpm.cmd --dir frontend build"
```

## 6. Acceptance criteria

- [ ] Audit table merged (inputs × sinks × verdict).
- [ ] Traversal battery green (`..`, abs, UNC `\\?\`, trailing dots).
- [ ] Catalog-read-anywhere design decision written down (not just assumed).

## 7. Tests to add / update

Traversal battery (table-driven, per sink).

## 8. Risks and rollback

Audit without exploitability framing causes alarm — verdicts must say 'exploitable? no, because…' or file the fix.

Rollback: `git stash` or revert the single commit for this issue. Nothing here
touches the wire protocol unless the header says *(breaking)* — and none of the
early-phase issues do.

## 9. Follow-ups (do NOT do in this issue)

`080` (trust model cites this), `010` (auth reduces impact of any residual).

## 10. Touch-and-learn notes

- Read the linked files first, top to bottom, before editing.
- Note one thing that surprised you in the commit message.
- If the step takes > M (1-3 days) estimate, stop and split it — open a follow-up note.
