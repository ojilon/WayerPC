from .Locate import get_project_root, search_root_subfolder
import ctypes
import os
import threading

# Module-level: ensure the "shared" folder exists at startup
# This is done once when the module is loaded, not at class definition time
location_to_shared = search_root_subfolder("shared")
if not location_to_shared or not location_to_shared.is_dir():
    # Could not find shared folder - exit or handle per-request
    exit(1)


def Initiate_file_search(filename: str) -> tuple:
    """
    Searches for a file using the backend/filesearch C++23 DLL.
    Returns (result_code, path_buffer) or None on failure.
    result_code: 0 = found, -1 = directory missing/unreadable, -2 = file not found
    """
    location_to_dll = search_root_subfolder("libfilesearch.dll")
    if not location_to_dll:
        return None

    try:
        dll = ctypes.CDLL(str(location_to_dll))
    except Exception:
        return None

    dll.search_file.argtypes = [ctypes.c_char_p, ctypes.c_char_p,
        ctypes.c_char_p, ctypes.c_char_p, ctypes.c_size_t]
    dll.search_file.restype = ctypes.c_int

    project_root = get_project_root()
    if not project_root:
        return None

    path_buffer = ctypes.create_string_buffer(260)
    shared_path_str = str(location_to_shared)
    project_root_str = str(project_root)

    result = dll.search_file(shared_path_str.encode('utf-8'), filename.encode('utf-8'),
                             project_root_str.encode('utf-8'), path_buffer, 260)

    return result, path_buffer


def Execute_server_command(data: str, conn, server_stats, stats_lock) -> tuple:

    #sending to client
    if data.startswith("/ask"):
        parts = data.split(" ", 1)
        if len(parts) < 2:
            conn.send(b"ERROR invalid_command")
            return 1, parts

        filename = parts[1].strip()
        project_root = get_project_root()
        if not project_root:
            return 2, project_root

        result, path_buffer = Initiate_file_search(filename)

        if result == 0:
            filepath = path_buffer.value.decode('utf-8')
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
                conn.send(b"ERROR file_access_denied")

        elif result == -1:
            conn.send(b"ERROR shared_file_directory_missing_or_unreadable")
        elif result == -2:
            conn.send(b"ERROR file_not_found")
        else:
            conn.send(b"ERROR unknown_system_fault")

    #receiving files from PC
    elif data.startswith("/upload"):
        parts = data.split(" ")
        if len(parts) < 3:
            conn.send(b"ERROR invalid_upload_command")
            return 1, parts

        filename = parts[2].strip()
        filesize = int(parts[1].strip())

        save_dir = search_root_subfolder("received")
        if not save_dir or not save_dir.is_dir():
            conn.send(b"ERROR: Failed to obtain directory this side, to store the file to receive")
            return 2, save_dir

        filepath = os.path.join(save_dir, filename)
        conn.send(b"READY")

        with open(filepath, "wb") as f:
            remaining = filesize
            while remaining > 0:
                chunk = conn.recv(min(4096, remaining))
                if not chunk:
                    break
                f.write(chunk)
                remaining -= len(chunk)

        if remaining == 0:
            conn.send(b"DONE")
            with stats_lock:
                server_stats["bytes_received"] += filesize
            return -1, filename
        else:
            return 1, filename

    else:
        conn.send(b"ERROR unknown_protocol_command.")
        return 1


def send_file(conn, filepath):
    with open(filepath, "rb") as f:
        while True:
            data = f.read(4096)

            if not data:
                break

            conn.sendall(data)


if __name__ == "__main__":
    pass