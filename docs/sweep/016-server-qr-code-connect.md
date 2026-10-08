# 016 QR-code connect (IP + pairing code)

| Field | Value |
|-------|-------|
| Status | `todo` (change to `doing` / `done` as you work) |
| Priority | P1 |
| Size | M (1-3 days) |
| Area | backend/frontend |
| Phase | Week 4 — frontend depth |
| Related | `015`, `010` |

## 1. Why this matters

Typing `192.168.43.41:5000` + a 6-digit code is error-prone. A QR in Console that the phone scans removes the worst onboarding step.

Read this file, then touch the code. This issue is deliberately scoped so you can
finish it in one sitting and see the result in the running app.

## 2. Current behavior

No QR lib, no canvas/img path. Console hero (`console.ts:20-27`) has state+addr+uptime only. Pairing code doesn't exist yet (`010`).

Key files:

- `app.go` (Wails bridge) + `internal/` packages — see issue body for exact paths.

## 3. Problems / symptoms

- Manual transcription errors on small phone keyboards.
- No single 'connect payload' definition shared by PC + Android.
- QR needs a Go encoder + frontend render path — two halves, one issue if done naively.

## 4. Proposal (what "good" looks like)

Define payload `WAYERPC:1:<lan-ip>:<port>:<pairing?>`; Go generates PNG (pick `skip2/go-qrcode` or `yeqown/go-qrcode`); frontend renders `<img src=data:…>` in Console hero. Start with IP:port even before `010` lands; append code when available.

## 5. Step-by-step implementation

1. Choose QR lib (pure-Go, no cgo); add to `go.mod`.
2. Add `App.GetConnectQR()` returning base64 PNG + payload string.
3. Render in `console.ts` hero-right; add Copy-payload button.
4. Test: decode PNG in test (round-trip payload).
5. Android: scan → prefill host/port/code.
6. Fallback when no LAN IP (`015`): show payload text only.

Work in small commits. After each step, run the relevant check:

```powershell
go build ./...; if ($?) { go test ./... }
cmd /c "pnpm.cmd --dir frontend build"
```

## 6. Acceptance criteria

- [ ] Phone camera scan connects without typing.
- [ ] QR refreshes when IP/port/code changes.
- [ ] Works offline (no network fetch for lib).

## 7. Tests to add / update

QR round-trip test; frontend preview via `dev-mock.ts`.

## 8. Risks and rollback

Dependency weight — prefer tiny QR lib. Large payload (IPv6) → denser QR; keep payload minimal.

Rollback: `git stash` or revert the single commit for this issue. Nothing here
touches the wire protocol unless the header says *(breaking)* — and none of the
early-phase issues do.

## 9. Follow-ups (do NOT do in this issue)

`010` (code in QR), Android scanner issue.

## 10. Touch-and-learn notes

- Read the linked files first, top to bottom, before editing.
- Note one thing that surprised you in the commit message.
- If the step takes > M (1-3 days) estimate, stop and split it — open a follow-up note.
