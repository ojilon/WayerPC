# 044 App-folder search: recursive toggle + case options

| Field | Value |
|-------|-------|
| Status | `todo` (change to `doing` / `done` as you work) |
| Priority | P2 |
| Size | S (<1 day) |
| Area | search |
| Phase | Week 4 — search depth |
| Related | `038` |

## 1. Why this matters

`SearchAppFolders` (`search.go:177-196`) is exact-name, always recursive, case-insensitive. `/ask` fallback depends on it — but Import's 'search app folders' path (`app.go:298-310`) gives no control: subfolder noise, no prefix mode.

Read this file, then touch the code. This issue is deliberately scoped so you can
finish it in one sitting and see the result in the running app.

## 2. Current behavior

Walk all of shared+received per query; `strings.ToLower(name)==query`. No depth option, no prefix/substring switch, no limit. Fine for small folders; noisy when received/ holds 5k uploads.

Key files:

- `app.go` (Wails bridge) + `internal/` packages — see issue body for exact paths.

## 3. Problems / symptoms

- Exact-only misses `report-final.pdf` when phone asks `report` (catalog fuzzy covers it first, but fallback path is brittle).
- Deep nesting (received/2024/…) always fully walked.
- No result cap — pathological folders return unbounded lists (caller doesn't cap).

## 4. Proposal (what "good" looks like)

Add options struct `{Exact, Prefix, Contains; MaxDepth (-1=all); Limit}`; default preserves current behavior; expose prefix mode to the phone fallback (try exact → prefix → contains). Cap at 200 with truncation note.

## 5. Step-by-step implementation

1. Add `SearchAppFoldersOpt(query, folders, opt)`; keep old func as default wrapper.
2. Wire fallback in `protocol.go:75` to try exact then prefix.
3. Test matrix: exact/prefix/contains/depth-1 on fixtures.
4. Log which mode succeeded (debug) for field diagnosis.
5. Document fallback chain in README protocol section.

Work in small commits. After each step, run the relevant check:

```powershell
go build ./...; if ($?) { go test ./... }
cmd /c "pnpm.cmd --dir frontend build"
```

## 6. Acceptance criteria

- [ ] `report` finds `report-final.pdf` via fallback prefix mode.
- [ ] Depth-1 option excludes nested hits.
- [ ] Defaults preserve legacy exact behavior (regression test).

## 7. Tests to add / update

Mode-matrix unit tests.

## 8. Risks and rollback

Changing fallback recall alters phone-visible behavior — test against `091` conformance expectations first.

Rollback: `git stash` or revert the single commit for this issue. Nothing here
touches the wire protocol unless the header says *(breaking)* — and none of the
early-phase issues do.

## 9. Follow-ups (do NOT do in this issue)

`038` (shared ranking could order fallback hits too).

## 10. Touch-and-learn notes

- Read the linked files first, top to bottom, before editing.
- Note one thing that surprised you in the commit message.
- If the step takes > S (<1 day) estimate, stop and split it — open a follow-up note.
