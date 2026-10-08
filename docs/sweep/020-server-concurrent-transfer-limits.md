# 020 Concurrent transfer limits + queuing

| Field | Value |
|-------|-------|
| Status | `todo` (change to `doing` / `done` as you work) |
| Priority | P2 |
| Size | M (1-3 days) |
| Area | backend |
| Phase | Week 5 — hardening |
| Related | `011`, `069` |

## 1. Why this matters

`011` caps connections; this caps *transfers*: 4 phones uploading 1GB each over one hotspot melts throughput and disk. A semaphore (e.g. 2 active transfers, rest get `ERROR busy_retry`) keeps the link usable.

Read this file, then touch the code. This issue is deliberately scoped so you can
finish it in one sitting and see the result in the running app.

## 2. Current behavior

No transfer semaphore: every conn with `/upload` streams immediately (`protocol.go:153-163`). `active` counts conns, not transfers. Disk writes contend; rates collapse; all four users blame the app.

Key files:

- `app.go` (Wails bridge) + `internal/` packages — see issue body for exact paths.

## 3. Problems / symptoms

- Throughput collapse under concurrency.
- No fairness: first big upload starves small asks.
- No visibility: Console shows Active links but not queued/waiting.

## 4. Proposal (what "good" looks like)

Semaphore around `streamFile` + upload receive path (e.g. 2 concurrent, queue with timeout 30s then `ERROR busy_retry`). Prioritize `/ask` (small reads) over bulk uploads. Show queued count in `Stats` + Console.

## 5. Step-by-step implementation

1. Add `transferSem chan struct{}` (size 2, configurable) + `queued` counter.
2. Acquire with timeout in `streamFile`/`handleUpload`; on timeout `ERROR busy_retry`.
3. Prioritize: try `/ask` fast-path first (reserve 1 slot for asks).
4. Expose `queued` in `Snapshot()`; Console shows '2 active, 1 queued'.
5. Load test: 4 parallel uploads → 2 run, 2 retry cleanly.
6. Document client retry guidance (backoff 2s).

Work in small commits. After each step, run the relevant check:

```powershell
go build ./...; if ($?) { go test ./... }
cmd /c "pnpm.cmd --dir frontend build"
```

## 6. Acceptance criteria

- [ ] 4-way upload storm: no corruption, all complete after retries.
- [ ] Small `/ask` stays fast during a big upload.
- [ ] Console shows queue depth.

## 7. Tests to add / update

Parallel-upload stress test (local loopback, small sizes).

## 8. Risks and rollback

Deadlock if semaphore held across fatal paths — use defer release + timeout acquire.

Rollback: `git stash` or revert the single commit for this issue. Nothing here
touches the wire protocol unless the header says *(breaking)* — and none of the
early-phase issues do.

## 9. Follow-ups (do NOT do in this issue)

`078` (queue metrics), Android backoff-on-busy.

## 10. Touch-and-learn notes

- Read the linked files first, top to bottom, before editing.
- Note one thing that surprised you in the commit message.
- If the step takes > M (1-3 days) estimate, stop and split it — open a follow-up note.
