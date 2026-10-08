# 038 User-search ranking (beyond substring)

| Field | Value |
|-------|-------|
| Status | `todo` (change to `doing` / `done` as you work) |
| Priority | P2 |
| Size | M (1-3 days) |
| Area | search |
| Phase | Week 4 — search depth |
| Related | `027`, `045` |

## 1. Why this matters

Import search is pure substring (`strings.Contains`, `search.go:141,158`) sorted by name (`search.go:166-171`). `report.pdf` beats `report-final-2026.pdf` only alphabetically; typo `reprot` finds nothing. Catalog scoring (`catalog.go:189-206`, exact→1.0/substring-boost/bigram-Dice) already solves this for `/ask` — unify the brains.

Read this file, then touch the code. This issue is deliberately scoped so you can
finish it in one sitting and see the result in the running app.

## 2. Current behavior

Two scoring worlds: catalog fuzzy (`Score`, Dice bigrams) vs user-search dumb-substring. `SearchUser` limit 200 (`import.ts:91`) truncates alphabetically, not by relevance — the best hit may be cut.

Key files:

- `app.go` (Wails bridge) + `internal/` packages — see issue body for exact paths.

## 3. Problems / symptoms

- Best match buried at #199 or truncated away.
- Typos return zero with no 'did you mean?'.
- Inconsistent UX: phone `/ask` fuzzy, PC import strict.

## 4. Proposal (what "good" looks like)

Rank user hits with catalog `Score()` (extract shared scorer into `internal/match` package later; for this issue just reuse): exact > prefix > substring > fuzzy ≥ cutoff; sort by score then name; apply limit AFTER ranking (walk deeper or multi-pass for top-N). Add 'did you mean' hint when zero hits but fuzzy near-misses exist in walked sample.

## 5. Step-by-step implementation

1. Refactor: move `Score`+`diceBigrams` to shared helper importable by `search` (no behavior change; update callers).
2. `SearchRoots`: collect-then-rank (or bounded heap top-N) instead of name-sort.
3. Truncate to limit post-rank; include score in `Hit` (internal; UI shows plain list).
4. Fixture test: query `reprot` ranks `report.pdf` top; `photo` prefers `photo.jpg` over `my-photos-backup.zip`? (define + lock expectations).
5. Document ranking order in Import view tooltip.

Work in small commits. After each step, run the relevant check:

```powershell
go build ./...; if ($?) { go test ./... }
cmd /c "pnpm.cmd --dir frontend build"
```

## 6. Acceptance criteria

- [ ] Typo query returns sensible top-3 on fixture tree.
- [ ] Limit truncation keeps best hits (test with >limit fixtures).
- [ ] `/ask` and import ranking agree on shared fixtures.

## 7. Tests to add / update

Ranking golden tests (fixture tree + query→expected top order).

## 8. Risks and rollback

Perf: scoring every walked file costs more than substring — gate fuzzy pass behind 'substring collected < limit' (two-phase: exact/substring first, fuzzy fill second).

Rollback: `git stash` or revert the single commit for this issue. Nothing here
touches the wire protocol unless the header says *(breaking)* — and none of the
early-phase issues do.

## 9. Follow-ups (do NOT do in this issue)

`027` (FTS prefilter feeds the ranker at scale), Zig `082` ports the shared scorer).

## 10. Touch-and-learn notes

- Read the linked files first, top to bottom, before editing.
- Note one thing that surprised you in the commit message.
- If the step takes > M (1-3 days) estimate, stop and split it — open a follow-up note.
