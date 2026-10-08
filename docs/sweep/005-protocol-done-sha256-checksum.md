# 005 DONE with SHA-256 checksum

| Field | Value |
|-------|-------|
| Status | `todo` (change to `doing` / `done` as you work) |
| Priority | P0 |
| Size | S (<1 day) |
| Area | protocol |
| Phase | Week 1 — protocol correctness |
| Related | `004`, `091`, `026` |

## 1. Why this matters

Hotspot transfers corrupt more often than LAN. Today `DONE` carries no integrity signal (`protocol.go:169` → `sendLine(conn, "DONE")`). A mismatch is silent until the user opens a broken file.

Read this file, then touch the code. This issue is deliberately scoped so you can
finish it in one sitting and see the result in the running app.

## 2. Current behavior

`recvExact` (`transfer.go:63-96`) writes `.part` then renames; `addReceived` counts bytes. No hash is computed. `sendFile` (`transfer.go:23-58`) streams without checksumming. `SUGGESTIONS-v0.2.1.md` §1 rates this P0/S.

Key files:

- `app.go` (Wails bridge) + `internal/` packages — see issue body for exact paths.

## 3. Problems / symptoms

- Silent corruption on flaky hotspots.
- No way for the phone to know it should re-send.
- Re-uploads today restart from zero with no diagnosis.

## 4. Proposal (what "good" looks like)

Server sends `DONE <sha256hex>` of the received file (hash during write or one pass after rename). Phone verifies and re-sends on mismatch. Keep backward compat: phone that ignores the token still sees `DONE` prefix; server accepts phones that don't verify.

## 5. Step-by-step implementation

1. Add `crypto/sha256` hashing to the upload path: hash while writing chunks in `recvExact` (preferred: single pass) or `os.Open`+hash after rename.
2. Change reply to `sendLine(conn, "DONE "+hexsum)`.
3. Parse-tolerant client guidance: `DONE` or `DONE <hex64>` both mean success.
4. Unit test with known content (`sha256('abc')` fixture) through `handleUpload`.
5. Update protocol docs (`README.md` Protocol section + `doc/`).
6. Coordinate with Android repo: verify + re-send prompt on mismatch.

Work in small commits. After each step, run the relevant check:

```powershell
go build ./...; if ($?) { go test ./... }
cmd /c "pnpm.cmd --dir frontend build"
```

## 6. Acceptance criteria

- [ ] `DONE <64-hex>` observed for a test upload.
- [ ] Corrupted-delivery simulation (bit-flip proxy) triggers mismatch path.
- [ ] Old client (ignoring token) still completes upload.

## 7. Tests to add / update

`TestUploadDoneCarriesSha256` (in-memory conn pair); conformance script asserts hex format.

## 8. Risks and rollback

Extra hash pass costs CPU on huge files — hash-during-write avoids the second read. Very large uploads on slow disks: measure.

Rollback: `git stash` or revert the single commit for this issue. Nothing here
touches the wire protocol unless the header says *(breaking)* — and none of the
early-phase issues do.

## 9. Follow-ups (do NOT do in this issue)

`008` (resume — checksum per chunk), `026` (content-hash dedup reuses this hash).

## 10. Touch-and-learn notes

- Read the linked files first, top to bottom, before editing.
- Note one thing that surprised you in the commit message.
- If the step takes > S (<1 day) estimate, stop and split it — open a follow-up note.
