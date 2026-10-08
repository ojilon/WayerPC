# 073 Code-signing story (SmartScreen)

| Field | Value |
|-------|-------|
| Status | `todo` (change to `doing` / `done` as you work) |
| Priority | P2 |
| Size | S (<1 day) |
| Area | installer |
| Phase | Week 6 — release |
| Related | `099` |

## 1. Why this matters

First public setup.exe without a signing story scares users (SmartScreen 'Unknown publisher' → abandon). Document self-signed vs cert vs none BEFORE publishing, not after the first report.

Read this file, then touch the code. This issue is deliberately scoped so you can
finish it in one sitting and see the result in the running app.

## 2. Current behavior

No signing in `build-installer.ps1` (verify); `wails.json` product info static. No docs on SmartScreen, no cert funding plan, no hash-publishing habit.

Key files:

- `app.go` (Wails bridge) + `internal/` packages — see issue body for exact paths.

## 3. Problems / symptoms

- Surprise warnings erode trust in a hotspot-network tool (ironic for `080`).
- No reproducible-hash story (did this exe come from this tag?).
- Release checklist (`099`) lacks the signing step.

## 4. Proposal (what "good" looks like)

Doc-only first (this issue): options table (uncertified + SmartScreen reality, self-signed limits, OV cert cost/process, CI sign step sketch), decision for v1 (recommend: publish SHA-256 + VirusTotal link + SmartScreen screenshot walkthrough; defer paid cert with cost noted), release-checklist entry. No cert purchase in this issue.

## 5. Step-by-step implementation

1. Research current SmartScreen behavior for unsigned NSIS exes (2026 state).
2. Write `docs/sweep` decision note + BUILD.md release step (hash publish, notes link).
3. Add hash-publish to `099` checklist (sha256sum of setup.exe per release).
4. Optional: self-sign spike in CI (proves pipeline, not trust) — timebox 2h.
5. Record cert decision + revisit version.

Work in small commits. After each step, run the relevant check:

```powershell
go build ./...; if ($?) { go test ./... }
cmd /c "pnpm.cmd --dir frontend build"
```

## 6. Acceptance criteria

- [ ] BUILD.md has a signing section with the v1 decision.
- [ ] Release checklist includes hash publish.
- [ ] No code change needed (doc-only allowed to close).

## 7. Tests to add / update

None (doc). Verify links/commands by running them once.

## 8. Risks and rollback

Advice rots (Microsoft changes prompts) — date-stamp the doc, revisit yearly.

Rollback: `git stash` or revert the single commit for this issue. Nothing here
touches the wire protocol unless the header says *(breaking)* — and none of the
early-phase issues do.

## 9. Follow-ups (do NOT do in this issue)

Paid cert (future budget decision); CI signing job if cert acquired.

## 10. Touch-and-learn notes

- Read the linked files first, top to bottom, before editing.
- Note one thing that surprised you in the commit message.
- If the step takes > S (<1 day) estimate, stop and split it — open a follow-up note.
