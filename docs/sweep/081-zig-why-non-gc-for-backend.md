# 081 Zig plan 1/10: why a non-GC language, and why Zig

| Field | Value |
|-------|-------|
| Status | `todo` (change to `doing` / `done` as you work) |
| Priority | P1 |
| Size | S (<1 day) |
| Area | zig |
| Phase | Week 6 — Zig track (read first) |
| Related | `082`, `089` |

## 1. Why this matters

This is the entry to the 10-part Zig track (`081-090`). After a month in the Go codebase you'll feel the GC pauses (rare but real during 100k scoring), cgo friction, and the 'rewrite everything in Rust?' temptation. This file sets the frame: stay Go for networked code, port NON-network CPU parts to Zig incrementally, behind flags, with parity tests.

Read this file, then touch the code. This issue is deliberately scoped so you can
finish it in one sitting and see the result in the running app.

## 2. Current behavior

Pure-Go backend (`internal/`), no cgo, no native libs (legacy C++ DLLs deleted; `NATIVE.md` describes the dead tree). GC pressure points: `FindRelated` scoring allocs (`catalog.go:209-227`, bigram maps per call `diceBigrams` `327-360`), `SearchRoots` walks, `recvExact/sendFile` 32KB bufs (fine). No profiles exist — feelings, not numbers.

Key files:

- `app.go` (Wails bridge) + `internal/` packages — see issue body for exact paths.

## 3. Problems / symptoms

- Premature rewrite risk: porting the TCP server to Zig replays `091`/`092` field bugs for zero user gain.
- Language-choice churn (Rust? C? Zig?) without criteria.
- No port boundary: 'rewrite backend in Zig' is unshippable as one issue.

## 4. Proposal (what "good" looks like)

Principles: (1) Go keeps ALL networked/IO code (server, framing, deadlines) — memory-safety + battle-tested; (2) Zig candidates are pure CPU: scoring (`082`), file-walk+filter (`083`), hashing (`084`); (3) C-ABI boundary (`085`), Go default + Zig opt-in (`090`); (4) numbers gate everything (`089`); (5) build stays one-command (`086`). Read `082-090` in order after this.

## 5. Step-by-step implementation

1. Read `082-090` headers (this sitting, no code).
2. Profile the three candidates on YOUR machine (100k catalog fixture `095`, 50k-file walk, 1GB hash) — record Go baselines.
3. Write the decision log (Zig vs Rust vs C vs 'stay Go'): criteria = C-ABI ease, cross-compile to windows/amd64, build friction, your learning curve.
4. Set the success bar: Zig path must be ≥2x faster or ≥5x less-alloc on at least one candidate AND parity-tested, else stay Go (document the 'no-port' outcome as success).
5. Do NOT write Zig yet — `082` picks the first target after numbers exist.

Work in small commits. After each step, run the relevant check:

```powershell
go build ./...; if ($?) { go test ./... }
cmd /c "pnpm.cmd --dir frontend build"
```

## 6. Acceptance criteria

- [ ] Baselines recorded for scoring/walk/hash on your hardware.
- [ ] Decision log names Zig (or another) with criteria, dated.
- [ ] `082-090` read; first target chosen (`082` default: scoring).

## 7. Tests to add / update

Baseline bench commands recorded (not CI-gated): `go test -bench` additions from `095`.

## 8. Risks and rollback

Rewrite fever. The guardrail: network code NEVER ports (write it here, sign it).

Rollback: `git stash` or revert the single commit for this issue. Nothing here
touches the wire protocol unless the header says *(breaking)* — and none of the
early-phase issues do.

## 9. Follow-ups (do NOT do in this issue)

`082` (first port), `089` (gates), `090` (flag discipline).

## 10. Touch-and-learn notes

- Read the linked files first, top to bottom, before editing.
- Note one thing that surprised you in the commit message.
- If the step takes > S (<1 day) estimate, stop and split it — open a follow-up note.
