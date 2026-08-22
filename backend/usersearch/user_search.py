"""Python glue for libusersearch. Falls back to a conservative walk if the DLL is missing."""

from __future__ import annotations

import ctypes
import os
from pathlib import Path

from backend.native.dll_loader import locate_native_lib

_SKIP_PARTS = {
    "windows",
    "program files",
    "program files (x86)",
    "programdata",
    "appdata",
    "$recycle.bin",
    "system volume information",
    "windowsapps",
}


def _skip(path: Path) -> bool:
    parts = [p.lower() for p in path.parts]
    joined = str(path).lower()
    if "wayerpc" in parts:
        return True
    for blocked in _SKIP_PARTS:
        if blocked in joined:
            return True
    return False


def _user_roots() -> list[Path]:
    home = Path.home()
    roots = []
    for name in ("Documents", "Downloads", "Desktop", "Pictures", "Videos", "Music"):
        p = home / name
        if p.is_dir():
            roots.append(p)
    # Extra Windows drives
    if os.name == "nt":
        for letter in "DEFGHIJKLMNOPQRSTUVWXYZ":
            drive = Path(f"{letter}:\\")
            if drive.is_dir():
                roots.append(drive)
    else:
        for extra in (Path("/mnt"), Path("/media")):
            if extra.is_dir():
                roots.append(extra)
    return roots


def _parse_buffer(raw: bytes) -> list[dict]:
    text = raw.split(b"\x00", 1)[0].decode("utf-8", errors="replace")
    results = []
    seen = set()
    for line in text.splitlines():
        if "\t" not in line:
            continue
        name, path = line.split("\t", 1)
        if path in seen:
            continue
        seen.add(path)
        results.append({"name": name, "path": path})
    return results


def search_user_storage(query: str, limit: int = 200) -> list[dict]:
    """Search user files/folders by name fragment. Skips system + app data."""
    query = (query or "").strip()
    if not query:
        return []

    lib_path = locate_native_lib("libusersearch")
    if lib_path and lib_path.is_file():
        try:
            dll = ctypes.CDLL(str(lib_path))
            dll.search_user_storage.argtypes = [
                ctypes.c_char_p,
                ctypes.c_char_p,
                ctypes.c_size_t,
            ]
            dll.search_user_storage.restype = ctypes.c_int
            buf = ctypes.create_string_buffer(1_048_576)
            n = dll.search_user_storage(query.encode("utf-8"), buf, 1_048_576)
            if n >= 0:
                return _parse_buffer(buf.raw)[:limit]
        except OSError:
            pass

    # Python fallback
    q = query.lower()
    hits: list[dict] = []
    for root in _user_roots():
        try:
            for dirpath, dirnames, filenames in os.walk(root):
                pdir = Path(dirpath)
                if _skip(pdir):
                    dirnames[:] = []
                    continue
                # prune skipped children
                dirnames[:] = [d for d in dirnames if not _skip(pdir / d)]
                folder_hit = q in pdir.name.lower()
                if folder_hit:
                    for fn in filenames:
                        fp = pdir / fn
                        hits.append({"name": fn, "path": str(fp)})
                        if len(hits) >= limit:
                            return hits
                for fn in filenames:
                    if q in fn.lower():
                        hits.append({"name": fn, "path": str(pdir / fn)})
                        if len(hits) >= limit:
                            return hits
        except OSError:
            continue
    return hits
