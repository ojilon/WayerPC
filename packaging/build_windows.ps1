# Build a folder-based Windows app with PyInstaller.
# Run from the repo root in PowerShell:
#   powershell -ExecutionPolicy Bypass -File packaging/build_windows.ps1

$ErrorActionPreference = "Stop"
Set-Location (Split-Path -Parent $PSScriptRoot)

Write-Host "Installing Python deps..."
python -m pip install -r requirements.txt

if (Test-Path "build/native") {
    Write-Host "Native libs present in build/native — they will be bundled."
} else {
    Write-Host "No build/native yet. Optional: cmake -S . -B build ; cmake --build build --config Release"
}

Write-Host "Running PyInstaller..."
python -m PyInstaller --noconfirm packaging/wayerpc.spec

Write-Host "Output: dist/WayerPC/WayerPC.exe"
