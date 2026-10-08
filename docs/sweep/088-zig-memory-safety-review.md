# 088 Zig plan 8/10: memory-safety review for Zig code

| Field | Value |
|-------|-------|
| Status | `todo` (change to `doing` / `done` as you work) |
| Priority | P1 |
| Size | S (<1 day) |
| Area | zig/security |
| Phase | Week 8 — Zig track |
| Related | `085`, `079` |

## 1. Why this matters

Zig is non-GC and explicit: the safety Go gives for free (bounds checks, no UAF) becomes YOUR checklist. Before any Zig ships (even behind a flag), review the ports for OOB, UAF, leaks, uninit reads, and hostile-input behavior (a 64KB command reaching scorer input?).

Read this file, then touch the code. This issue is deliberately scoped so you can
finish it in one sitting and see the result in the running app.

## 2. Current behavior

Go code is memory-safe by construction (remaining risks are logic: traversal `079`, not memory). Zig ports handle untrusted strings (filenames, queries) across an FFI boundary — the exact place to get this wrong.

Key files:

- `app.go` (Wails bridge) + `internal/` packages — see issue body for exact paths.

## 3. Problems / symptoms

- FFI strings (ptr+len) invite OOB on length confusion (bytes vs codepoints).
- Allocator misuse (arena vs page vs fixed) leaks across calls.
- `undefined` reads in Zig are silent until they aren't.

## 4. Proposal (what "good" looks like)

Checklist review per port: bounds on all indexing (fuzz `092`-style with Zig corpus), ownership per `085` contract, allocator choice justified + leak-tested (`zig build test` with GPA leak detection in Debug), hostile inputs (empty, 64KB, invalid UTF-8, NUL bytes) defined-behavior (error, never crash). Record the review as a note per port.

## 5. Step-by-step implementation

1. Enable GPA + leak-check in Zig tests (fail on leak).
2. Hostile-input battery per port (empty/NUL/huge/invalid-UTF8) — assert error-or-sane, never crash.
3. Fuzz scorer input (byte-mutations vs Go oracle — reuse `082` goldens as seed).
4. Review ownership annotations against `085` contract (two-reader sign-off: you + AI).
5. File the review note (date, scope, findings) next to the port.

Work in small commits. After each step, run the relevant check:

```powershell
go build ./...; if ($?) { go test ./... }
cmd /c "pnpm.cmd --dir frontend build"
```

## 6. Acceptance criteria

- [ ] GPA leak-clean on port test suites.
- [ ] Hostile battery green (defined behavior documented per case).
- [ ] Review note merged per ported lib.

## 7. Tests to add / update

Zig-side tests (GPA) + Go-side hostile-input tests through the boundary.

## 8. Risks and rollback

False confidence from 'it passed tests' — fuzz briefly, document limits, keep flag-off default (`090`).

Rollback: `git stash` or revert the single commit for this issue. Nothing here
touches the wire protocol unless the header says *(breaking)* — and none of the
early-phase issues do.

## 9. Follow-ups (do NOT do in this issue)

`079` (logic audit cites this for memory), `090` (flag stays until review done).

## 10. Touch-and-learn notes

- Read the linked files first, top to bottom, before editing.
- Note one thing that surprised you in the commit message.
- If the step takes > S (<1 day) estimate, stop and split it — open a follow-up note.
