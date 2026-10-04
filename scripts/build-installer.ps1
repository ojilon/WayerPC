# Builds the Windows installer.
#   powershell -ExecutionPolicy Bypass -File scripts/build-installer.ps1
#   powershell -ExecutionPolicy Bypass -File scripts/build-installer.ps1 -Portable
#
# Steps: sync version -> pnpm install+build -> wails build (exe) -> makensis.
# Output: out/WayerPC-<version>-setup.exe  (or out/WayerPC-<version>-portable.zip)
param([switch]$Portable)

$ErrorActionPreference = "Stop"
$root = Split-Path -Parent $PSScriptRoot
Set-Location $root

$ver = (Get-Content "$root/version.json" -Raw | ConvertFrom-Json).version
if ($ver -notmatch '^\d+\.\d+\.\d+$') { throw "version.json is not semver: $ver" }
Write-Host "WayerPC v$ver"

Write-Host "[1/4] Syncing version..."
node scripts/sync-version.mjs

Write-Host "[2/4] Building frontend (pnpm + vite)..."
cmd /c "pnpm.cmd --dir frontend install"
if ($LASTEXITCODE -ne 0) { throw "pnpm install failed" }
cmd /c "pnpm.cmd --dir frontend build"
if ($LASTEXITCODE -ne 0) { throw "pnpm build failed" }

Write-Host "[3/4] Building Windows exe (wails)..."
wails build -platform windows/amd64 -o WayerPC.exe
if ($LASTEXITCODE -ne 0) { throw "wails build failed" }
$exe = "$root/build/bin/WayerPC.exe"
if (-not (Test-Path $exe)) { throw "expected exe missing: $exe" }
& $exe --version

New-Item -ItemType Directory -Path "$root/out" -Force | Out-Null

if ($Portable) {
  Write-Host "[4/4] Packing portable zip..."
  $zip = "$root/out/WayerPC-$ver-portable.zip"
  if (Test-Path $zip) { Remove-Item $zip -Force }
  Compress-Archive -Path "$root/build/bin/WayerPC.exe", "$root/.data" -DestinationPath $zip
  Write-Host "Output: $zip"
  return
}

Write-Host "[4/4] Building installer (NSIS)..."
if (-not (Get-Command makensis -ErrorAction SilentlyContinue)) {
  throw "makensis not found. Install NSIS 3 (https://nsis.sourceforge.io) and retry."
}
makensis "/DAPP_VERSION=$ver" "$root/build/installer.nsi"
if ($LASTEXITCODE -ne 0) { throw "makensis failed" }
Write-Host "Output: $root/out/WayerPC-$ver-setup.exe"
