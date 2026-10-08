# 070 Drop the EnVar plugin dependency (installer)

| Field | Value |
|-------|-------|
| Status | `todo` (change to `doing` / `done` as you work) |
| Priority | P0 |
| Size | S (<1 day) |
| Area | installer |
| Phase | Week 2 — installer |
| Related | `075` |

## 1. Why this matters

`installer.nsi` uses `EnVar::AddValue/DeleteValue`, NOT in stock NSIS 3 — the installer fails to compile on a fresh NSIS install. This blocks every setup.exe. Do before cutting any release.

Read this file, then touch the code. This issue is deliberately scoped so you can
finish it in one sitting and see the result in the running app.

## 2. Current behavior

`scripts/build-installer.ps1` drives NSIS; `packaging/` holds the script (verify exact filename with glob). `SUGGESTIONS-v0.2.1.md` §4 marks P0/S with the fix: `WriteRegExpandStr HKCU Environment` + `StrStr` guard (both stock).

Key files:

- `app.go` (Wails bridge) + `internal/` packages — see issue body for exact paths.

## 3. Problems / symptoms

- Fresh-machine builds fail; only the author's cached plugins work.
- PATH edit via plugin is opaque (no idempotence proof).
- Uninstall PATH cleanup tied to the same plugin.

## 4. Proposal (what "good" looks like)

Replace EnVar calls with registry writes: read `HKCU\Environment\Path`, `StrStr` guard for `bin`, append once, `SendMessage HWND_BROADCAST WM_SETTINGCHANGE`. Uninstall removes exactly our segment. Verify on a clean Windows VM (or fresh NSIS portable).

## 5. Step-by-step implementation

1. Read `installer.nsi` fully; locate every `EnVar::` call.
2. Implement stock-NSIS PATH add/remove functions (registry + broadcast).
3. Guard duplicates (install twice → one entry).
4. Test: clean NSIS portable build → install → PATH has bin → uninstall → PATH clean.
5. Document NSIS version + plugin-free claim in BUILD.md.

Work in small commits. After each step, run the relevant check:

```powershell
go build ./...; if ($?) { go test ./... }
cmd /c "pnpm.cmd --dir frontend build"
```

## 6. Acceptance criteria

- [ ] Fresh NSIS 3 (no extra plugins) compiles the installer.
- [ ] Double-install yields one PATH entry.
- [ ] Uninstall removes the entry, keeps the rest of PATH intact.

## 7. Tests to add / update

Manual VM matrix (clean install/uninstall/reinstall); script asserts in CI later (`094`).

## 8. Risks and rollback

Registry PATH length limits (~1024/2048 legacy) — check length, warn instead of truncating.

Rollback: `git stash` or revert the single commit for this issue. Nothing here
touches the wire protocol unless the header says *(breaking)* — and none of the
early-phase issues do.

## 9. Follow-ups (do NOT do in this issue)

`075` (remember-dir uses registry too — share helpers).

## 10. Touch-and-learn notes

- Read the linked files first, top to bottom, before editing.
- Note one thing that surprised you in the commit message.
- If the step takes > S (<1 day) estimate, stop and split it — open a follow-up note.
