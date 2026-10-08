# 042 Content-snippet search mode (opt-in, late)

| Field | Value |
|-------|-------|
| Status | `todo` (change to `doing` / `done` as you work) |
| Priority | P2 |
| Size | L (3+ days) |
| Area | search |
| Phase | Week 8 — advanced (defer) |
| Related | `043` |

## 1. Why this matters

Filename search fails when the user remembers text inside the file ('the invoice with OTC…'). An opt-in content mode (text files under a size cap, with snippets) answers it — but it is expensive, so it ships last and off by default.

Read this file, then touch the code. This issue is deliberately scoped so you can
finish it in one sitting and see the result in the running app.

## 2. Current behavior

No content reading anywhere in search; `SearchAppFolders` is exact-name; catalog has no content index. `recvExact`/`sendFile` stream bytes but never inspect them.

Key files:

- `app.go` (Wails bridge) + `internal/` packages — see issue body for exact paths.

## 3. Problems / symptoms

- 'I know what's in it, not what it's called' has no answer.
- Full-content grep over drives is minutes-to-hours — naive version will be judged as 'search is slow'.
- Binary files (videos, zips) waste effort if sniffed blindly.

## 4. Proposal (what "good" looks like)

Explicitly DEFERRED: design only in this phase. Opt-in toggle 'Search inside files (slow)'; scope: text-ish extensions (<5MB, skip binaries via sniffing); return `Hit + snippet`; never default-on; never index content without consent (`043` decides). This issue = spike + limits doc, not the full build — timebox 1 day, then stop if slow.

## 5. Step-by-step implementation

1. Timeboxed spike: grep `*.{txt,md,csv,log}` under a fixture; measure throughput.
2. Define limits: ext allow-list, 5MB cap, 30s budget, cancelable (`036`).
3. Snippet extraction (first match ±40 chars, encoding-tolerant).
4. UI mock behind a disabled toggle (no backend wiring yet).
5. Decision record: build / defer / drop based on numbers.
6. If deferred, keep this file as the design for later.

Work in small commits. After each step, run the relevant check:

```powershell
go build ./...; if ($?) { go test ./... }
cmd /c "pnpm.cmd --dir frontend build"
```

## 6. Acceptance criteria

- [ ] Spike numbers recorded (MB/s, fixture size).
- [ ] Explicit build/defer decision logged.
- [ ] No perf regression to filename search (untouched path).

## 7. Tests to add / update

Spike-only: snippet extractor unit tests on fixtures.

## 8. Risks and rollback

Scope explosion (OCR, PDFs, docx) — explicitly out. Text-only, or stop.

Rollback: `git stash` or revert the single commit for this issue. Nothing here
touches the wire protocol unless the header says *(breaking)* — and none of the
early-phase issues do.

## 9. Follow-ups (do NOT do in this issue)

`043` (index would make this fast — revisit after).

## 10. Touch-and-learn notes

- Read the linked files first, top to bottom, before editing.
- Note one thing that surprised you in the commit message.
- If the step takes > L (3+ days) estimate, stop and split it — open a follow-up note.
