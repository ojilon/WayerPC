# 008 Upload resume from offset (breaking-ish)

| Field | Value |
|-------|-------|
| Status | `todo` (change to `doing` / `done` as you work) |
| Priority | P1 |
| Size | M (1-3 days) |
| Area | protocol |
| Phase | Week 6 — advanced protocol |
| Related | `005`, `006`, `091` |

## 1. Why this matters

Flaky hotspots restart hundred-MB uploads from zero. Resume (`/upload <size> <name> <offset>`, server appends to `.part`) is the single biggest reliability win for big files — but it changes semantics, so it ships late, gated on hello proto ≥ 2.

Read this file, then touch the code. This issue is deliberately scoped so you can
finish it in one sitting and see the result in the running app.

## 2. Current behavior

`handleUpload` (`protocol.go:86-175`) always creates a fresh `.part` (`os.Create`), requires exact `filesize` bytes, deletes `.part` on mismatch. No offset concept; `maxUpload` caps at 2GiB (`server.go:58`).

Key files:

- `app.go` (Wails bridge) + `internal/` packages — see issue body for exact paths.

## 3. Problems / symptoms

- 90%-done uploads that drop restart fully — users stop trusting big transfers.
- `.part` files are deleted on failure instead of reused.
- No way to query 'how much do you already have?'.

## 4. Proposal (what "good" looks like)

Gate behind proto ≥ 2 (see `006`): client sends `/upload <size> <name> <offset>`; server stats existing `.part`, seeks, replies `READY <offset>`; client streams the tail. Add `/stat <name>` (returns `SIZE <n>`) so a fresh client can discover resumable state. Keep legacy path untouched for proto 1.

## 5. Step-by-step implementation

1. Ship `006` hello first; define proto 2 = resume + checksum (`005`).
2. Implement offset parsing + validation (`offset <= size`, `offset <= existing .part size`).
3. Change `.part` handling: `os.OpenFile(O_APPEND)` when resuming; keep `os.Create` for fresh.
4. Reply `READY <offset>` (newline-terminated per `004`).
5. Add `/stat <name>` query command.
6. Tests: partial `.part` + resume completes; byte-identical final file; offset mismatch → ERROR.
7. Bump `version.json` protocolVersion 1→2; coordinate Android release.

Work in small commits. After each step, run the relevant check:

```powershell
go build ./...; if ($?) { go test ./... }
cmd /c "pnpm.cmd --dir frontend build"
```

## 6. Acceptance criteria

- [ ] Kill mid-upload at 50%, resume completes; sha256 matches source.
- [ ] Proto-1 client unaffected (no offset → legacy path).
- [ ] Protocol doc marks resume as proto-2-only.

## 7. Tests to add / update

Resume integration test (scripted kill + resume); fuzz offset parsing (`092`).

## 8. Risks and rollback

Breaking-ish: needs coordinated app releases. Partial-file confusion if user replaces source file between attempts — include size check and warn.

Rollback: `git stash` or revert the single commit for this issue. Nothing here
touches the wire protocol unless the header says *(breaking)* — and none of the
early-phase issues do.

## 9. Follow-ups (do NOT do in this issue)

Android resume UI; `078` metrics for resumed-vs-fresh counts.

## 10. Touch-and-learn notes

- Read the linked files first, top to bottom, before editing.
- Note one thing that surprised you in the commit message.
- If the step takes > M (1-3 days) estimate, stop and split it — open a follow-up note.
