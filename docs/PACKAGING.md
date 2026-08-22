# Packaging WayerPC as a Windows app

## What to add to `.gitignore` **before** you run a package build

These are already listed in the repo `.gitignore`. Confirm they stay ignored so build junk never gets committed:

```
build/
dist/
__pycache__/
*.py[cod]
.venv/
venv/
*.dll
*.so
*.obj
*.pdb
logs/
*.log
release/
*.zip
/WayerPC.spec
backend/third_party/sqlite/sqlite3.c
backend/third_party/sqlite/sqlite3.h
```

Keep **`packaging/wayerpc.spec`** tracked. Do **not** add `packaging/` to gitignore.

## One-shot build

```powershell
powershell -ExecutionPolicy Bypass -File packaging/build_windows.ps1
```

Or by hand:

```powershell
python -m pip install -r requirements.txt
cmake -S . -B build
cmake --build build --config Release
python -m PyInstaller --noconfirm packaging/wayerpc.spec
```

Result: `dist/WayerPC/WayerPC.exe` plus dependent DLLs (folder dist, not one-file — easier for native libs).

## After packaging

Ship the whole `dist/WayerPC/` folder. Native libs from `build/native/` are pulled in by the spec when they exist.

On first run the packaged app still creates `%LOCALAPPDATA%\WayerPC\config.json` and the drive-level `WayerPC\` data folder.
