# 05 — Installer (Windows, NSIS)

Goal: double-click setup → choose **drive + folder** → app lands in
`<chosen>\WayerPC\bin\WayerPC.exe`, data in `<chosen>\WayerPC\data`
(`shared/`, `received/`, `Data/wayerpc.db`), Start-Menu shortcut (so
**Windows search finds it**) + user-`PATH` entry (so **terminal** works).

## What the installer does (`build/installer.nsi`, MUI2)

1. `MUI_PAGE_DIRECTORY` — the only required chooser. Defaults to
   `D:\projects\WayerPC` if `D:` exists else `$PROGRAMFILES\WayerPC`;
   user may pick a drive root (`D:\`) or any subfolder (`D:\projects\`):
   the script appends `WayerPC` only when the leaf isn't already it.
2. Installs: `bin\WayerPC.exe` (+ `wayerpc.exe` console shim copy for
   terminal discoverability), `README.txt`, version stamp.
3. Creates `data\{shared,received,Data}` + `data\.installed` marker with
   version; never deletes `data\` on uninstall (checkbox-guarded).
4. Writes `%LOCALAPPDATA%\WayerPC\config.json` (`{app_dir, data_root}`)
   pointing at the chosen data dir (unless the file already points at a
   valid dir — reinstalls don't clobber a relocated library).
5. `CreateShortCut $SMPROGRAMS\WayerPC.lnk → bin\WayerPC.exe`
   (+ uninstall link) → appears in **Windows search**; `CreateShortCut
   $DESKTOP` optional section (default off).
6. Adds `<inst>\bin` to **HKCU `PATH`** via `EnvVarUpdate.nsh`
   (`HKCU\Environment`), broadcasts `WM_SETTINGCHANGE`; removes it on
   uninstall. Terminal check: reopen shell → `where WayerPC` / `wayerpc --version`.
7. Registry: `HKCU\Software\WayerPC` (`InstallDir`, `DataDir`, `Version`),
   `HKCU\...\Uninstall\WayerPC` (DisplayName with version, for Add/Remove).
8. `RequestExecutionLevel user` — no admin needed (HKCU + user-chosen dir);
   picking a protected path (e.g. `C:\Program Files`) triggers the
   elevation hint page.

## Build

```powershell
# from repo root (NSIS 3 + Wails CLI installed):
powershell -ExecutionPolicy Bypass -File scripts/build-installer.ps1
# → 1) node scripts/sync-version.mjs
#   → 2) wails build -platform windows/amd64 -o bin/WayerPC.exe
#   → 3) makensis /DAPP_VERSION=<from version.json> build/installer.nsi
#   → out/WayerPC-<version>-setup.exe
```

`--portable` flag in the script skips NSIS and zips `bin/`+`data-stub/`
instead. CI builds the exe first so a missing NSIS only fails the last step.

## First run (app-side, mirrors installer choice)

If `%LOCALAPPDATA%\WayerPC\config.json` is missing/corrupt, Settings shows
an onboarding banner: current candidate root → `Choose folder…` (Go
directory dialog) → `Init` creates the tree and saves config. The app never
silently writes to `C:`; dev runs use `.data/` via `WAYERPC_DATA_DIR`.
