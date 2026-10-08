# 053 --data-dir CLI override

| Field | Value |
|-------|-------|
| Status | `todo` (change to `doing` / `done` as you work) |
| Priority | P1 |
| Size | S (<1 day) |
| Area | config |
| Phase | Week 3 — backend reliability |
| Related | `055` |

## 1. Why this matters

`main.go:26-31` parses only `--version`. Power users / multi-profile setups (`055`) / CI (`094`) want `WayerPC.exe --data-dir D:\alt` without env vars. `ResolveDataRoot` (`config.go:142-150`) already honors env first — CLI should win over all.

Read this file, then touch the code. This issue is deliberately scoped so you can
finish it in one sitting and see the result in the running app.

## 2. Current behavior

Flags: `--version` only. Env `WAYERPC_DATA_DIR` overrides config (`config.go:143-145`); `.data/` stub is dev fallback (`config.go:153-163`). No `--data-dir`, no `--port`, no headless flags. Terminal use documented in `main.go:2-5` header.

Key files:

- `app.go` (Wails bridge) + `internal/` packages — see issue body for exact paths.

## 3. Problems / symptoms

- Testing a second library needs env juggling.
- CI smoke (`094`) can't pass an isolated dir cleanly.
- Support can't ask 'run with a fresh dir and send logs' in one command.

## 4. Proposal (what "good" looks like)

Add `--data-dir`, `--port` (bonus), `--version` stays. Precedence: CLI > env > config > stub. Log effective root at startup (`openRoot` already logs root — include source: cli/env/config). Document in BUILD.md + `--help`.

## 5. Step-by-step implementation

1. Add flags in `main.go`; pass into `App` (extend `NewApp` opts or setter before `startup`).
2. `ResolveDataRoot` gains CLI param (or App prefers CLI value first).
3. Startup log: `Data folder: X (from --data-dir)`.
4. Test: `--data-dir` temp dir creates fresh tree; `--help` lists flags.
5. BUILD.md documents + `094` uses it.

Work in small commits. After each step, run the relevant check:

```powershell
go build ./...; if ($?) { go test ./... }
cmd /c "pnpm.cmd --dir frontend build"
```

## 6. Acceptance criteria

- [ ] `WayerPC.exe --data-dir D:\alt` uses D:\alt with empty env/config.
- [ ] Precedence CLI > env > config verified in test.
- [ ] `--help` output documented.

## 7. Tests to add / update

Precedence unit tests on resolve func; manual CLI smoke.

## 8. Risks and rollback

Flag parsing vs Wails arg handling — keep flags stdlib-only, parse before `wails.Run`.

Rollback: `git stash` or revert the single commit for this issue. Nothing here
touches the wire protocol unless the header says *(breaking)* — and none of the
early-phase issues do.

## 9. Follow-ups (do NOT do in this issue)

`055` (profiles build on this), `094` (CI uses it).

## 10. Touch-and-learn notes

- Read the linked files first, top to bottom, before editing.
- Note one thing that surprised you in the commit message.
- If the step takes > S (<1 day) estimate, stop and split it — open a follow-up note.
