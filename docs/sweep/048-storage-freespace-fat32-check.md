# 048 Free-space + filesystem checks (FAT32 warning)

| Field | Value |
|-------|-------|
| Status | `todo` (change to `doing` / `done` as you work) |
| Priority | P2 |
| Size | S (<1 day) |
| Area | storage/frontend |
| Phase | Week 4 — storage+config |
| Related | `047`, `012` |

## 1. Why this matters

Before adopting a data root (Settings onboarding, `settings.ts:54-60`) nobody checks free space or filesystem. FAT32 (4GB file cap, common on USB sticks) + a 5GB video upload = cryptic mid-transfer `write_failed`. Warn before, not after.

Read this file, then touch the code. This issue is deliberately scoped so you can
finish it in one sitting and see the result in the running app.

## 2. Current behavior

No space query anywhere; `Init` (`storage.go:37-48`) `MkdirAll`s blindly. Upload cap is fixed 2GiB (`012`) with no FS awareness. Onboarding copy says 'any drive or subfolder works' — mostly true, expensively false on full disks.

Key files:

- `app.go` (Wails bridge) + `internal/` packages — see issue body for exact paths.

## 3. Problems / symptoms

- Onboarding onto a nearly-full drive guarantees later failures.
- FAT32 4GB limit discovered mid-upload.
- No 'needs X, has Y' message anywhere.

## 4. Proposal (what "good" looks like)

Add `FreeSpace(path)` (syscall: `GetDiskFreeSpaceExW` / `statvfs`) + `FsType` where cheap; Settings shows 'D: has 41 GB free (NTFS)' before confirm; warn (not block) on FAT32/removable/<5GB free; upload path (`012` cap) cross-checks file size vs fs limits for a clean early ERROR.

## 5. Step-by-step implementation

1. Implement `FreeSpace` per-OS (Windows + linux/darwin statvfs) with graceful fallback ('unknown').
2. Settings data-root picker: show free space + fs type in the confirm dialog.
3. Warn thresholds: FAT32/exFAT-removable note, <5GB caution, <500MB block.
4. Upload pre-check: if `filesize` exceeds fs file-size limit → `ERROR upload_too_large` early with reason.
5. Test with mocked free-space func (injectable).

Work in small commits. After each step, run the relevant check:

```powershell
go build ./...; if ($?) { go test ./... }
cmd /c "pnpm.cmd --dir frontend build"
```

## 6. Acceptance criteria

- [ ] Onboarding onto full-drive fixture blocks with numbers.
- [ ] FAT32 shows an explicit warning (manual verify on a USB stick).
- [ ] Unknown-fs degrades to 'could not determine' (never crashes).

## 7. Tests to add / update

Threshold unit tests with fake free-space values.

## 8. Risks and rollback

Fs-type detection varies by OS — keep warnings heuristic, blocks only on bytes-free numbers.

Rollback: `git stash` or revert the single commit for this issue. Nothing here
touches the wire protocol unless the header says *(breaking)* — and none of the
early-phase issues do.

## 9. Follow-ups (do NOT do in this issue)

`047` (preflight uses this), `012` (cap interplay).

## 10. Touch-and-learn notes

- Read the linked files first, top to bottom, before editing.
- Note one thing that surprised you in the commit message.
- If the step takes > S (<1 day) estimate, stop and split it — open a follow-up note.
