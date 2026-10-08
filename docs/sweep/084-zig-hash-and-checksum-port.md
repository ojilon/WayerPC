# 084 Zig plan 4/10: hashing (SHA-256) port

| Field | Value |
|-------|-------|
| Status | `todo` (change to `doing` / `done` as you work) |
| Priority | P2 |
| Size | S (<1 day) |
| Area | zig |
| Phase | Week 7 — Zig track |
| Related | `081`, `005`, `026` |

## 1. Why this matters

Hashing (`005` DONE checksum, `026` dedup, `033` sha column) is CPU-bound streaming — a clean Zig candidate using a vendored SHA-256 (or std.crypto) with zero semantic risk (test vectors are universal). Smallest port after scoring.

Read this file, then touch the code. This issue is deliberately scoped so you can
finish it in one sitting and see the result in the running app.

## 2. Current behavior

Go `crypto/sha256` (stdlib, fast, assembly-assisted on amd64). Any Zig port competes with an excellent baseline — the bar (`089`) is 'match or beat with less alloc', not 'beat at all costs'.

Key files:

- `app.go` (Wails bridge) + `internal/` packages — see issue body for exact paths.

## 3. Problems / symptoms

- Go's sha256 is already hardware-friendly; Zig may not win on speed.
- Streaming API across C-ABI (init/update/final) needs state-handle design.
- Two hash impls = two code paths to keep vector-tested.

## 4. Proposal (what "good" looks like)

Spike-only first: Zig streaming hasher (init/update/final over opaque handle) vs Go stdlib on 100MB–2GB files; keep whichever wins on YOUR hardware; if Go wins, document 'stay Go' and close the Zig-hash idea (success = decision, not code). If Zig wins, use for `005` hot path + `026` backfill.

## 5. Step-by-step implementation

1. NIST vectors test (both impls, committed fixtures).
2. Bench matrix: 10MB/100MB/1GB × Go vs Zig (3 runs, note CPU/disk).
3. Streaming-handle C-ABI sketch (only if Zig competitive).
4. Decision record in this file's PR/commit.
5. If adopted: wire `recvExact`-side hashing to Zig streaming handle.

Work in small commits. After each step, run the relevant check:

```powershell
go build ./...; if ($?) { go test ./... }
cmd /c "pnpm.cmd --dir frontend build"
```

## 6. Acceptance criteria

- [ ] Vectors green on both impls.
- [ ] Bench table recorded; explicit adopt/stay decision.
- [ ] No regression to `005` format regardless of choice.

## 7. Tests to add / update

Vector tests; bench harness (manual-run, numbers in commit).

## 8. Risks and rollback

Chasing 5% on hashing while scoring/walk dominate — check profiles first (`081` baselines decide priority).

Rollback: `git stash` or revert the single commit for this issue. Nothing here
touches the wire protocol unless the header says *(breaking)* — and none of the
early-phase issues do.

## 9. Follow-ups (do NOT do in this issue)

`026` (uses winner), `089` (records verdict).

## 10. Touch-and-learn notes

- Read the linked files first, top to bottom, before editing.
- Note one thing that surprised you in the commit message.
- If the step takes > S (<1 day) estimate, stop and split it — open a follow-up note.
