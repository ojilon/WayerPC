# 045 Search history + saved queries

| Field | Value |
|-------|-------|
| Status | `todo` (change to `doing` / `done` as you work) |
| Priority | P2 |
| Size | S (<1 day) |
| Area | search/frontend |
| Phase | Week 4 — frontend depth |
| Related | `038` |

## 1. Why this matters

Import searches repeat: same 'invoice', same 'photos 2025' every week. Re-typing is friction; history (last 10, one click) is a tiny feature with daily payoff.

Read this file, then touch the code. This issue is deliberately scoped so you can
finish it in one sitting and see the result in the running app.

## 2. Current behavior

No history: `import.ts` query input is stateless; no settings kv usage for search. `GetSetting/SetSetting` (`app.go:493-514`) exists and could persist, but nothing does for search.

Key files:

- `app.go` (Wails bridge) + `internal/` packages — see issue body for exact paths.

## 3. Problems / symptoms

- Repeated typing of the same queries.
- No signal about what users search (even locally) to guide `038` ranking.
- Dead-end UX after 'no matches' (no retry suggestion).

## 4. Proposal (what "good" looks like)

Persist last 10 queries (settings kv `search.history`, local only); show as clickable chips under the search bar; add 'Save' star for pinned queries (kv `search.saved`).Selecting re-runs instantly. No telemetry — local only, documented.

## 5. Step-by-step implementation

1. History helpers in `import.ts` (load/save via `api.getSetting/setSetting`).
2. Chips UI under search bar; clear-history button.
3. Saved (pinned) section; pin/unpin per chip.
4. Empty-state hint uses history ('try “invoice” again?').
5. Cap 10 history + 20 saved; dedupe + most-recent-first.

Work in small commits. After each step, run the relevant check:

```powershell
go build ./...; if ($?) { go test ./... }
cmd /c "pnpm.cmd --dir frontend build"
```

## 6. Acceptance criteria

- [ ] Restart keeps history + saved.
- [ ] Clicking a chip re-runs the search.
- [ ] No network calls; local-only documented in code comment.

## 7. Tests to add / update

History rotation unit (pure funcs) — extract for testability.

## 8. Risks and rollback

Storing queries is mild PII on shared PCs — note in Settings; provide Clear. Keep in local settings table, never logged.

Rollback: `git stash` or revert the single commit for this issue. Nothing here
touches the wire protocol unless the header says *(breaking)* — and none of the
early-phase issues do.

## 9. Follow-ups (do NOT do in this issue)

Ranking (`038`) could weight history — note, don't build.

## 10. Touch-and-learn notes

- Read the linked files first, top to bottom, before editing.
- Note one thing that surprised you in the commit message.
- If the step takes > S (<1 day) estimate, stop and split it — open a follow-up note.
