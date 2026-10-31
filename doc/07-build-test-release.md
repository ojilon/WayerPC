# 07 — Build, test, release

Prereqs: **Go ≥ 1.23** (`go version`), **Node 20+**, **pnpm**
(`pnpm.cmd --version` — on locked-down PowerShell use `pnpm.cmd`, not
`pnpm.ps1`), **Wails CLI v2** (`wails version`), **NSIS 3** (installer only).

## Dev

```powershell
node scripts/sync-version.mjs
go test ./...                                   # backend, uses t.TempDir()
cmd /c "pnpm.cmd --dir frontend install"       # first time only
cmd /c "pnpm.cmd --dir frontend dev"            # vite browser mock (no Go)
$env:WAYERPC_DATA_DIR = "$PWD/.data"; wails dev # full app, stub data
```

## Production build

```powershell
node scripts/sync-version.mjs   # regenerate version.ts + wails.json
cmd /c "pnpm.cmd --dir frontend install"
cmd /c "pnpm.cmd --dir frontend build"   # tsc + vite → frontend/dist
wails build -platform windows/amd64 -o bin/WayerPC.exe
.\bin\WayerPC.exe --version
```

Version bump: edit `version.json` → run sync → commit the three touched
files (`version.json`, `frontend/src/version.ts`, `wails.json`).

## Installer

```powershell
powershell -ExecutionPolicy Bypass -File scripts/build-installer.ps1
# out/WayerPC-<version>-setup.exe
```

Install → pick drive/folder (`D:\projects\` → `D:\projects\WayerPC\`);
verify: Windows-search `WayerPC` launches; new terminal
`where WayerPC` + `wayerpc --version` work; data at
`<chosen>\WayerPC\data`.

## Phone smoke test (protocol parity with legacy)

1. PC and Android on same hotspot; note PC IP; app Console shows
   `Listening on 0.0.0.0:5000`.
2. `ncat <pc-ip> 5000` (or Android client):
   `/ask <imported-name>` → `FOUND`+bytes or `MATCHES` list;
   `/upload <query>` → `MATCHES`; `/upload <size> <name>` + bytes → `DONE`,
   file appears in `received/` + catalog.
3. Cutoff/fuzzy rule: ≥20% name similarity lists; exact/strong-single streams.

## Release checklist

- [ ] `version.json` bumped + sync ran
- [ ] `go test ./...` green, `pnpm --dir frontend build` green
- [ ] `wails build` exe smoke-tested with `.data/` then real dir
- [ ] installer built, fresh-VM install to root **and** subfolder verified,
      uninstall keeps `data/` only when asked
- [ ] `BUILD.md` version table updated, tag `v<version>` pushed
