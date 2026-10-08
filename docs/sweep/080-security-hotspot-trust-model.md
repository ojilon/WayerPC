# 080 Hotspot trust-model document

| Field | Value |
|-------|-------|
| Status | `todo` (change to `doing` / `done` as you work) |
| Priority | P2 |
| Size | S (<1 day) |
| Area | security |
| Phase | Week 5 — hardening |
| Related | `010`, `079` |

## 1. Why this matters

Every security decision (open port, no auth, LAN IPs in QR) assumes 'hotspot = trusted-ish but observable'. That assumption is nowhere written. New contributors (and you in month 2) need the threat model to judge `010` pairing, `078` no-listener, `071` update checks.

Read this file, then touch the code. This issue is deliberately scoped so you can
finish it in one sitting and see the result in the running app.

## 2. Current behavior

Scattered hints: README 'same hotspot', `010` 'hotspots are open networks', `SUGGESTIONS` §6 rejects cloud relay. No consolidated doc, no attacker list, no explicit non-goals.

Key files:

- `app.go` (Wails bridge) + `internal/` packages — see issue body for exact paths.

## 3. Problems / symptoms

- Auth debates (`010`) lack a shared baseline.
- QR-over-shoulder, Evil-Twin-hotspot, port-scan-neighbor threats unranked.
- Future 'add cloud relay' requests can't be answered by pointer.

## 4. Proposal (what "good" looks like)

One-page doc: assets (files, catalog paths), actors (owner phone, guest phone, hotspot neighbor, evil-twin AP), assumptions (WPA2 hotspot w/ password, physical proximity), non-goals (no cloud, no multi-user ACL), residual risks + mitigations table (→`010`,`011`,`079`). Date-stamp; revisit yearly.

## 5. Step-by-step implementation

1. Draft the page (steal structure from OWASP threat-model-lite).
2. Map each residual to an issue (`010` pairing, `011` caps, `079` audit).
3. State the QR-shoulder-surfing + evil-twin posture explicitly.
4. Review against Android repo assumptions (same model both sides).
5. Link from README Security section (add the section if missing).

Work in small commits. After each step, run the relevant check:

```powershell
go build ./...; if ($?) { go test ./... }
cmd /c "pnpm.cmd --dir frontend build"
```

## 6. Acceptance criteria

- [ ] Doc merged + README-linked.
- [ ] Every 'why no X?' (cloud, accounts, E2E) answered by pointer.
- [ ] Dated + review-date set.

## 7. Tests to add / update

None (doc). Review = one careful read-through against the code's actual exposures.

## 8. Risks and rollback

Overclaiming ('secure') — use residual-risk language throughout.

Rollback: `git stash` or revert the single commit for this issue. Nothing here
touches the wire protocol unless the header says *(breaking)* — and none of the
early-phase issues do.

## 9. Follow-ups (do NOT do in this issue)

`010` (implements the top mitigation), yearly review reminder.

## 10. Touch-and-learn notes

- Read the linked files first, top to bottom, before editing.
- Note one thing that surprised you in the commit message.
- If the step takes > S (<1 day) estimate, stop and split it — open a follow-up note.
