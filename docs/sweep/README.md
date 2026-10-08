# WayerPC hand-sweep — 100 single-case issues

One file = one sitting. Read numerically within each phase, flip `Status` to
`doing`/`done` as you work, commit as `sweep(NNN): <title>`, one issue per commit.

Start here: [`001-how-to-use-this-sweep.md`](001-how-to-use-this-sweep.md) →
[`002-roadmap-overview-and-order.md`](002-roadmap-overview-and-order.md) →
[`003-coding-conventions-for-this-sweep.md`](003-coding-conventions-for-this-sweep.md)

Progress: `Get-ChildItem docs/sweep/*.md | Select-String -Pattern '\| Status \| `done`' | Measure-Object`

## Phase map (month-or-two arc)

| Weeks | Files | Theme |
|-------|-------|-------|
| 1 | 001–003, 004–005, 022, 056, 091, 099 | Orientation, first protocol fixes, stop the CPU waste, copy-logs, conformance gate, changelog habit |
| 1–2 | 006–007, 011, 013–014, 057, 062, 068, 070 | Handshake/ping, conn cap, error taxonomy, toasts, shortcuts, empty states, installer EnVar fix |
| 3 | 015, 017–019, 021, 023, 029, 032, 036–037, 040, 052–054, 064, 066 | LAN IP, speed/ETA, port help, restart, timeouts, sqlite busy, bulk txn, prune, cancellable search + progress, symlink safety, host/port UI, CLI flag, config repair, dry-run, settings restart |
| 4 | 016, 024–025, 028, 034, 041, 044–046, 048, 050–051, 060, 063, 065 | QR, watcher, persisted logs, paging, ordering, filters, app-folder modes, history, preview, space checks, sorting, reveal, light theme, onboarding, import progress |
| 5 | 010, 020, 030–031, 035, 059, 061, 067, 069, 076–077, 079–080, 094 | Pairing auth, transfer limits, vacuum, backup, case-unique, thumbnails, i18n, a11y, phone panel, structured logs, rotation, traversal audit, trust model, CI smoke |
| 6 | 008–009, 012, 026, 033, 071–075, 081, 095–096, 098 | Resume + richer MATCHES (gated), upload cap, dedup, schema columns, updates/uninstall/signing/autostart/install-dir, Zig intro, 100k bench, unit coverage, QA checklist |
| 7 | 055, 082–087, 097 | Profiles, Zig ports 2–7 (scorer, walk, hash, ABI, build, Windows), frontend tests |
| 8 | 027, 042–043, 088–090, 100 | FTS5, content search + index (deferred designs), Zig review/bench/flags, 1.0 gate |

Deferred by design (read, spike, then decide): `042`, `043`. Biggest blast radius,
do last and carefully: `047` relocate safety (Week 4 — move it later if nervous).

## The Zig track (non-GC language plan)

After you know the Go code, files `081–090` port the **non-network** CPU parts to
Zig, incrementally, behind flags, Go staying default:

| # | File | Question it answers |
|---|------|---------------------|
| 1 | [081](081-zig-why-non-gc-for-backend.md) | Why Zig, what never ports (all network code stays Go) |
| 2 | [082](082-zig-what-to-port-first-scoring.md) | First target: similarity scorer (pure function) |
| 3 | [083](083-zig-file-walk-port-plan.md) | Second target: file-walk + filter |
| 4 | [084](084-zig-hash-and-checksum-port.md) | Third target: SHA-256 streaming |
| 5 | [085](085-zig-cabi-and-cgo-integration.md) | The one C-ABI/cgo pattern all ports share |
| 6 | [086](086-zig-build-system-integration.md) | One-command build stays; Zig is opt-in |
| 7 | [087](087-zig-cross-compile-windows.md) | windows/amd64 + long-path proof |
| 8 | [088](088-zig-memory-safety-review.md) | Leak/hostile-input review per port |
| 9 | [089](089-zig-benchmark-vs-go-parity.md) | Bench gates + parity harness (ship/stay verdicts) |
| 10 | [090](090-zig-fallback-and-feature-flag.md) | Runtime flags + Go fallback discipline |

Rule, signed in `081`: a `stay Go` verdict is a successful outcome, recorded with numbers.

## Soundness + tests (advanced stages)

