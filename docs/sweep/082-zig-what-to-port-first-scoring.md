# 082 Zig plan 2/10: port the similarity scorer first

| Field | Value |
|-------|-------|
| Status | `todo` (change to `doing` / `done` as you work) |
| Priority | P1 |
| Size | M (1-3 days) |
| Area | zig |
| Phase | Week 7 — Zig track |
| Related | `081`, `038`, `089` |

## 1. Why this matters

The scorer (`Score`+`diceBigrams`, `catalog.go:189-206,327-360`) is the ideal first Zig target: pure function (query, name → f64), hot in every `/ask` (`027`/`095` scale story), zero IO, trivially parity-testable. Small blast radius, real payoff if 100k libraries matter.

Read this file, then touch the code. This issue is deliberately scoped so you can
finish it in one sitting and see the result in the running app.

## 2. Current behavior

Go scorer: lowercases, exact/substring-boost/bigram-Dice; per-call map allocs (`counts := make(map[string]int…)`) — the exact alloc Zig would kill. Callers: `FindRelated` linear scan; `038` wants it shared with user-search too.

Key files:

- `app.go` (Wails bridge) + `internal/` packages — see issue body for exact paths.

## 3. Problems / symptoms

- No shared home: scorer lives in `catalog` but `search` needs it (`038`).
- Unicode (rune bigrams) semantics must match exactly — easy to subtly diverge.
- Boundary cost (Go→Zig per name) could eat gains if called per-row naively (batch instead).

## 4. Proposal (what "good" looks like)

Steps: (1) extract Go scorer to `internal/match` (pure Go, both callers, golden tests) — do this BEFORE any Zig; (2) Zig `match` lib, C-ABI `score(query_ptr,len,name_ptr,len)->f64`, batch API `score_batch` to amortize crossings; (3) parity harness (1M random pairs incl. unicode, cutoff edge 0.2) + bench. Go stays default; Zig behind flag (`090`).

## 5. Step-by-step implementation

1. Extract `internal/match` from `catalog.go` (move + alias old funcs; all tests green).
2. Golden corpus: 10k (query,name)→score fixtures from Go (JSON) committed.
3. Zig `build.zig` lib: toLower (unicode-aware? decide: ASCII-fold + document; full unicode tables are the hard part — spike first), Dice over bigrams, boost rule; must match goldens bit-tolerant (f64 epsilon 1e-9).
4. C-ABI batch fn; Go wrapper with build-tag `zigmatch` (off by default).
5. Bench: Go vs Zig on 100k names; record in `089`.
6. Wire `FindRelated` to optionally use Zig batch when flag on.

Work in small commits. After each step, run the relevant check:

```powershell
go build ./...; if ($?) { go test ./... }
cmd /c "pnpm.cmd --dir frontend build"
```

## 6. Acceptance criteria

- [ ] Golden parity: 10k fixtures match within epsilon.
- [ ] Bench numbers recorded (both, same machine).
- [ ] Flag-off build has zero Zig dependency (pure Go path intact).

## 7. Tests to add / update

Golden parity test (CI-gated for Go side); Zig-side `zig build test` too.

## 8. Risks and rollback

Unicode case-fold divergence — scope ASCII-first + document; full tables only if parity demands. Crossing cost — batch API mandatory.

Rollback: `git stash` or revert the single commit for this issue. Nothing here
touches the wire protocol unless the header says *(breaking)* — and none of the
early-phase issues do.

## 9. Follow-ups (do NOT do in this issue)

`085` (ABI pattern reused), `089` (numbers judged), `038` (both search paths share it).

## 10. Touch-and-learn notes

- Read the linked files first, top to bottom, before editing.
- Note one thing that surprised you in the commit message.
- If the step takes > M (1-3 days) estimate, stop and split it — open a follow-up note.
