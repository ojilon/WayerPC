# 035 Case-insensitive path uniqueness (Windows)

| Field | Value |
|-------|-------|
| Status | `todo` (change to `doing` / `done` as you work) |
| Priority | P2 |
| Size | S (<1 day) |
| Area | backend |
| Phase | Week 5 — hardening |
| Related | `032` |

## 1. Why this matters

Windows paths are case-insensitive; SQLite `UNIQUE` is case-sensitive by default. `D:\Pics\A.jpg` vs `d:\pics\a.jpg` become two rows pointing at one file — prune (`032`) and dedup (`026`) both get confused.

Read this file, then touch the code. This issue is deliberately scoped so you can
finish it in one sitting and see the result in the running app.

## 2. Current behavior

`path TEXT NOT NULL UNIQUE` (`catalog.go:24-30`); `AddPath` uses `filepath.Abs` (`catalog.go:101-104`) with no normalization; `INSERT OR REPLACE` treats case variants as distinct. `Remove` tries Abs + symlink-eval fallback (`catalog.go:140-163`) — partial mitigation only.

Key files:

- `app.go` (Wails bridge) + `internal/` packages — see issue body for exact paths.

## 3. Problems / symptoms

- Duplicate rows for one file after drive-letter case changes or relink.
- MATCHES shows the same file twice.
- Prune counts ghosts that are really case twins.

## 4. Proposal (what "good" looks like)

Normalize on write: on Windows, store `Abs` + casefolded comparison via `COLLATE NOCASE` unique index (or a `path_norm` column). Keep display path as-typed. Dedupe existing twins with a one-time cleanup in migration (`033` framework if landed, else standalone).

## 5. Step-by-step implementation

1. Decide: `COLLATE NOCASE` on `path` (simplest) vs norm column (explicit). Recommend NOCASE + test.
2. Normalize separators (`/` vs `\`) before compare on Windows.
3. One-time cleanup: find NOCASE-duplicates, keep newest, delete rest (log count).
4. Tests: add `D:\X\A.JPG` then `d:\x\a.jpg` → one row.
5. Verify Linux behavior unchanged (case-sensitive FS — NOCASE could over-merge `A` vs `a` as different files; gate by GOOS or accept documented tradeoff).

Work in small commits. After each step, run the relevant check:

```powershell
go build ./...; if ($?) { go test ./... }
cmd /c "pnpm.cmd --dir frontend build"
```

## 6. Acceptance criteria

- [ ] Case-twin inserts collapse to one row on Windows.
- [ ] Linux: distinct-case files stay distinct (or tradeoff documented).
- [ ] Existing twins cleaned with logged count.

## 7. Tests to add / update

Case-twin tests (GOOS-gated); cleanup test on crafted DB.

## 8. Risks and rollback

Linux over-merge if NOCASE applied blindly — gate per-OS or use norm column only on Windows.

Rollback: `git stash` or revert the single commit for this issue. Nothing here
touches the wire protocol unless the header says *(breaking)* — and none of the
early-phase issues do.

## 9. Follow-ups (do NOT do in this issue)

`026` (dedup interacts: same bytes + case-twin paths).

## 10. Touch-and-learn notes

- Read the linked files first, top to bottom, before editing.
- Note one thing that surprised you in the commit message.
- If the step takes > S (<1 day) estimate, stop and split it — open a follow-up note.
