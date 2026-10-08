# 004 Terminate every reply with newline (including READY)

| Field | Value |
|-------|-------|
| Status | `todo` (change to `doing` / `done` as you work) |
| Priority | P0 |
| Size | S (<1 day) |
| Area | protocol |
| Phase | Week 1 — protocol correctness |
| Related | `091`, `092`, `006` |

## 1. Why this matters

`READY` is the only bare token (`internal/server/protocol.go:147` uses `fmt.Fprint(conn, "READY")` with no `\n`). Every other reply goes through `sendLine` (`transfer.go:13-17`). Clients need two read paths; the Android client trims one `read()` today but any future client needs a line reader.

Read this file, then touch the code. This issue is deliberately scoped so you can
finish it in one sitting and see the result in the running app.

## 2. Current behavior

`handleUpload` writes `READY` bare, then `DONE\n` via `sendLine`. `readCommand` (`server.go:275-298`) already tolerates newline-less input with a 500ms settle window — the asymmetry is server→client only. Field symptom: silent phone↔PC stalls blamed on hotspot were partly framing.

Key files:

- `app.go` (Wails bridge) + `internal/` packages — see issue body for exact paths.

## 3. Problems / symptoms

- Special-case client code for one token.
- Packet coalescing (`FOUND <size>` + first bytes in one segment) already caused a field bug; bare `READY` invites the same class.
- Cannot use a single line-reader on the client.

## 4. Proposal (what "good" looks like)

Send `READY\n` via `sendLine`. Verify the current APK tolerates trailing newline (it trims a single read — confirm against `ojilon/Wayer` client code or a live device) before release. Keep parsing tolerant: clients should accept `READY` with or without `\n` forever.

## 5. Step-by-step implementation

1. In `protocol.go:147` replace `fmt.Fprint(conn, "READY")` with `sendLine(conn, "READY")`.
2. Check the Android client read path (repo `ojilon/Wayer`) for `trim()` / line-reader tolerance.
3. Extend `internal/server/server_test.go` with a raw-socket test asserting the READY bytes end in `\n`.
4. Run the conformance script from `091` (write it first if missing) against the new build.
5. Test with the real APK on hotspot: upload a 1MB and a 200MB file.
6. Bump nothing yet — this stays compatible, no proto bump needed if the APK tolerates it.

Work in small commits. After each step, run the relevant check:

```powershell
go build ./...; if ($?) { go test ./... }
cmd /c "pnpm.cmd --dir frontend build"
```

## 6. Acceptance criteria

- [ ] Wireshark/raw log shows `READY\n`.
- [ ] Existing APK uploads succeed unmodified.
- [ ] New test fails on old code, passes on new code.

## 7. Tests to add / update

Add `TestReadyTerminatedByNewline` in `server_test.go`; extend `091` conformance with a READY-framing assertion.

## 8. Risks and rollback

Old third-party clients doing exact `== \"READY\"` match would break. Mitigation: verify against the only known client (your APK); document tolerance in protocol doc.

Rollback: `git stash` or revert the single commit for this issue. Nothing here
touches the wire protocol unless the header says *(breaking)* — and none of the
early-phase issues do.

## 9. Follow-ups (do NOT do in this issue)

`005` (checksum), `006` (hello handshake that advertises this fix).

## 10. Touch-and-learn notes

- Read the linked files first, top to bottom, before editing.
- Note one thing that surprised you in the commit message.
- If the step takes > S (<1 day) estimate, stop and split it — open a follow-up note.
