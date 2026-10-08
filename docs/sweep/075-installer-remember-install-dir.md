# 075 Remember install dir across reinstalls

| Field | Value |
|-------|-------|
| Status | `todo` (change to `doing` / `done` as you work) |
| Priority | P2 |
| Size | S (<1 day) |
| Area | installer |
| Phase | Week 6 — release |
| Related | `070` |

## 1. Why this matters

`InstallDirRegKey` already exists (verify in `installer.nsi`) — but nobody verified it pre-fills on reinstall. Users who picked `D:\apps\WayerPC` get dumped back to default on update and end up with two installs.

Read this file, then touch the code. This issue is deliberately scoped so you can
finish it in one sitting and see the result in the running app.

## 2. Current behavior

Registry key claimed; behavior unverified. `GetStorageInfo.installDir` (`app.go:428`) surfaces it, but the NSIS `InstallDirRegKey` wiring + `InstallDir` default order needs reading to confirm.

Key files:

- `app.go` (Wails bridge) + `internal/` packages — see issue body for exact paths.

## 3. Problems / symptoms

- Update installs to a second folder; shortcuts/PATH point at the new one, data stays with the old.
- `070` PATH work multiplies the confusion (which bin is on PATH?).
- No release-note line telling users reinstalls keep their folder.

## 4. Proposal (what "good" looks like)

Verify + fix: read the key, pre-fill `InstallDir`, test install→custom→reinstall flow in a VM, note in release notes. Small, but prevents real support tickets.

## 5. Step-by-step implementation

1. Read `installer.nsi` InstallDir/registry ordering fully.
2. VM test: install default → reinstall (keeps?) + install custom → reinstall (prefills custom?).
3. Fix ordering if broken (key read before directory page).
4. PATH (`070`) must follow the remembered dir (verify both entries don't linger).
5. Note behavior in BUILD.md + `099` notes.

Work in small commits. After each step, run the relevant check:

```powershell
go build ./...; if ($?) { go test ./... }
cmd /c "pnpm.cmd --dir frontend build"
```

## 6. Acceptance criteria

- [ ] Custom-dir reinstall prefills custom (VM screenshots/proof in commit).
- [ ] No duplicate installs after update flow.
- [ ] PATH tracks the remembered dir.

## 7. Tests to add / update

Manual VM matrix; no unit tests (NSIS).

## 8. Risks and rollback

Registry redirection (32/64-bit views) — use the same view consistently; verify on 64-bit Windows.

Rollback: `git stash` or revert the single commit for this issue. Nothing here
touches the wire protocol unless the header says *(breaking)* — and none of the
early-phase issues do.

## 9. Follow-ups (do NOT do in this issue)

None — close and note.

## 10. Touch-and-learn notes

- Read the linked files first, top to bottom, before editing.
- Note one thing that surprised you in the commit message.
- If the step takes > S (<1 day) estimate, stop and split it — open a follow-up note.
