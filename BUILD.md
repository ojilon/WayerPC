# Building WayerPC

WayerPC is a Wails app: **pure-Go backend** (`internal/`, `main.go`, `app.go`)
plus a **Vite + TypeScript frontend** (`frontend/`, built with `pnpm`).
The old Python/C++ implementation lives under `legacy/` for reference only.

## Prereqs

| Tool | Check | Notes |
|------|-------|-------|
| Go ≥ 1.23 | `go version` | backend + exe |
| Node 20+ | `node --version` | Vite build |
| pnpm | `pnpm.cmd --version` | see ExecutionPolicy note below |
| Wails CLI v2 | `wails version` | `go install github.com/wailsapp/wails/v2/cmd/wails@latest` |
| NSIS 3 | `makensis -version` | installer only ([nsis.sourceforge.io](https://nsis.sourceforge.io)) |

> **PowerShell + pnpm:** if `pnpm` fails with “running scripts is disabled”,
> call `pnpm.cmd` (e.g. `cmd /c "pnpm.cmd --dir frontend build"`) or run
> `Set-ExecutionPolicy -Scope CurrentUser RemoteSigned` once. All commands
> below use `pnpm.cmd` so they work with the default policy.
>
> **First pnpm install** asks to approve the `esbuild` build script
> (one time): `pnpm.cmd --dir frontend approve-builds esbuild`.
> The approval is stored in `frontend/pnpm-workspace.yaml`.

## Version first

`version.json` is the single source of truth. Change it **before** building:

```powershell
# edit version.json, then:
node scripts/sync-version.mjs   # regenerates frontend/src/version.ts,
                                # internal/version/version.json, wails.json
```

## Dev

```powershell
go test ./internal/...                    # backend suite (no phone needed)
cmd /c "pnpm.cmd --dir frontend dev"      # browser preview with mock data (http://localhost:5173)

# Full app with the .data/ stub (no install, no %LOCALAPPDATA% writes):
$env:WAYERPC_DATA_DIR = "$PWD/.data"
wails dev
```

## Production build

```powershell
node scripts/sync-version.mjs
cmd /c "pnpm.cmd --dir frontend install"
cmd /c "pnpm.cmd --dir frontend build"    # tsc + vite -> frontend/dist (required before go build)
wails build -platform windows/amd64 -o WayerPC.exe
# -> build/bin/WayerPC.exe
.\build\bin\WayerPC.exe --version
```

## Installer

```powershell
powershell -ExecutionPolicy Bypass -File scripts/build-installer.ps1
# -> out/WayerPC-<version>-setup.exe
# Portable zip instead:
powershell -ExecutionPolicy Bypass -File scripts/build-installer.ps1 -Portable
```

Install flow: pick any drive/folder on the Directory page (`D:\` or
`D:\projects\` both work — `WayerPC` is appended unless already the leaf).
You get `<chosen>\WayerPC\bin\WayerPC.exe` (+ user `PATH`), data in
`<chosen>\WayerPC\data`, a Start Menu entry (Windows search), and a
first-run `%LOCALAPPDATA%\WayerPC\config.json`. Verify:

```powershell
where WayerPC            # new terminal (PATH refresh)
wayerpc --version
```

Uninstall keeps `data/` (your library outlives the app).

## Phone smoke test (protocol parity with the legacy server)

1. PC + Android on the same hotspot; Console tab shows `Listening on 0.0.0.0:5000`.
2. `ncat <pc-ip> 5000` (or the Android client):
   - `/ask <imported-name>` → `FOUND <size>` + bytes, or `MATCHES n` + names
   - `/upload <query>` → `MATCHES n` (catalog search, no transfer)
   - `/upload <size> <name>`, wait for `READY`, send bytes → `DONE`;
     file appears in `received/` and the catalog.
3. Fuzzy rule: ≥ 20% name similarity lists candidates; exact/strong-single streams.

## Release checklist

- [ ] `version.json` bumped + `node scripts/sync-version.mjs` run
- [ ] `go test ./internal/...` green, `pnpm --dir frontend build` green
- [ ] `wails build` exe smoke-tested (`.data/` first, then a real folder)
- [ ] Installer built; fresh install to a drive root **and** a subfolder verified
- [ ] Tag `v<version>` pushed

## Troubleshooting

| Symptom | Fix |
|---------|-----|
| `pnpm` blocked by ExecutionPolicy | use `pnpm.cmd` / run from `cmd`, or set policy per above |
| `ERR_PNPM_IGNORED_BUILDS` (esbuild) | `pnpm.cmd --dir frontend approve-builds esbuild` |
| `go build` fails on `frontend/dist` | run `pnpm --dir frontend build` first (dist is generated, untracked) |
| wails build “watcher” JSON error | fixed — `wails.json` no longer sets `frontend:dev:watcher` |
| Port 5000 busy | another instance is running (single app lauches one server); stop it first |
| `/ask` finds nothing | file must be imported (Import tab) or dropped in `shared/` |
| `failed to send READY … i/o timeout` | phone idled past the 5-min window or the hotspot stalled (sleep/doze); fixed server-side so slow-but-alive transfers survive — just retry the upload from the phone |
