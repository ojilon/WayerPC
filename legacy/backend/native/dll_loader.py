"""Locate native libs (dll/so) in the project build tree and copy them into the app folder."""

from __future__ import annotations

from pathlib import Path

from backend.manage_storage.bridge_to_app_folder import (
    copy_dll_to_app_folder,
    get_app_folder_path,
    obtain_file_location,
)

NATIVE_NAMES = (
    "libfilesearch",
    "libusersearch",
    "libimportcatalog",
)

_EXTENSIONS = (".dll", ".so", ".dylib")


def project_root() -> Path:
    return Path(__file__).resolve().parents[2]


def _candidates_for(stem: str) -> list[Path]:
    root = project_root()
    found: list[Path] = []
    for ext in _EXTENSIONS:
        name = stem + ext
        for folder in (root / "build", root / "build" / "native", root):
            if folder.is_dir():
                for item in folder.rglob(name):
                    if item.is_file():
                        found.append(item)
    return found


def sync_native_libs_to_app_folder() -> list[Path]:
    """Copy every known native lib into the app folder. Returns destinations copied."""
    copied: list[Path] = []
    try:
        get_app_folder_path()
    except FileNotFoundError:
        return copied

    for stem in NATIVE_NAMES:
        for src in _candidates_for(stem):
            dest_dir = copy_dll_to_app_folder(str(src))
            if dest_dir is not None:
                copied.append(Path(dest_dir) / src.name)
            break
    return copied


def locate_native_lib(stem: str) -> Path | None:
    """App folder first, then project build tree (and copy)."""
    for ext in _EXTENSIONS:
        status, location = obtain_file_location(stem + ext)
        if status == 0 and location.is_file():
            return location
    for src in _candidates_for(stem):
        copy_dll_to_app_folder(str(src))
        return src
    return None
