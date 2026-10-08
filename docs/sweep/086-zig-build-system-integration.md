# 086 Zig plan 6/10: build system integration

| Field | Value |
|-------|-------|
| Status | `todo` (change to `doing` / `done` as you work) |
| Priority | P1 |
| Size | M (1-3 days) |
| Area | zig/build |
| Phase | Week 7 — Zig track |
| Related | `085`, `087`, `094` |

## 1. Why this matters

A Zig port that needs a wiki page to compile is dead. `BUILD.md` + `scripts/` + `wails build` must stay one-command for the default path, with Zig steps additive and clearly optional.

Read this file, then touch the code. This issue is deliberately scoped so you can
finish it in one sitting and see the result in the running app.

## 2. Current behavior

`BUILD.md` documents pnpm + wails flows; `scripts/sync-version.mjs`, `build-installer.ps1`; `wails.json` frontend wiring. No Zig toolchain references. `.data/` stub keeps clones buildable offline.

Key files:

- `app.go` (Wails bridge) + `internal/` packages — see issue body for exact paths.

## 3. Problems / symptoms

- `zig` version drift (0.13 vs 0.14 API breaks) irreproducible builds.
- Wails `frontend:build` hook unaware of native libs (where do .a/.lib land?).
- CI (`094`) must test BOTH paths (pure-Go always; Zig when toolchain present).

## 4. Proposal (what "good" looks like)

Pin `zig` version (`.tool-versions` or `zig-version` file + check script); `scripts/build-zig.sh/ps1` compiles libs into `build/zig/` (gitignored); Go picks them via tags (see `085`); default `BUILD.md` flow unchanged (pure-Go, no zig needed); `BUILD_ZIG.md` (or section) documents the opt-in second flow. CI: pure-Go gate required, Zig build advisory until stable.

## 5. Step-by-step implementation

1. Pin version + `scripts/check-zig.(ps1|sh)` (prints version, advisory only).
2. `build.zig` per lib (`082` scorer first) emitting static lib to `build/zig/`.
3. Go link via `zdep` tag files (cgo LDFLAGS to build dir; relative, no abs paths).
4. BUILD.md: default flow untouched + 'optional Zig acceleration' section.
5. CI sketch (`094`): matrix job `zig=true` advisory.
6. Verify clean-clone pure-Go build with NO zig installed (the critical test).

Work in small commits. After each step, run the relevant check:

```powershell
go build ./...; if ($?) { go test ./... }
cmd /c "pnpm.cmd --dir frontend build"
```

## 6. Acceptance criteria

- [ ] No-zig clone builds + tests green (proves opt-in).
- [ ] With-zig build produces + links the scorer lib.
- [ ] Pinned version documented; mismatch warns, doesn't silently build.

## 7. Tests to add / update

CI matrix (when `094` lands); local clean-env check (rename zig binary, rebuild).

## 8. Risks and rollback

Zig breaking changes per release — pin + upgrade deliberately, note in changelog (`099`).

Rollback: `git stash` or revert the single commit for this issue. Nothing here
touches the wire protocol unless the header says *(breaking)* — and none of the
early-phase issues do.

## 9. Follow-ups (do NOT do in this issue)

`087` (Windows specifics), `094` (CI wires it).

## 10. Touch-and-learn notes

- Read the linked files first, top to bottom, before editing.
- Note one thing that surprised you in the commit message.
- If the step takes > M (1-3 days) estimate, stop and split it — open a follow-up note.
