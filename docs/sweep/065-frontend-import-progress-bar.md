# 065 Import + search progress bars

| Field | Value |
|-------|-------|
| Status | `todo` (change to `doing` / `done` as you work) |
| Priority | P1 |
| Size | M (1-3 days) |
| Area | frontend/backend |
| Phase | Week 4 — frontend depth |
| Related | `037`, `029`, `064` |

## 1. Why this matters

`037` streams search progress; imports have the same black box (`doImport`/`catalogAddFolder` show static 'Importing…' `import.ts:67-83`). `029` batch transactions give natural progress points (per 1000 rows). One pattern for both.

Read this file, then touch the code. This issue is deliberately scoped so you can
finish it in one sitting and see the result in the running app.

## 2. Current behavior

Static status text; `ImportResult{added,skipped}` only at end. No `import-progress` event. `037` design exists for search — mirror it, don't reinvent.

Key files:

- `app.go` (Wails bridge) + `internal/` packages — see issue body for exact paths.

## 3. Problems / symptoms

- 50k import looks hung for minutes (users kill it mid-transaction → rollback confusion).
- No per-batch error visibility (which files skipped? why?).
- Search and import progress look/behave differently if built separately.

## 4. Proposal (what "good" looks like)

`import-progress {done, total?, phase}` events per batch (total from `064` dry-run when available, else done-only + spinner); Import view determinate bar when total known, indeterminate+counts otherwise; completion summary (added/skipped + 'view in Files' link). Skipped-sample list (first 10 reasons) in expandable.

## 5. Step-by-step implementation

1. Backend: emit per-batch in `AddFolder` txn loop (`029`) + per-path in `CatalogAddPaths` (throttle 250ms).
2. Event `import-progress` (+ `import-done {added, skipped, sampleErrors}`); document with `037`.
3. `import.ts`: progress bar UI + cancel (`036` ctx through AddFolder — extend signature compatibly).
4. Skipped-sample rendering (first 10 + count).
5. Mock events in `dev-mock.ts`.

Work in small commits. After each step, run the relevant check:

```powershell
go build ./...; if ($?) { go test ./... }
cmd /c "pnpm.cmd --dir frontend build"
```

## 6. Acceptance criteria

- [ ] 10k import shows live done/total + bar; cancel stops promptly.
- [ ] Skipped files explained (sample reasons visible).
- [ ] Search and import bars share styling/behavior.

## 7. Tests to add / update

Batch-event test (counts monotonic, done events exact); mock-UI smoke.

## 8. Risks and rollback

Event ordering across concurrent imports — serialize per-operation id in payload (`opId`).

Rollback: `git stash` or revert the single commit for this issue. Nothing here
touches the wire protocol unless the header says *(breaking)* — and none of the
early-phase issues do.

## 9. Follow-ups (do NOT do in this issue)

`028` (post-import 'view' jumps to paged catalog), `093` (mock parity for new events).

## 10. Touch-and-learn notes

- Read the linked files first, top to bottom, before editing.
- Note one thing that surprised you in the commit message.
- If the step takes > M (1-3 days) estimate, stop and split it — open a follow-up note.
