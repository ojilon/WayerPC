# 092 Framing fuzz test (readCommand + uploads)

| Field | Value |
|-------|-------|
| Status | `todo` (change to `doing` / `done` as you work) |
| Priority | P1 |
| Size | S (<1 day) |
| Area | tests |
| Phase | Week 2 — tests |
| Related | `014`, `091` |

## 1. Why this matters

`readCommand` (`server.go:275-298`) faces a hostile network: split segments, `\r\n`, EOF mid-line, 64KB garbage. A Go fuzz test splitting random commands at every offset (with/without `\n`) is cheap, high-value — the parser's contract in executable form.

Read this file, then touch the code. This issue is deliberately scoped so you can
finish it in one sitting and see the result in the running app.

## 2. Current behavior

No fuzz tests; `server_test.go` has fixed cases (verify). `maxLine` bound exists but fuzz-unproven. Upload `recvExact` length handling (`transfer.go:63-96`) similarly unfuzzed (short/long/overlong bodies).

Key files:

- `app.go` (Wails bridge) + `internal/` packages — see issue body for exact paths.

## 3. Problems / symptoms

- Edge inputs (CR-only, NUL bytes, 64KB+1) behavior unknown.
- Refactors of framing (`004` newline work) risk regressions no unit test pins.
- Sanitizer (`014`) claims unproven without adversarial inputs.

## 4. Proposal (what "good" looks like)

Go native fuzz (`testing.F`): `FuzzReadCommand` (arbitrary bytes → command or clean error, never panic/hang/OOM) + `FuzzSanitizeFilename` (from `014`) + seed corpus from `091` cases. Run 30s locally; CI runs seed corpus (not full fuzz) + optional nightly 5-min.

## 5. Step-by-step implementation

1. Extract framing core for testability if needed (no behavior change).
2. Write `FuzzReadCommand`: invariants = terminates, output ≤ maxLine, valid trims only trailing CR/space/tab.
3. Seed corpus: `091` headers (newline, newline-less, CRLF, split, empty, 64KB).
4. `FuzzSanitizeFilename`: never escapes containment (assert within-base or reject).
5. Run `go test -fuzz` 30s; commit corpus; document nightly-longer command.

Work in small commits. After each step, run the relevant check:

```powershell
go build ./...; if ($?) { go test ./... }
cmd /c "pnpm.cmd --dir frontend build"
```

## 6. Acceptance criteria

- [ ] 30s fuzz clean (no panic/hang/OOM) on this machine.
- [ ] Seed corpus committed and CI-green.
- [ ] Sanitizer fuzz asserts containment (not just no-crash).

## 7. Tests to add / update

The fuzz targets + corpus. Nightly command in BUILD.md.

## 8. Risks and rollback

Fuzz-found crashes must be fixed or triaged immediately — don't commit a fuzzer that fails; fix-then-commit.

Rollback: `git stash` or revert the single commit for this issue. Nothing here
touches the wire protocol unless the header says *(breaking)* — and none of the
early-phase issues do.

## 9. Follow-ups (do NOT do in this issue)

`014` (uses the sanitizer fuzz), `079` (audit cites coverage).

## 10. Touch-and-learn notes

- Read the linked files first, top to bottom, before editing.
- Note one thing that surprised you in the commit message.
- If the step takes > S (<1 day) estimate, stop and split it — open a follow-up note.
