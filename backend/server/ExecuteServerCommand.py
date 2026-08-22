from pathlib import Path
from backend.manage_storage.bridge_to_app_folder import (
    ensure_shared_folder,
    ensure_received_folder,
    obtain_folder_location,
    get_app_folder_path,
)
from backend.native.dll_loader import locate_native_lib, sync_native_libs_to_app_folder
from backend.catalog.import_catalog import find_for_request, add_imported_path
import ctypes
import os


try:
    ensure_shared_folder()
    ensure_received_folder()
    sync_native_libs_to_app_folder()
except FileNotFoundError:
    pass


def locate_dll() -> Path | None:
    return locate_native_lib("libfilesearch")


def Initiate_file_search(filename: str) -> tuple:
    """
    Exact search via libfilesearch (shared folder, then app folder).
    Returns (result_code, path_or_none).
    0 = found, -1 = directory missing, -2 = not found / dll missing
    """
    location_to_dll = locate_dll()
    if not location_to_dll or not location_to_dll.is_file():
        return -2, None

    try:
        dll = ctypes.CDLL(str(location_to_dll))
    except OSError:
        return -1, None

    dll.search_file.argtypes = [
        ctypes.c_char_p,
        ctypes.c_char_p,
        ctypes.c_char_p,
        ctypes.c_char_p,
        ctypes.c_size_t,
    ]
    dll.search_file.restype = ctypes.c_int

    status, shared_path = obtain_folder_location("shared")
    if status != 0:
        return -1, shared_path

    try:
        project_root = get_app_folder_path()
    except FileNotFoundError:
        return -1, None

    path_buffer = ctypes.create_string_buffer(1024)
    result = dll.search_file(
        str(shared_path).encode("utf-8"),
        filename.encode("utf-8"),
        str(project_root).encode("utf-8"),
        path_buffer,
        1024,
    )
    if result == 0:
        return 0, path_buffer.value.decode("utf-8", errors="replace")
    return result, None


def _send_matches(conn, hits: list) -> tuple:
    lines = [f"MATCHES {len(hits)}"]
    for h in hits[:50]:
        lines.append(f"{h['name']}\t{h['path']}")
    payload = ("\n".join(lines) + "\n").encode("utf-8")
    try:
        conn.send(payload)
    except OSError:
        return 1, "failed to send MATCHES list"
    names = ", ".join(h["name"] for h in hits[:8])
    return 4, f"related names ({len(hits)}): {names}"


def _stream_file(conn, filepath: str, stats, stats_lock) -> tuple:
    try:
        filesize = os.path.getsize(filepath)
        conn.send(f"FOUND {filesize}\n".encode())
        send_file(conn, filepath)
        with stats_lock:
            stats["bytes_sent"] += filesize
        return 0, Path(filepath).name
    except FileNotFoundError:
        try:
            conn.send(b"ERROR file_access_denied")
        except OSError:
            pass
        return -1, f"file not accessible: {filepath}"
    except OSError as e:
        try:
            conn.send(b"ERROR file_access_denied")
        except OSError:
            pass
        return -1, f"OS error while sending: {e}"


def _resolve_and_send(query: str, conn, server_stats, stats_lock) -> tuple:
    """Catalog first (20%+ related names), then shared-folder DLL search."""
    hits = []
    try:
        hits = find_for_request(query, cutoff=0.2)
    except FileNotFoundError:
        hits = []

    exact = [h for h in hits if h["name"].lower() == query.lower()]
    if len(exact) == 1:
        return _stream_file(conn, exact[0]["path"], server_stats, stats_lock)
    if len(hits) == 1 and hits[0].get("score", 0) >= 0.8:
        return _stream_file(conn, hits[0]["path"], server_stats, stats_lock)
    if len(hits) > 1 or (len(hits) == 1 and hits[0].get("score", 0) < 0.8):
        return _send_matches(conn, hits)

    result, found_path = Initiate_file_search(query)
    if result == 0 and found_path:
        return _stream_file(conn, found_path, server_stats, stats_lock)
    if result == -1:
        try:
            conn.send(b"ERROR file_directory_missing_or_unreadable")
        except OSError:
            pass
        return 2, "shared/search directory missing or unreadable"
    try:
        conn.send(b"ERROR file_not_found")
    except OSError:
        pass
    return -1, f"file not found: {query}"


