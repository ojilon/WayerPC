# 087 Zig plan 7/10: Windows cross-compile + long-path story

| Field | Value |
|-------|-------|
| Status | `todo` (change to `doing` / `done` as you work) |
| Priority | P2 |
| Size | S (<1 day) |
| Area | zig/build |
| Phase | Week 7 — Zig track |
| Related | `086`, `083` |

## 1. Why this matters

Primary target is `windows/amd64` (`wails build -platform windows/amd64`, installer NSIS). Zig's `zig cc`/`zig build -Dtarget=x86_64-windows` story must be proven for scorer+walker, including Windows path semantics (separators, `\\?\` long paths, junctions) — Linux-green means nothing here.

Read this file, then touch the code. This issue is deliberately scoped so you can
finish it in one sitting and see the result in the running app.

## 2. Current behavior

Dev on Windows (this workspace); releases Windows-only for now. Walker semantics Windows-specific (`083` risk). No cross-compile needs yet (build on Windows), but reproducibility + future CI-on-Linux demand the target story.

Key files:

- `app.go` (Wails bridge) + `internal/` packages — see issue body for exact paths.

## 3. Problems / symptoms

- Long paths (>260) behave differently per API (manifest longPathAware? `\\?\` prefixes?).
- Junction/reparse handling in Zig std on Windows unverified.
- MSVC vs gnu ABI for the static lib linkage with cgo/gcc.

## 4. Proposal (what "good" looks like)

Prove: build Zig libs with `-Dtarget=x86_64-windows(-gnu?)` from this machine AND (stretch) from Linux CI; run parity suites on Windows (goldens from `082`/`083`); document the chosen target triple + libc (gnu vs msvc) + long-path test results (300-char fixture).

## 5. Step-by-step implementation

1. Fix target triple(s) in `build.zig` (recommend `x86_64-windows-gnu` if cgo-gcc links gnu; verify). 
2. Long-path fixture test (deep tree >260 chars) through walker parity (`083`).
3. Junction fixture (mklink /J) behavior recorded.
4. CI note: can Linux runners produce the Windows lib? (try once, record).
5. Document final triple + flags in BUILD_ZIG section.

Work in small commits. After each step, run the relevant check:

```powershell
go build ./...; if ($?) { go test ./... }
cmd /c "pnpm.cmd --dir frontend build"
```

## 6. Acceptance criteria

- [ ] Windows parity suites green on cross-built lib.
- [ ] Long-path + junction behaviors documented (pass or known-limit).
- [ ] Target triple pinned (no 'works on my machine' flags).

## 7. Tests to add / update

Windows-only fixture tests (skip gracefully elsewhere with message).

## 8. Risks and rollback

gnu-vs-msvc linkage rabbit hole — timebox 1 day; if stuck, native-Windows-zig-build is the documented path, cross-compile deferred.

Rollback: `git stash` or revert the single commit for this issue. Nothing here
touches the wire protocol unless the header says *(breaking)* — and none of the
early-phase issues do.

## 9. Follow-ups (do NOT do in this issue)

`083` (walker needs this verdict), `094` (CI target job).

## 10. Touch-and-learn notes

- Read the linked files first, top to bottom, before editing.
- Note one thing that surprised you in the commit message.
- If the step takes > S (<1 day) estimate, stop and split it — open a follow-up note.
