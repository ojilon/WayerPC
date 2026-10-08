# 009 Richer MATCHES rows (breaking)

| Field | Value |
|-------|-------|
| Status | `todo` (change to `doing` / `done` as you work) |
| Priority | P2 |
| Size | M (1-3 days) |
| Area | protocol |
| Phase | Week 6 — advanced protocol |
| Related | `006`, `028` |

## 1. Why this matters

Today `MATCHES` is names-only (`protocol.go:189-211`, max 50). The phone cannot show sizes/dates, so users pick blind among similar names.

Read this file, then touch the code. This issue is deliberately scoped so you can
finish it in one sitting and see the result in the running app.

## 2. Current behavior

`sendMatches` writes `MATCHES <n>\n<name>\n…`. No sizes, no disambiguation when two folders hold same filename. Legacy parity is the reason — but the Android list UI is ready to parse more.

Key files:

- `app.go` (Wails bridge) + `internal/` packages — see issue body for exact paths.

## 3. Problems / symptoms

- Phone shows bare names with no sizes/dates — picking among five `report.pdf` is blind luck.
- 50-row truncation in `sendMatches` is invisible to the phone; no "more exist" signal.
- Same filename in two folders is indistinguishable — no path or date discriminator.

## 4. Proposal (what "good" looks like)

Proto-gated (≥2 or ≥3): `name\tbytes\tmodified` per row. Old clients break if they split naively — hence coordinate + gate. Alternatively add `/ask2` while keeping `/ask` frozen; decide during implementation.

## 5. Step-by-step implementation

1. Decide vehicle: extend `MATCHES` under proto ≥ 2 vs new `/ask2` command (recommend latter for safety).
2. Implement tab-separated rows; cap still 50.
3. Update `sendMatches` signature to include size/modtime from catalog rows.
4. Android: parse new columns, show size/date in picker.
5. Tests for both formats via capability negotiation.
6. Doc the version matrix.

Work in small commits. After each step, run the relevant check:

```powershell
go build ./...; if ($?) { go test ./... }
cmd /c "pnpm.cmd --dir frontend build"
```

## 6. Acceptance criteria

- [ ] New client shows sizes/dates; old client pinned to old format still works.
- [ ] 50-row cap retained; truncation noted in reply.
- [ ] Version matrix documented.

## 7. Tests to add / update

Golden tests for both MATCHES formats.

## 8. Risks and rollback

Breaking if done in place — prefer `/ask2` addition over changing `/ask`. Needs Android coordination.

Rollback: `git stash` or revert the single commit for this issue. Nothing here
touches the wire protocol unless the header says *(breaking)* — and none of the
early-phase issues do.

## 9. Follow-ups (do NOT do in this issue)

`028` (pagination if 50-cap becomes a UX limit).

## 10. Touch-and-learn notes

- Read the linked files first, top to bottom, before editing.
- Note one thing that surprised you in the commit message.
- If the step takes > M (1-3 days) estimate, stop and split it — open a follow-up note.
