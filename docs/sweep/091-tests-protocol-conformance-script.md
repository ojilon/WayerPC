# 091 Protocol conformance script (the gate for all protocol work)

| Field | Value |
|-------|-------|
| Status | `todo` (change to `doing` / `done` as you work) |
| Priority | P0 |
| Size | M (1-3 days) |
| Area | tests |
| Phase | Week 1 — tests first |
| Related | all protocol files |

## 1. Why this matters

The two field bugs (absolute-deadline kill, newline-less framing) would both have been caught by one script driving FOUND/MATCHES/READY+bytes/DONE/ERROR paths — including coalesced bursts and newline-less headers — against any build. Write it BEFORE changing any protocol issue (`004-011`).

Read this file, then touch the code. This issue is deliberately scoped so you can
finish it in one sitting and see the result in the running app.

## 2. Current behavior

Go tests exist (`server_test.go`) but no end-to-end socket script runnable against the real exe with `.data/`; no coalesced-burst case; no newline-less header case in CI. `SUGGESTIONS-v0.2.1.md` §5 marks P0/M.

Key files:

- `app.go` (Wails bridge) + `internal/` packages — see issue body for exact paths.

## 3. Problems / symptoms

- Protocol changes (`004`-`011`) unverifiable without the APK.
- Regression risk: fixing READY framing could break FOUND coalescing.
- CI (`094`) has nothing to drive the TCP surface.

## 4. Proposal (what "good" looks like)

Single script (Python stdlib socket, no deps): cases = ask-found ( indivisual + coalesced FOUND+bytes burst), ask-matches, ask-missing, upload-bytes (newline-less header!), upload-query, unknown-cmd, malformed sizes, READY-newline assert (`004`), DONE-sha assert (`005` when landed, tolerant before). Runnable `python conformance.py --addr 127.0.0.1:5000 --data .data`. CI runs it against a test build (`094`).

## 5. Step-by-step implementation

1. Write `tests/conformance.py` (stdlib only) with case list + strict asserts + verbose log.
2. Cover: newline + newline-less variants of every command; coalesced burst (send no delay, read all); split-byte header (1-byte writes).
3. Seed `.data/` fixture (received/ + catalog rows) in-script or via `--seed`.
4. Run against current build → record baseline (all green before any protocol change).
5. Wire into `094` CI smoke (fail on red).
6. Update per protocol change (`004` READY assert tightens, `005` DONE-sha, `006` hello).

Work in small commits. After each step, run the relevant check:

```powershell
go build ./...; if ($?) { go test ./... }
cmd /c "pnpm.cmd --dir frontend build"
```

## 6. Acceptance criteria

- [ ] Script green on current build (baseline commit).
- [ ] Script catches the two historical bugs (verify by reverting fixes locally — document).
- [ ] CI runs it (or has a tracked TODO with owner + date if CI not yet).

## 7. Tests to add / update

The script IS the test. Plus a self-test (script vs stub server).

## 8. Risks and rollback

Flakiness on loaded CI (timing asserts) — assert protocol bytes, not timings; retries only for bind-wait.

Rollback: `git stash` or revert the single commit for this issue. Nothing here
touches the wire protocol unless the header says *(breaking)* — and none of the
early-phase issues do.

## 9. Follow-ups (do NOT do in this issue)

Every protocol file lists its conformance case here; `092` fuzzes beneath it.

## 10. Touch-and-learn notes

- Read the linked files first, top to bottom, before editing.
- Note one thing that surprised you in the commit message.
- If the step takes > M (1-3 days) estimate, stop and split it — open a follow-up note.
