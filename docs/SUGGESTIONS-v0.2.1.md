# Suggestions for v0.2.1 (proposed — not scheduled)

Author: codebase agent, after the Go+Wails rewrite + two phone-interop
hotfixes (absolute-deadline kill, newline-less framing). Each item has a
rough size (S < 1 day, M 1–3 days, L bigger) and a priority (P0 must,
P1 should, P2 nice). Nothing here is committed to — it is a menu for
prioritization. Items marked *(breaking)* need a protocol-version bump and
a coordinated Android release.

## Theme proposal for v0.2.1: reliability + phone UX

v0.2.0 proved the stack (Go backend, frameless UI, installer). The failure
mode we actually observed in the field was **silent phone↔PC stalls on the
hotspot**. v0.2.1 should make transfers provably correct and failures loud.

## 1. Protocol robustness (phone interop)

- [ ] **P0/S — Terminate every reply with `\n` (including `READY`).**
  `READY` is currently the only bare token, which forces clients into a
  special-case read. Appending `\n` stays compatible with the existing
  Android client (it trims a single `read()`), and lets all future clients
  use one line-reader. Verify against the current APK before release.
- [ ] **P0/S — `DONE <sha256>` checksum.** Hotspot transfers corrupt more
  often than LAN ones. Server appends the SHA-256 of the received file to
  `DONE`; the phone verifies and re-sends on mismatch. Small change,
  kills an entire class of silent corruption.
- [ ] **P1/M — `/hello` handshake.** Client opens with
  `/hello <client-version>` → server answers
  `WAYERPC <server-version> <proto>`. Version mismatches surface as a clear
  message instead of cryptic mid-transfer failures. *(Minor protocol
  addition, backward compatible — unknown today, answered tomorrow.)*
- [ ] **P1/S — `/ping` keepalive.** Lets the phone probe a reused/stale
  socket before committing to a 200 MB upload, instead of discovering a
  dead peer mid-transfer.
- [ ] **P1/M — Upload resume.** `READY` → client sends from an offset
  (`/upload <size> <name> <offset>` + server appends to `.part`). Flaky
  hotspots currently restart hundred-MB uploads from zero. *(breaking-ish:
  gate behind proto ≥ 2.)*
- [ ] **P2/M — Richer `MATCHES`.** Today names-only; consider
  `name\tbytes\tmodified` so the phone can show sizes/dates. Needs the
  Android list UI to parse it — coordinate. *(breaking.)*
- [ ] **P2/S — Pairing code.** Hotspots are open networks: anyone joined can
  currently push files. Show a 6-digit code in Console; phone sends
  `/auth <code>` once per install. Big trust win, small protocol cost.
  *(breaking.)*
- [ ] **P2/S — Per-IP connection cap.** A neighbor port-scanning the hotspot
  currently spawns unbounded goroutines. Cap concurrent conns per IP (e.g.
  8) with an `ERROR busy` reply.

## 2. Backend reliability & performance

- [ ] **P0/S — Cache `DiskUsage`.** `GetStatus()` walks the whole data tree
  on every stats tick (1 s) plus Console polling. Cache the total and
  invalidate on upload/import/remove/relocate. Trivial, saves real CPU on
  big libraries.
- [ ] **P1/S — SQLite busy-timeout.** WAL is on but no `busy_timeout` is set;
  a phone upload racing a UI import can hit `SQLITE_BUSY`. Set
  `PRAGMA busy_timeout=5000` in `catalog.Open` + retry once on busy.
- [ ] **P1/S — Show the LAN IP, not `0.0.0.0`.** `Status.Addr` reports the
  bind address, which is useless to type into a phone. Enumerate interfaces
  (first private IPv4, e.g. `192.168.43.41`) and show `that:5000` in Console
  + Settings. Biggest phone-UX win per line of code in this file.
- [ ] **P1/M — QR-code connect.** Render the `IP:port` (+ future pairing
  code) as a QR in Console; phone scans instead of typing. Needs a Go QR
  lib and a frontend `<canvas>`/`<img>` render path.
- [ ] **P1/S — Transfer speed + ETA.** Track bytes-delta per stats tick;
  Console shows `↑ 3.2 MB/s` during active transfers. Almost free given
  the stats loop exists.
- [ ] **P1/S — Port-in-use help.** Bind failure currently just logs. Detect
  `EADDRINUSE`, suggest the owning process / offer next-port in Settings
  (persisted to config).
- [ ] **P2/M — `shared/` auto-catalog watcher.** Files dropped into `shared/`
  are servable but invisible in the catalog UI. `fsnotify` watcher →
  auto-`AddPath` on create (debounced). Makes the drop folder feel alive.
- [ ] **P2/M — Persist recent logs.** Keep last ~2000 lines in
  `Data/logs.jsonl` across restarts so user bug reports (like the READY
  timeout) survive a restart. Rotate by size.
- [ ] **P2/M — Content-hash dedup.** SHA-256 index over cataloged files;
  re-importing/receiving identical bytes skips the write and links the
  existing row. Saves hotspot time, not just disk.
- [ ] **P2/L — FTS5 catalog index.** `FindRelated` is a linear scan with
  per-name scoring — fine to ~10k rows, sloggy past 100k. Add an FTS5
  auxiliary index for the substring fast-path, keep fuzzy scoring on the
  top candidates. Benchmark first (see §5).