"Sound" improvements are woven in, not bolted on: each issue's §7 names its tests
and §8 its rollback. The dedicated test spine is `091` (conformance — write first),
`092` (fuzz), `093` (mock parity), `094` (CI smoke), `095` (100k bench — before
`027`), `096`–`097` (unit/component coverage), `098` (30-min phone QA). If an
issue's estimate is exceeded, split it and note the split — the sweep survives
contact with reality that way.

## Full index

- 001 [How to use this sweep](001-how-to-use-this-sweep.md) — meta, P0
- 002 [Roadmap overview and order](002-roadmap-overview-and-order.md) — meta, P0
- 003 [Coding conventions](003-coding-conventions-for-this-sweep.md) — meta, P0
- 004 [READY newline](004-protocol-ready-newline.md) — protocol, P0
- 005 [DONE + SHA-256](005-protocol-done-sha256-checksum.md) — protocol, P0
- 006 [/hello handshake](006-protocol-hello-handshake.md) — protocol, P1
- 007 [/ping keepalive](007-protocol-ping-keepalive.md) — protocol, P1
- 008 [Upload resume](008-protocol-upload-resume-offset.md) — protocol, P1, gated proto ≥ 2
- 009 [Richer MATCHES](009-protocol-richer-matches.md) — protocol, P2, breaking
- 010 [Pairing code /auth](010-protocol-pairing-code-auth.md) — protocol/security, P2
- 011 [Per-IP conn cap](011-protocol-per-ip-connection-cap.md) — protocol, P2
- 012 [Configurable max upload](012-protocol-max-upload-configurable.md) — protocol, P2
- 013 [Error taxonomy](013-protocol-error-taxonomy.md) — protocol, P1
- 014 [Command limits + sanitization](014-protocol-command-limits-and-sanitization.md) — protocol/security, P1
- 015 [LAN IP display](015-server-lan-ip-display.md) — backend, P1
- 016 [QR-code connect](016-server-qr-code-connect.md) — backend/frontend, P1
- 017 [Speed + ETA](017-server-transfer-speed-eta.md) — backend/frontend, P1
- 018 [Port-in-use help](018-server-port-in-use-help.md) — backend/frontend, P1
- 019 [Graceful restart](019-server-graceful-restart.md) — backend, P1
- 020 [Concurrent transfer limits](020-server-concurrent-transfer-limits.md) — backend, P2
- 021 [Idle vs stall timeouts](021-server-idle-vs-stall-timeouts.md) — backend, P2
- 022 [DiskUsage cache](022-backend-diskusage-cache.md) — backend, P0
- 023 [SQLite busy-timeout](023-backend-sqlite-busy-timeout.md) — backend, P1
- 024 [shared/ watcher](024-backend-shared-auto-catalog-watcher.md) — backend, P2
- 025 [Persist logs jsonl](025-backend-persist-logs-jsonl.md) — backend, P2
- 026 [Content-hash dedup](026-backend-content-hash-dedup.md) — backend, P2
- 027 [FTS5 index](027-backend-fts5-catalog-index.md) — backend, P2, after 095
- 028 [Catalog paging](028-catalog-pagination-and-virtual-list.md) — backend/frontend, P2
- 029 [Bulk txn import](029-catalog-bulk-insert-transaction.md) — backend, P1
- 030 [Vacuum + integrity](030-catalog-vacuum-and-integrity.md) — backend, P2
- 031 [Backup + restore](031-catalog-backup-and-restore.md) — backend/frontend, P2
- 032 [Orphan prune](032-catalog-orphan-detection-prune.md) — backend/frontend, P1
- 033 [Hash/metadata columns](033-catalog-hash-size-columns.md) — backend, P2
- 034 [imported_at index](034-catalog-imported-at-index.md) — backend, P2
- 035 [Case-insensitive unique](035-catalog-case-insensitive-unique.md) — backend, P2
- 036 [Cancellable search](036-search-user-cancellation.md) — search, P1
- 037 [Search progress events](037-search-progress-events.md) — search/frontend, P1
- 038 [Search ranking](038-search-ranking-and-scoring.md) — search, P2
- 039 [Ignore-rules editor](039-search-ignore-rules-editor.md) — search/frontend, P2
- 040 [Symlink safety](040-search-symlink-loop-safety.md) — search, P1
- 041 [Extension + size filters](041-search-extension-and-size-filters.md) — search/frontend, P2
- 042 [Content search (deferred)](042-search-content-snippet-mode.md) — search, P2
- 043 [Indexed cache (deferred)](043-search-indexed-cache.md) — search/backend, P2
- 044 [App-folder search modes](044-search-app-folder-recursive-toggle.md) — search, P2
- 045 [History + saved queries](045-search-history-and-saved-queries.md) — search/frontend, P2
- 046 [Result preview pane](046-search-result-preview-pane.md) — search/frontend, P2
- 047 [Relocate safety](047-storage-relocate-safety.md) — storage, P1
- 048 [Free-space + FAT32](048-storage-freespace-fat32-check.md) — storage/frontend, P2
- 049 [Permission errors](049-storage-permissions-errors.md) — storage/frontend, P2
- 050 [Sorting modes](050-storage-listfiles-sorting-modes.md) — storage/frontend, P2
- 051 [Reveal fallbacks](051-storage-reveal-fallbacks.md) — storage/frontend, P2
- 052 [Host/port UI](052-config-host-port-ui.md) — config/frontend, P1
- 053 [--data-dir flag](053-config-cli-data-dir-override.md) — config, P1
- 054 [Config validation](054-config-validation-and-repair.md) — config, P1
- 055 [Profiles](055-config-multi-profile-support.md) — config/frontend, P2
- 056 [Copy/download logs](056-frontend-console-copy-logs.md) — frontend, P2
- 057 [Toast history](057-frontend-toast-history-drawer.md) — frontend, P2
- 058 [Double-click open](058-frontend-files-open-doubleclick.md) — frontend, P2
- 059 [Thumbnails](059-frontend-image-thumbnails.md) — frontend/backend, P2
- 060 [Light theme](060-frontend-theme-toggle-light.md) — frontend, P2
- 061 [i18n scaffolding](061-frontend-i18n-scaffolding.md) — frontend, P2
- 062 [Shortcuts overlay](062-frontend-shortcuts-overlay.md) — frontend, P2
- 063 [Onboarding wizard](063-frontend-onboarding-wizard.md) — frontend, P1
- 064 [Dry-run counts](064-frontend-import-dryrun-counts.md) — frontend/backend, P1
- 065 [Import progress bar](065-frontend-import-progress-bar.md) — frontend/backend, P1
- 066 [Settings restart](066-frontend-settings-server-restart.md) — frontend, P1
- 067 [a11y audit](067-frontend-accessibility-audit.md) — frontend, P2
- 068 [Empty states](068-frontend-empty-states-illustrations.md) — frontend, P2
- 069 [Phone panel](069-frontend-connected-phone-panel.md) — frontend/backend, P2
- 070 [Drop EnVar](070-installer-drop-envar-dependency.md) — installer, P0
- 071 [Update feed](071-installer-inapp-updates-feed.md) — installer/frontend, P1
- 072 [Uninstall choice](072-installer-uninstall-data-choice.md) — installer, P2
- 073 [Signing story](073-installer-code-signing-story.md) — installer, P2
- 074 [Autostart](074-installer-start-with-windows.md) — installer/frontend, P2
- 075 [Remember install dir](075-installer-remember-install-dir.md) — installer, P2
- 076 [Log levels](076-logging-structured-levels.md) — backend/frontend, P1
- 077 [Log rotation](077-logging-log-rotation-policy.md) — backend, P2
- 078 [Metrics snapshot](078-observability-metrics-endpoint.md) — backend, P2
- 079 [Traversal audit](079-security-path-traversal-audit.md) — security, P1
- 080 [Trust model](080-security-hotspot-trust-model.md) — security, P2
- 081–090 [Zig track](081-zig-why-non-gc-for-backend.md) — zig, see table above
- 091 [Conformance script](091-tests-protocol-conformance-script.md) — tests, P0
- 092 [Fuzz test](092-tests-framing-fuzz-test.md) — tests, P1
- 093 [Mock parity](093-tests-mock-parity-check.md) — tests/frontend, P1
- 094 [CI smoke](094-tests-headless-ci-smoke.md) — tests/ci, P1
- 095 [100k bench](095-tests-catalog-benchmark-100k.md) — tests/backend, P2
- 096 [Unit coverage](096-tests-backend-unit-coverage.md) — tests/backend, P1
- 097 [Frontend tests](097-tests-frontend-component-tests.md) — tests/frontend, P2
- 098 [QA checklist](098-tests-manual-qa-checklist.md) — tests, P1
- 099 [Changelog](099-changelog-and-release-notes.md) — release, P2, start now
- 100 [1.0 definition](100-final-polish-and-1-0-definition.md) — meta/release, P0
