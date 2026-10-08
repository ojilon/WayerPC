# 055 Multi-profile support (design + lite switcher)

| Field | Value |
|-------|-------|
| Status | `todo` (change to `doing` / `done` as you work) |
| Priority | P2 |
| Size | M (1-3 days) |
| Area | config/frontend |
| Phase | Week 7 — advanced |
| Related | `053` |

## 1. Why this matters

Work library vs home library vs test sandbox: today one `config.json` + one root. Users juggle env vars. A named-profile switcher (profile = {name, dataRoot, host, port}) turns `--data-dir` (`053`) into a real feature.

Read this file, then touch the code. This issue is deliberately scoped so you can
finish it in one sitting and see the result in the running app.

## 2. Current behavior

Single `Config` (`config.go:33-39`) at `config.json` (`config.go:64-70`). No profile concept. `053` CLI override is the only escape hatch. Installer writes one root.

Key files:

- `app.go` (Wails bridge) + `internal/` packages — see issue body for exact paths.

## 3. Problems / symptoms

- No clean separation of libraries.
- Test runs risk polluting the real catalog.
- Family PC sharing one Windows user can't split libraries.

## 4. Proposal (what "good" looks like)

LITE version (this issue): profiles stored as `profiles.json` (list) + Settings dropdown to switch (validates + reopens via `openRoot`, restarts server `019`). Full version (separate issue): per-profile ports, tray switcher. Keep CLI override winning over profiles (`053` precedence extended: CLI > profile > env > legacy).

## 5. Step-by-step implementation

1. `profiles.json` schema + load/save alongside `config.json` (migrate current config → 'Default' profile).
2. Settings profile row: list, add (pick root), switch, rename, delete (never delete data, only the profile).
3. Switch = validate (`054`) → `openRoot` → server restart → `catalog-changed`.
4. CLI `--profile NAME` (optional; `--data-dir` still wins).
5. Test: two profiles, switch, catalogs independent.

Work in small commits. After each step, run the relevant check:

```powershell
go build ./...; if ($?) { go test ./... }
cmd /c "pnpm.cmd --dir frontend build"
```

## 6. Acceptance criteria

- [ ] Two profiles with different roots switch cleanly, server follows.
- [ ] Legacy single config migrates to 'Default' untouched.
- [ ] Deleting a profile never deletes data (test asserts tree still exists).

## 7. Tests to add / update

Profile CRUD + switch integration tests on temp dirs.

## 8. Risks and rollback

Scope creep into per-profile everything — bound it: profiles own dataRoot+port only; everything else global.

Rollback: `git stash` or revert the single commit for this issue. Nothing here
touches the wire protocol unless the header says *(breaking)* — and none of the
early-phase issues do.

## 9. Follow-ups (do NOT do in this issue)

Tray quick-switch (future); per-profile auto-start (`074`).

## 10. Touch-and-learn notes

- Read the linked files first, top to bottom, before editing.
- Note one thing that surprised you in the commit message.
- If the step takes > M (1-3 days) estimate, stop and split it — open a follow-up note.
