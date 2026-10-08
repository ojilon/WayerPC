# 002 Roadmap overview and suggested order

| Field | Value |
|-------|-------|
| Status | `todo` (change to `doing` / `done` as you work) |
| Priority | P0 |
| Size | S (<2h) |
| Area | meta |
| Phase | Week 1 — orientation |
| Related | `001`, `003`, `100` |

## 1. Why this matters

100 files without an order is a maze. This is the month-or-two plan: orientation → protocol hardening → backend → search → frontend → installer → observability → Zig → advanced tests → 1.0 gate.

Read this file, then touch the code. This issue is deliberately scoped so you can
finish it in one sitting and see the result in the running app.

## 2. Current behavior

Prioritization lives only in `docs/SUGGESTIONS-v0.2.1.md` (P0/P1/P2 + S/M/L). It has no calendar mapping and no dependency graph. `app.go:84-106` (startup), `internal/server/server.go:133-161` (Start/Accept), `frontend/src/main.ts:100-132` (boot) are the three flows to learn first.

Key files:

- `app.go` (Wails bridge) + `internal/` packages — see issue body for exact paths.

## 3. Problems / symptoms

- P0 items (READY newline, DiskUsage cache, EnVar drop, conformance script) are buried among P2 nice-to-haves.
- No phase gate: e.g. FTS5 (`027`) needs the benchmark (`095`) first.
- Zig files (`081-090`) look tempting on day 1 but require Go familiarity.

## 4. Proposal (what "good" looks like)

8 phases across ~8 weeks: (1) orientation 001-003; (2) protocol correctness 004-014; (3) backend reliability 015-035; (4) search depth 036-046; (5) storage+config 047-055; (6) frontend 056-069; (7) installer+observability+security 070-080; (8) Zig 081-090 then tests+release 091-100. One phase per week, 2-3 files per evening.

## 5. Step-by-step implementation

1. Skim all 100 headers (titles only) to build a mental map.
2. Tag each file's Status: leave `todo`; mark any already-true as `done` with commit ref.
3. Schedule: Weeks 1-2 files 001-030, Weeks 3-4 files 031-060, Weeks 5-6 files 061-080, Weeks 7-8 files 081-100.
4. Enforce gates: do `095` before `027`; `091` before any protocol change; `081` before `082-090`.
5. Track progress in `099` changelog as you go.
6. Re-plan at each phase boundary; move files, don't delete them.

Work in small commits. After each step, run the relevant check:

```powershell
go build ./...; if ($?) { go test ./... }
cmd /c "pnpm.cmd --dir frontend build"
```

## 6. Acceptance criteria

- [ ] Every file has a Phase field you agree with (edit outliers).
- [ ] Gates documented: `095→027`, `091→004-011`, `081→082-090`.
- [ ] You have a personal Week 1 pick (suggested: `004`, `022`, `056`).

## 7. Tests to add / update

Doc-only. Sanity: every `docs/sweep/*.md` contains a `| Phase |` row.

## 8. Risks and rollback

Plan rot. Mitigate with the `099` changelog — one line per finished issue keeps the plan honest.

Rollback: `git stash` or revert the single commit for this issue. Nothing here
touches the wire protocol unless the header says *(breaking)* — and none of the
early-phase issues do.

## 9. Follow-ups (do NOT do in this issue)

`100` (1.0 definition — the finish line this order drives toward).

## 10. Touch-and-learn notes

- Read the linked files first, top to bottom, before editing.
- Note one thing that surprised you in the commit message.
- If the step takes > S (<2h) estimate, stop and split it — open a follow-up note.
