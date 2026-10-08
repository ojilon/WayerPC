# 012 Configurable max upload size

| Field | Value |
|-------|-------|
| Status | `todo` (change to `doing` / `done` as you work) |
| Priority | P2 |
| Size | S (<1 day) |
| Area | protocol |
| Phase | Week 4 — storage+config |
| Related | `052`, `048` |

## 1. Why this matters

The 2GiB cap (`DefaultMaxUploadBytes = 2<<30`, `server.go:57`) is a const. Users with FAT32 destinations (4GB file limit minus overhead) or tiny disks need a knob; power users moving videos need more.

Read this file, then touch the code. This issue is deliberately scoped so you can
finish it in one sitting and see the result in the running app.

## 2. Current behavior

`SetMaxUploadBytes` exists but only tests call it. No UI, no config key, no `ERROR upload_too_large` guidance beyond the code (`protocol.go:120-123`).

Key files:

- `app.go` (Wails bridge) + `internal/` packages — see issue body for exact paths.

## 3. Problems / symptoms

- One-size cap with no explanation in Settings.
- FAT32 users hit filesystem errors instead of a clean protocol ERROR.
- No per-destination adjustment after relocate (`047`).

## 4. Proposal (what "good" looks like)

Add `max_upload_bytes` to config + Settings field (bytes/MB) with validation; surface the effective cap in Console; keep 2GiB default. `ERROR upload_too_large` stays the wire signal.

## 5. Step-by-step implementation

1. Add `MaxUploadBytes` to `config.Config` + load/save + validation.
2. Wire through `app.go:openRoot` → `server.New` (or a setter on attach).
3. Settings UI field + Console hint (`015` area).
4. Test: set 1MB cap, upload 2MB → clean ERROR.
5. Document FAT32 interplay (`048`).

Work in small commits. After each step, run the relevant check:

```powershell
go build ./...; if ($?) { go test ./... }
cmd /c "pnpm.cmd --dir frontend build"
```

## 6. Acceptance criteria

- [ ] Cap editable in Settings, persisted across restarts.
- [ ] Oversize upload fails fast with `ERROR upload_too_large` before any bytes flow.
- [ ] Default stays 2GiB when unset.

## 7. Tests to add / update

Config round-trip test + oversize-upload test.

## 8. Risks and rollback

Setting cap above filesystem/disk realities gives false promise — cross-link `048` free-space check.

Rollback: `git stash` or revert the single commit for this issue. Nothing here
touches the wire protocol unless the header says *(breaking)* — and none of the
early-phase issues do.

## 9. Follow-ups (do NOT do in this issue)

`048`, `052`.

## 10. Touch-and-learn notes

- Read the linked files first, top to bottom, before editing.
- Note one thing that surprised you in the commit message.
- If the step takes > S (<1 day) estimate, stop and split it — open a follow-up note.
