# 094 Headless CI smoke (Windows runner)

| Field | Value |
|-------|-------|
| Status | `todo` (change to `doing` / `done` as you work) |
| Priority | P1 |
| Size | M (1-3 days) |
| Area | tests/ci |
| Phase | Week 5 — tests/CI |
| Related | `091`, `093`, `053` |

## 1. Why this matters

Packaging-only breakage (embed, icons, manifest), mock drift, and protocol regressions currently need a human + a phone to catch. A Windows runner doing pnpm build → wails build → `--version` → conformance (`091`) against the real exe with `.data/` + `--data-dir` (`053`) isolation catches all three without the APK.

Read this file, then touch the code. This issue is deliberately scoped so you can
finish it in one sitting and see the result in the running app.

## 2. Current behavior

No CI config (verify: no `.github/workflows`? check). Version compare + download manual. `053` CLI flag may not exist yet — CI needs it (order: `053` before or with this). `BUILD.md` has commands but no automation.

Key files:

- `app.go` (Wails bridge) + `internal/` packages — see issue body for exact paths.

## 3. Problems / symptoms

- 'Works on my machine' releases (missing embed, stale frontend/dist).
- No gate for `070` installer compile, `093` mock parity, `091` conformance.
- Manual release checklist (`099`) unenforced.

## 4. Proposal (what "good" looks like)

Minimal GitHub Actions `windows-latest`: pnpm install+build, `wails build`, run `WayerPC.exe --version`, boot with `WAYERPC_DATA_DIR` temp (or `--data-dir` `053`), run `091` conformance + `093` mock check + `go test ./...`. Installer compile (`070`-fixed NSIS) as a second job (advisory first). Cache pnpm + Go modules.

## 5. Step-by-step implementation

1. Check for existing workflows; create `.github/workflows/smoke.yml` minimal.
2. Use `--data-dir` temp isolation (`053`); seed `.data/` stub.
3. Steps: build frontend → `go test` → wails build → `--version` assert → boot → conformance → mock check.
4. Artifacts: upload exe + conformance log on failure.
5. Document badge + how to run the same locally (BUILD.md).
6. Installer job: NSIS portable compile check (after `070`).

Work in small commits. After each step, run the relevant check:

```powershell
go build ./...; if ($?) { go test ./... }
cmd /c "pnpm.cmd --dir frontend build"
```

## 6. Acceptance criteria

- [ ] Green run on a PR that changes only a comment (proves infra, not luck).
- [ ] Intentionally-broken protocol change fails conformance in CI (prove the gate bites).
- [ ] Local repro commands documented and verified.

## 7. Tests to add / update

CI *is* the test. Keep workflow <100 lines; no matrix sprawl yet.

## 8. Risks and rollback

Runner minutes + flaky sleeps (server boot wait) — poll for port-ready, don't sleep-fixed. Wails build needs WebView2 SDK on runner — verify image has it.

Rollback: `git stash` or revert the single commit for this issue. Nothing here
touches the wire protocol unless the header says *(breaking)* — and none of the
early-phase issues do.

## 9. Follow-ups (do NOT do in this issue)

`070` (installer job hardens), `099` (release uses the CI exe).

## 10. Touch-and-learn notes

- Read the linked files first, top to bottom, before editing.
- Note one thing that surprised you in the commit message.
- If the step takes > M (1-3 days) estimate, stop and split it — open a follow-up note.
