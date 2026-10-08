# 085 Zig plan 5/10: C-ABI + cgo integration pattern

| Field | Value |
|-------|-------|
| Status | `todo` (change to `doing` / `done` as you work) |
| Priority | P1 |
| Size | M (1-3 days) |
| Area | zig |
| Phase | Week 7 — Zig track |
| Related | `081`, `086`, `090` |

## 1. Why this matters

All Zig ports (`082-084`) need ONE clean Go↔Zig boundary. Getting the ABI/memory-ownership pattern right once (who allocates, who frees, string passing, error returns, no Go pointers into Zig-held memory) prevents three different ad-hoc bridges.

Read this file, then touch the code. This issue is deliberately scoped so you can
finish it in one sitting and see the result in the running app.

## 2. Current behavior

No cgo, no C code in build (legacy DLLs gone). `go.mod` pure-Go (+Wails). Adding cgo changes build requirements (gcc on Windows) — must be opt-in via build tags (`090`), never default.

Key files:

- `app.go` (Wails bridge) + `internal/` packages — see issue body for exact paths.

## 3. Problems / symptoms

- cgo pointer rules (no Go memory with Go pointers passed to C and kept) — violations crash subtly.
- String passing (ptr+len, UTF-8) + error signaling (code + msg buffer) need conventions.
- Memory ownership confusion (who frees batch buffers?) leaks or double-frees.

## 4. Proposal (what "good" looks like)

One documented pattern: Zig exports `*_create/_destroy` handles + `*_process(in_ptr,in_len,out_ptr*,out_len*)->int32 code`; all buffers Zig-allocated, Go copies out then calls `*_free`; errors as codes + optional msg fill; zero Go-pointer passing (copy strings to C memory first). Provide `internal/zdep` package with tag-gated `//go:cgo` wiring + pure-Go fallback stubs (same API, flag-off = Go impl).

## 5. Step-by-step implementation

1. Write the ABI contract doc (1 page: types, ownership, errors, threading: Zig fns must be thread-safe or documented not). 
2. Implement `internal/zdep` skeleton with `//go:build zig` tag + fallback file (same funcs, Go impls).
3. Spike: `score` through the real boundary (from `082`) with -race + leak check.
4. Document cgo toolchain needs (Windows gcc via zig-cc? or mingw — decide in `086`).
5. Review checklist for future ports (copy-paste gate).

Work in small commits. After each step, run the relevant check:

```powershell
go build ./...; if ($?) { go test ./... }
cmd /c "pnpm.cmd --dir frontend build"
```

## 6. Acceptance criteria

- [ ] Contract doc merged; all ports reference it (no bespoke bridges).
- [ ] Tag-off build needs no C toolchain (CI proves: build without gcc).
- [ ] -race + repeated-call leak test green.

## 7. Tests to add / update

ABI round-trip + ownership tests (free-after-copy, double-free guard, error-path msg).

## 8. Risks and rollback

cgo on Windows is the friction — keep default builds cgo-free; Zig path is opt-in only.

Rollback: `git stash` or revert the single commit for this issue. Nothing here
touches the wire protocol unless the header says *(breaking)* — and none of the
early-phase issues do.

## 9. Follow-ups (do NOT do in this issue)

`086` (toolchain), `090` (flag gating uses this package).

## 10. Touch-and-learn notes

- Read the linked files first, top to bottom, before editing.
- Note one thing that surprised you in the commit message.
- If the step takes > M (1-3 days) estimate, stop and split it — open a follow-up note.
