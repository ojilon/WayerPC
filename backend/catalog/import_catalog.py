"""Import catalog — SQLite, procedural (no classes).

Stores absolute paths of files the user chose to import. Files are NOT copied
into shared/; the catalog is the source of truth for /ask lookups.
"""

from __future__ import annotations

import difflib
import os
import sqlite3
from pathlib import Path

from backend.manage_storage.bridge_to_app_folder import get_app_folder_path


def catalog_db_path() -> Path:
    app_dir = get_app_folder_path()
    data = app_dir / "Data"
    data.mkdir(parents=True, exist_ok=True)
    return data / "imports.db"


def _connect() -> sqlite3.Connection:
    path = catalog_db_path()
    conn = sqlite3.connect(str(path))
    conn.row_factory = sqlite3.Row
    conn.execute(
        """
        CREATE TABLE IF NOT EXISTS imported_files (
            id INTEGER PRIMARY KEY AUTOINCREMENT,
            path TEXT NOT NULL UNIQUE,
            name TEXT NOT NULL,
            size_bytes INTEGER,
            imported_at TEXT NOT NULL DEFAULT (datetime('now'))
        )
        """
    )
    conn.commit()
    return conn


def add_imported_path(file_path: str) -> tuple[int, str]:
    """
    Register a file path. Does not copy the file.
    Returns (0, path) on success, (1, reason) on failure.
    """
    p = Path(file_path)
    if not p.is_file():
        return 1, f"not a file: {file_path}"
    resolved = str(p.resolve())
    size = p.stat().st_size
    conn = _connect()
    try:
        conn.execute(
            "INSERT OR REPLACE INTO imported_files(path, name, size_bytes) VALUES (?,?,?)",
            (resolved, p.name, size),
        )
        conn.commit()
    finally:
        conn.close()
    return 0, resolved


def add_imported_folder(folder_path: str) -> tuple[int, int]:
    """Register every regular file under a folder. Returns (ok_count, skip_count)."""
    root = Path(folder_path)
    ok = 0
    skip = 0
    if not root.is_dir():
        return 0, 1
    for item in root.rglob("*"):
        if item.is_file():
            status, _ = add_imported_path(str(item))
            if status == 0:
                ok += 1
            else:
                skip += 1
    return ok, skip


def remove_imported_path(file_path: str) -> int:
    conn = _connect()
    try:
        cur = conn.execute("DELETE FROM imported_files WHERE path = ?", (file_path,))
        conn.commit()
        return cur.rowcount
    finally:
        conn.close()


def list_imported() -> list[dict]:
    conn = _connect()
    try:
        rows = conn.execute(
            "SELECT id, path, name, size_bytes, imported_at FROM imported_files ORDER BY name"
        ).fetchall()
        return [dict(r) for r in rows]
    finally:
        conn.close()


def _score(query: str, name: str) -> float:
    q = query.lower()
    n = name.lower()
    if q == n:
        return 1.0
    if q in n:
        return max(0.6, len(q) / max(len(n), 1))
    return difflib.SequenceMatcher(None, q, n).ratio()


def find_related(query: str, cutoff: float = 0.2) -> list[dict]:
    """Return catalog rows whose name is at least `cutoff` similar to query (default 20%)."""
    query = (query or "").strip()
    if not query:
        return []
    rows = list_imported()
    scored = []
    for row in rows:
        s = _score(query, row["name"])
        if s >= cutoff:
            item = dict(row)
            item["score"] = s
            scored.append(item)
    scored.sort(key=lambda r: r["score"], reverse=True)
    return scored


def find_for_request(query: str, cutoff: float = 0.2) -> list[dict]:
    """
    Resolve a phone request against the catalog.
    Drops rows whose files no longer exist on disk.
    """
    hits = find_related(query, cutoff=cutoff)
    alive = []
    for h in hits:
        if os.path.isfile(h["path"]):
            alive.append(h)
    return alive
