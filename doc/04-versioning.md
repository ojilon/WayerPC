# 04 — Versioning (change before building)

Single source of truth: **`version.json` at repo root**. Edit it *before*
running any build; everything else is generated/embedded from it.

```json
{
  "appName": "WayerPC",
  "version": "0.2.0",
  "protocolVersion": 1,
  "description": "Phone ↔ PC transfer over hotspot"
}
```

## Consumers

| Consumer | How it reads | When |
|----------|--------------|------|
| Go (`internal/version`) | `//go:embed ../../version.json` → `Info` struct, `User-Agent`/log prefix, Settings screen | compile time (always fresh) |
| Frontend | `scripts/sync-version.mjs` generates `frontend/src/version.ts` (`export const APP_VERSION=…`) + injects `__APP_VERSION__` via Vite `define` | `pnpm --dir frontend prebuild` / `scripts/sync-version.mjs` |
| `wails.json` `info.productVersion` | `scripts/sync-version.mjs --wails` rewrites the field | pre-build |
| NSIS installer | `scripts/build-installer.ps1` parses `version.json` → `makensis /DAPP_VERSION=x.y.z` → `VIProductVersion`, installer filename `WayerPC-<ver>-setup.exe`, install registry key | installer build |
| `.data/` stub + logs | version stamped into DB `settings(app_version)` on open + log header | runtime |

## Changing the version

1. Edit `version.json` (`version` must be semver `MAJOR.MINOR.PATCH`).
2. Run `node scripts/sync-version.mjs` (or it runs automatically as
   `pnpm prebuild` / `wails build` pre-step).
3. `git diff` should show `frontend/src/version.ts` + `wails.json` updated;
   commit all three files together.

No other file may hard-code a version string (CI greps for `\d+\.\d+\.\d+`
outside `version.ts`/`wails.json`/CHANGELOG).