def Execute_server_command(data: str, conn, server_stats, stats_lock) -> tuple:
    """
    Always returns (status: int, detail: str).

    Status:
        0  /ask success (file streamed)
        3  /upload binary success (saved under received/)
        4  related-name list sent (MATCHES)
        1  protocol / validation / incomplete
        2  storage problem
       -1  logical failure already reported to client
    """

    if data.startswith("/ask"):
        parts = data.split(" ", 1)
        if len(parts) < 2 or not parts[1].strip():
            try:
                conn.send(b"ERROR invalid_command")
            except OSError:
                pass
            return 1, "invalid /ask command (missing filename)"
        return _resolve_and_send(parts[1].strip(), conn, server_stats, stats_lock)

    if data.startswith("/upload"):
        parts = data.split(" ")
        if len(parts) < 2:
            try:
                conn.send(b"ERROR invalid_upload_command")
            except OSError:
                pass
            return 1, "invalid /upload command"

        # /upload <query>  → related imported names (no numeric size)
        # /upload <size> <filename> → binary receive into received/
        size_token = parts[1].strip()
        if not size_token.isdigit() or len(parts) < 3:
            query = data[len("/upload") :].strip()
            if not query:
                try:
                    conn.send(b"ERROR invalid_upload_command")
                except OSError:
                    pass
                return 1, "invalid /upload command (need size+name or a search query)"
            try:
                hits = find_for_request(query, cutoff=0.2)
            except FileNotFoundError:
                hits = []
            if not hits:
                try:
                    conn.send(b"ERROR file_not_found")
                except OSError:
                    pass
                return -1, f"no imported files matching {query!r}"
            return _send_matches(conn, hits)

        try:
            filesize = int(size_token)
        except ValueError:
            try:
                conn.send(b"ERROR invalid_upload_command")
            except OSError:
                pass
            return 1, f"invalid filesize in /upload: {size_token!r}"

        if filesize < 0:
            try:
                conn.send(b"ERROR invalid_upload_command")
            except OSError:
                pass
            return 1, f"negative filesize in /upload: {filesize}"

        filename = Path(parts[2].strip()).name
        if not filename:
            try:
                conn.send(b"ERROR invalid_upload_command")
            except OSError:
                pass
            return 1, "empty filename in /upload"

        status, save_dir = obtain_folder_location("received")
        if status != 0 or not save_dir.is_dir():
            try:
                conn.send(
                    b"ERROR: Failed to obtain directory this side, to store the file to receive"
                )
            except OSError:
                pass
            return 2, f"received folder unavailable (status={status}, path={save_dir})"

        filepath = os.path.join(str(save_dir), filename)

        try:
            conn.send(b"READY")
        except OSError as e:
            return 1, f"failed to send READY for upload of '{filename}': {e}"

        try:
            with open(filepath, "wb") as f:
                remaining = filesize
                while remaining > 0:
                    chunk = conn.recv(min(4096, remaining))
                    if not chunk:
                        break
                    f.write(chunk)
                    remaining -= len(chunk)
        except OSError as e:
            try:
                conn.send(b"ERROR write_failed")
            except OSError:
                pass
            try:
                if os.path.isfile(filepath):
                    os.remove(filepath)
            except OSError:
                pass
            return 2, f"write failed for '{filename}': {e}"

        if remaining == 0:
            try:
                conn.send(b"DONE")
            except OSError:
                pass
            with stats_lock:
                server_stats["bytes_received"] += filesize
            add_imported_path(filepath)
            return 3, filename

        try:
            if os.path.isfile(filepath):
                os.remove(filepath)
        except OSError:
            pass
        try:
            conn.send(b"ERROR upload_incomplete")
        except OSError:
            pass
        return 1, f"upload incomplete for '{filename}' ({filesize - remaining}/{filesize} bytes)"

    try:
        conn.send(b"ERROR unknown_protocol_command.")
    except OSError:
        pass
    return 1, f"unknown protocol command: {data!r}"


def send_file(conn, filepath):
    with open(filepath, "rb") as f:
        while True:
            data = f.read(4096)
            if not data:
                break
            conn.sendall(data)


if __name__ == "__main__":
    pass
