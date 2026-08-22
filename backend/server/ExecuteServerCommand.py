from pathlib import Path
from backend.manage_storage.bridge_to_app_folder import (
    ensure_shared_folder,
    ensure_received_folder,
    obtain_folder_location,
    obtain_file_location,
    copy_dll_to_app_folder,
    get_app_folder_path,
)
import ctypes
import os


# Module-level: ensure "shared" and "received" folders exist at startup.
# Done once when the module is loaded (after storage has been initialized by the app).
try:
    ensure_shared_folder()
    ensure_received_folder()
except FileNotFoundError:
    # Storage not set up yet (e.g. first import before setup_storage runs).
    # Folders will be created on first use via obtain_folder_location.
    pass


def locate_dll() -> Path | None:
    """
    Finds libfilesearch.dll for the C++ file search.
    Priority: inside the app folder first, then dev fallback ->
    search the project tree and copy it into the app folder.
    Returns the Path to the dll or None if unavailable.
    """
    # 1. Look inside the app folder
    status, location = obtain_file_location("libfilesearch.dll")
    if status == 0 and location.is_file():
        return location

    # 2. Development fallback: search the project source tree
    project_root = Path(__file__).resolve().parents[2]
    if project_root.is_dir():
        for candidate in project_root.rglob("libfilesearch.dll"):
            if candidate.is_file():
                # Copy a working copy into the app folder for future runs
                copied = copy_dll_to_app_folder(str(candidate))
                if copied is not None:
                    print(f"[DLL] Copied '{candidate.name}' into app folder: {copied}")
                    return candidate
                return candidate

    return None


def Initiate_file_search(filename: str) -> tuple:
    """
    Searches for a file using the backend/filesearch C++23 DLL.
    Returns (result_code, path_buffer).
    result_code: 0 = found, -1 = directory missing/unreadable, -2 = file not found/dll missing
    """

    # Copy dll to app folder if not already there (for development testing)
    location_to_dll = locate_dll()

    if not location_to_dll or not location_to_dll.is_file():
        return -2, None

    try:
        dll = ctypes.CDLL(str(location_to_dll))
    except Exception:
        return -1, None

    dll.search_file.argtypes = [
        ctypes.c_char_p,
        ctypes.c_char_p,
        ctypes.c_char_p,
        ctypes.c_char_p,
        ctypes.c_size_t,
    ]
    dll.search_file.restype = ctypes.c_int

    # Use app folder as shared path context
    status, shared_path = obtain_folder_location("shared")
    if status != 0:
        return -1, shared_path

    project_root = get_app_folder_path()
    if not project_root:
        return -1, project_root

    path_buffer = ctypes.create_string_buffer(260)
    shared_path_str = str(shared_path)
    project_root_str = str(project_root)

    result = dll.search_file(
        shared_path_str.encode("utf-8"),
        filename.encode("utf-8"),
        project_root_str.encode("utf-8"),
        path_buffer,
        260,
    )

    return result, path_buffer


def Execute_server_command(data: str, conn, server_stats, stats_lock) -> tuple:
    """
    Parse and execute a client command.

    Always returns (status: int, detail: str) where detail is never None
    (so the server handler can log meaningfully).

    Status codes:
        0  – /ask success: file found and streamed to client
        3  – /upload success: file fully received and saved
        1  – protocol / validation error, or upload interrupted / incomplete
        2  – storage / directory problem (received/shared/app folder)
       -1  – logical failure already reported to client (file not found,
             access denied, unknown command result, etc.)
    """

    # ---------- /ask : send a file to the client ----------
    if data.startswith("/ask"):
        parts = data.split(" ", 1)
        if len(parts) < 2 or not parts[1].strip():
            try:
                conn.send(b"ERROR invalid_command")
            except Exception:
                pass
            return 1, "invalid /ask command (missing filename)"

        filename = parts[1].strip()
        result, path_buffer = Initiate_file_search(filename)

        if result == 0:
            filepath = path_buffer.value.decode("utf-8")
            try:
                filesize = os.path.getsize(filepath)
                msg = f"FOUND {filesize}\n"
                conn.send(msg.encode())

                # Stream file data
                send_file(conn, filepath)

                with stats_lock:
                    server_stats["bytes_sent"] += filesize

                return 0, filename

            except FileNotFoundError:
                try:
                    conn.send(b"ERROR file_access_denied")
                except Exception:
                    pass
                return -1, f"file found by search but not accessible: {filepath}"
            except OSError as e:
                try:
                    conn.send(b"ERROR file_access_denied")
                except Exception:
                    pass
                return -1, f"OS error while sending '{filename}': {e}"

        elif result == -1:
            try:
                conn.send(b"ERROR file_directory_missing_or_unreadable")
            except Exception:
                pass
            return 2, "shared/search directory missing or unreadable"

        elif result == -2:
            try:
                conn.send(b"ERROR file_not_found")
            except Exception:
                pass
            return -1, f"file not found: {filename}"

        else:
            try:
                conn.send(b"ERROR unknown_system_fault")
            except Exception:
                pass
            return -1, f"unknown search result code {result} for '{filename}'"

    # ---------- /upload : receive a file from the client ----------
    elif data.startswith("/upload"):
        parts = data.split(" ")
        if len(parts) < 3:
            try:
                conn.send(b"ERROR invalid_upload_command")
            except Exception:
                pass
            return 1, "invalid /upload command (need: /upload <size> <filename>)"

        try:
            filesize = int(parts[1].strip())
        except ValueError:
            try:
                conn.send(b"ERROR invalid_upload_command")
            except Exception:
                pass
            return 1, f"invalid filesize in /upload: {parts[1]!r}"

        if filesize < 0:
            try:
                conn.send(b"ERROR invalid_upload_command")
            except Exception:
                pass
            return 1, f"negative filesize in /upload: {filesize}"

        # Keep only the basename to avoid path traversal
        filename = Path(parts[2].strip()).name
        if not filename:
            try:
                conn.send(b"ERROR invalid_upload_command")
            except Exception:
                pass
            return 1, "empty filename in /upload"

        # Ensure 'received' exists (created at startup; recreate if deleted)
        status, save_dir = obtain_folder_location("received")
        if status != 0 or not save_dir.is_dir():
            try:
                conn.send(
                    b"ERROR: Failed to obtain directory this side, to store the file to receive"
                )
            except Exception:
                pass
            return 2, f"received folder unavailable (status={status}, path={save_dir})"

        filepath = os.path.join(str(save_dir), filename)

        try:
            conn.send(b"READY")
        except Exception as e:
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
            except Exception:
                pass
            # Best-effort cleanup of partial file
            try:
                if os.path.isfile(filepath):
                    os.remove(filepath)
            except OSError:
                pass
            return 2, f"write failed for '{filename}': {e}"

        if remaining == 0:
            try:
                conn.send(b"DONE")
            except Exception:
                pass
            with stats_lock:
                server_stats["bytes_received"] += filesize
            return 3, filename
        else:
            # Incomplete transfer — remove partial file
            try:
                if os.path.isfile(filepath):
                    os.remove(filepath)
            except OSError:
                pass
            try:
                conn.send(b"ERROR upload_incomplete")
            except Exception:
                pass
            return 1, f"upload incomplete for '{filename}' ({filesize - remaining}/{filesize} bytes)"

    # ---------- unknown command ----------
    else:
        try:
            conn.send(b"ERROR unknown_protocol_command.")
        except Exception:
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
