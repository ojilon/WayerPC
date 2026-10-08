# 077 Log rotation + retention policy

| Field | Value |
|-------|-------|
| Status | `todo` (change to `doing` / `done` as you work) |
| Priority | P2 |
| Size | S (<1 day) |
| Area | backend |
| Phase | Week 5 — observability |
| Related | `025` |

## 1. Why this matters

`025` persists logs; without a policy they grow forever (5MB rotation proposed there). This issue sets the numbers + the UI: how much, how long, where, and the 'clear everything' path.

Read this file, then touch the code. This issue is deliberately scoped so you can
finish it in one sitting and see the result in the running app.

## 2. Current behavior

Memory ring 2000 lines (`app.go:24,179-192`); `ClearLogs` empties memory only. No disk policy yet (`025` designs it). No privacy story (filenames are PII).

Key files:

- `app.go` (Wails bridge) + `internal/` packages — see issue body for exact paths.

## 3. Problems / symptoms

- Unbounded growth if `025` lands without caps.
- No user control (keep 7 days? 50MB?).
- Clear must mean clear (memory + disk) for privacy-sensitive users.

## 4. Proposal (what "good" looks like)

Policy: `Data/logs.jsonl` + 1 rotated sibling, 5MB each, 2000-line memory ring unchanged; Settings 'Diagnostics' row shows size + Clear (memory+disk) + 'Open folder'. `025` implements; this issue reviews/tunes + adds the UI + documents. If `025` isn't done, implement the policy here directly (merge them — note the merge in both files).

## 5. Step-by-step implementation

1. If `025` done: add size display + Clear-disk + folder button; verify rotation caps.
2. If not: implement persistence WITH this policy (don't land unbounded).
3. Privacy line in UI ('logs contain filenames — review before sharing').
4. Test: spam 20MB → disk ≤ ~10MB + current; Clear → 0 bytes + empty Console.
5. Document in README troubleshooting ('attach logs from …').

Work in small commits. After each step, run the relevant check:

```powershell
go build ./...; if ($?) { go test ./... }
cmd /c "pnpm.cmd --dir frontend build"
```

## 6. Acceptance criteria

- [ ] Disk bounded with proof (spam test numbers in commit).
- [ ] Clear wipes memory + disk.
- [ ] Size visible in Settings.

## 7. Tests to add / update

Rotation-cap test; clear-covers-disk test.

## 8. Risks and rollback

Deleting logs users wanted for a bug — confirm before Clear; Download (`056`) offered first.

Rollback: `git stash` or revert the single commit for this issue. Nothing here
touches the wire protocol unless the header says *(breaking)* — and none of the
early-phase issues do.

## 9. Follow-ups (do NOT do in this issue)

Log-level persistence per `076` (debug logs excluded from disk by default?).

## 10. Touch-and-learn notes

- Read the linked files first, top to bottom, before editing.
- Note one thing that surprised you in the commit message.
- If the step takes > S (<1 day) estimate, stop and split it — open a follow-up note.
