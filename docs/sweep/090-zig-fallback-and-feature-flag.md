# 090 Zig plan 10/10: fallback + feature-flag discipline

| Field | Value |
|-------|-------|
| Status | `todo` (change to `doing` / `done` as you work) |
| Priority | P1 |
| Size | S (<1 day) |
| Area | zig |
| Phase | Week 8 — Zig track |
| Related | `085`, `089` |

## 1. Why this matters

Every Zig port must be reversible at runtime AND build-time: users on weird AV/Windows builds, future Zig regressions, or a `089` stay-Go verdict must never strand the app. Flags + fallbacks are the contract that makes experimenting safe.

Read this file, then touch the code. This issue is deliberately scoped so you can
finish it in one sitting and see the result in the running app.

## 2. Current behavior

`085` `zdep` gives tag-gated build choice; no runtime flag exists yet (settings kv could hold `perf.zig_scorer=on/off`). No fallback telemetry (which path served this `/ask`?), no flag documentation.

Key files:

- `app.go` (Wails bridge) + `internal/` packages — see issue body for exact paths.

## 3. Problems / symptoms

- Build-tag-only = restart + rebuild to compare (support can't ask 'try with Zig off').
- Silent fallback confusion (is my fast `/ask` Zig or Go?).
- A crashing Zig path (see `088`) with no runtime kill-switch bricks the feature.

## 4. Proposal (what "good" looks like)

Two layers: build tag `zig` (compile the Zig path in) + runtime kv flags per port (`perf.zig_scorer`, default OFF until `089` says otherwise, auto-fallback to Go on any Zig error with a warn log). Startup log notes active paths ('scorer: go (zig available, disabled)'). Panic-safe: recover at the boundary, fall back, log.

## 5. Step-by-step implementation

1. Runtime flags in settings kv + `App.GetPerfFlags/SetPerfFlag` bridge.
2. Call sites: try Zig (if enabled+built) → on error/warn fallback Go + `[ZIG-FALLBACK]` log.
3. Startup 'perf paths' log line + Settings Diagnostics row (+ `077` area).
4. Boundary `recover()` (never let Zig-side issues panic the server).
5. Document: how support asks users to toggle; how `089` verdicts flip defaults.
6. Test: flag on/off matrix; forced-Zig-error → Go result + fallback log.

Work in small commits. After each step, run the relevant check:

```powershell
go build ./...; if ($?) { go test ./... }
cmd /c "pnpm.cmd --dir frontend build"
```

## 6. Acceptance criteria

- [ ] Zig path disable-able at runtime without rebuild.
- [ ] Forced failure falls back with a visible log (not silent, not crash).
- [ ] Default-off until a dated `089` verdict flips a specific flag.

## 7. Tests to add / update

Flag-matrix tests; fallback-on-error test (stub Zig impl returning error).

## 8. Risks and rollback

Flag sprawl (3 ports × flags) — one `perf.*` namespace + one Diagnostics row, no per-port UI beyond that.

Rollback: `git stash` or revert the single commit for this issue. Nothing here
touches the wire protocol unless the header says *(breaking)* — and none of the
early-phase issues do.

## 9. Follow-ups (do NOT do in this issue)

Flip defaults per `089` verdicts (separate dated commits).

## 10. Touch-and-learn notes

- Read the linked files first, top to bottom, before editing.
- Note one thing that surprised you in the commit message.
- If the step takes > S (<1 day) estimate, stop and split it — open a follow-up note.