## 3. Frontend UX

- [ ] **P1/S — LAN IP + QR in Console** (frontend half of §2 items 3–4).
- [ ] **P1/S — Settings: host/port editing.** Currently config-file-only;
  add fields + validate + restart-server button. Pairs with port-in-use help.
- [ ] **P1/S — Import dry-run counts.** Importing a folder with 50k files
  is a leap of faith. `CatalogAddFolder` already returns counts — show
  "≈ N files" (fast pre-walk) with a confirm step above a threshold.
- [ ] **P1/M — Search/import progress.** Long user-storage walks give no
  feedback today. Stream `search-progress` events (backend already has the
  hook point) into the Import status bar + determinate progress.
- [ ] **P2/S — Copy-logs button.** One click copies Console lines for bug
  reports (we wanted the user's log here — make that frictionless).
- [ ] **P2/S — Toast history.** Toasts vanish; keep a dismissible history
  drawer (errors especially).
- [ ] **P2/M — Files: open-on-double-click + image thumbnails.** Reveal is
  there; default-app open and small thumbs make the view a real library.
- [ ] **P2/S — Theme toggle.** Dark-only today; respect OS theme with a
  light token set (CSS variables already isolate the palette).
- [ ] **P2/M — i18n scaffolding.** Strings are already mostly in one place
  per view; extract to `src/i18n/en.ts` + a `t()` helper before they spread.
- [ ] **P2/S — Shortcuts overlay.** `Ctrl+1..4` exist but are undiscoverable;
  add a `?` cheat-sheet + focus-visible audit.
- [ ] **P2/S — Onboarding drive-space check.** Before adopting a data root,
  verify free space (and warn on FAT32/removable drives).

## 4. Installer & updates

- [ ] **P0/S — Drop the `EnVar` plugin dependency.** `installer.nsi` uses
  `EnVar::AddValue/DeleteValue`, which is **not** in the stock NSIS 3
  plugin set — the installer fails to compile on a fresh NSIS install.
  Edit user-`PATH` via `WriteRegExpandStr HKCU Environment` + `StrStr`
  guard (both stock) instead. *(Do this before cutting any setup.exe.)*
- [ ] **P1/M — In-app updates.** Ship a GitHub-Releases feed check on start
  (version compare + download link, later one-click). Manual re-installs
  will rot the install base otherwise.
- [ ] **P1/S — `--data-dir` CLI override.** Power users / multi-profile
  setups want `WayerPC.exe --data-dir D:\alt` without env vars. Trivial
  argparse in `main.go`, document in BUILD.md.
- [ ] **P2/S — Uninstall data choice.** Uninstaller always keeps `data/`
  (good default) but never asks. Add a checkbox page: keep vs remove.
- [ ] **P2/M — Code signing.** Document the SmartScreen story (self-signed
  vs cert) so the first public setup.exe doesn't scare users.
- [ ] **P2/S — Start-with-Windows toggle.** Registry `Run` key + Settings
  checkbox. Server-first users will ask for it.
- [ ] **P2/S — Remember install dir.** `InstallDirRegKey` already does this
  — verify it actually pre-fills on reinstall and note it in release notes.

## 5. Testing & dev experience

- [ ] **P0/M — Protocol conformance script.** A single script (ncat/python)
  that drives `FOUND`/`MATCHES`/`READY`+bytes/`DONE`/`ERROR` paths —
  **including coalesced `FOUND`-plus-bytes bursts and newline-less
  headers** — runnable against any build in CI. The two field bugs we hit
  would both have been caught by this.
- [ ] **P1/S — Framing fuzz test.** Go fuzz test over `readCommand`: split
  random commands at every offset, with/without `\n`, `\r\n`, EOF mid-line.
  Cheap, high value for a parser that faces a hostile network.
- [ ] **P1/S — Mock parity checklist.** `frontend/src/dev-mock.ts` drifts
  from `app.go` silently. Add a CI check that every `api.*` method exists
  in the mock (names + arg counts), or generate the mock from `wailsjs`.
- [ ] **P1/M — Headless CI smoke.** Windows runner: `pnpm build` →
  `wails build` → run `--version` → drive the TCP script above against the
  real exe with `.data/`. Catches packaging-only breakage (embed, icons,
  manifest).
- [ ] **P2/S — `CHANGELOG.md`.** Keep-a-Changelog, fed by PR titles. Tag
  v0.2.0 for the rewrite now; v0.2.1 collects this list.
- [ ] **P2/M — Catalog benchmark.** Seed 100k rows, measure
  `FindRelated` p50/p99, decide cutoff/index policy with numbers (§2 FTS5
  depends on this).
- [ ] **P2/S — `wails dev` parity doc.** Hot-reload works for frontend;
  document the Go-rebuild loop + the `.data/` stub reset recipe in one
  place (BUILD.md covers build, not the inner loop).

## 6. Explicitly NOT suggested (rejected)

- Rewriting the Android client here — it lives in `ojilon/Wayer`, separate
  repo and release train. PC side stays client-tolerant instead.
- Changing the default port — the Android app and every doc references 5000.
- Real-time sync/watch of the whole user drive — hotspot + battery hostile;
  explicit import stays the model.
- Cloud relay / account system — out of scope for a hotspot-local tool.
