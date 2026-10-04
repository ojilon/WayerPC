# PyInstaller spec for WayerPC (Windows).
# Run from the repository root:
#   pyinstaller packaging/wayerpc.spec
#
# Native DLLs in build/native/ are collected if present.

import os
from pathlib import Path
from PyInstaller.utils.hooks import collect_all

block_cipher = None
root = Path(SPECPATH).resolve().parent  # type: ignore[name-defined]

datas = []
binaries = []

native_dir = root / "build" / "native"
if native_dir.is_dir():
    for item in native_dir.iterdir():
        if item.suffix.lower() in {".dll", ".so", ".dylib"}:
            binaries.append((str(item), "."))

ctk_datas, ctk_binaries, ctk_hidden = collect_all("customtkinter")
datas += ctk_datas
binaries += ctk_binaries

a = Analysis(
    [str(root / "frontend" / "main.py")],
    pathex=[str(root)],
    binaries=binaries,
    datas=datas,
    hiddenimports=["customtkinter"] + ctk_hidden,
    hookspath=[],
    hooksconfig={},
    runtime_hooks=[],
    excludes=[],
    win_no_prefer_redirects=False,
    win_private_assemblies=False,
    cipher=block_cipher,
    noarchive=False,
)

pyz = PYZ(a.pure, a.zipped_data, cipher=block_cipher)

exe = EXE(
    pyz,
    a.scripts,
    [],
    exclude_binaries=True,
    name="WayerPC",
    debug=False,
    bootloader_ignore_signals=False,
    strip=False,
    upx=True,
    console=False,
    disable_windowed_traceback=False,
    icon=None,
)

coll = COLLECT(
    exe,
    a.binaries,
    a.zipfiles,
    a.datas,
    strip=False,
    upx=True,
    upx_exclude=[],
    name="WayerPC",
)
