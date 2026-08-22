from backend.manage_storage.bridge_to_app_folder import (
    ensure_shared_folder,
    obtain_folder_location,
    copy_dll_to_app_folder,
    get_app_folder_path,
    obtain_file_location,
)
import ctypes
import os
import threading


# Module-level: ensure the "shared" folder exists at startup
# This is done once when the module is loaded, not at class definition time
ensure_shared_folder()


def Initiate_file_search(filename: str) -> tuple:
    """
    Searches for a file using the backend/filesearch C++23 DLL.
    Returns (result_code, path_buffer) or None on failure.
    result_code: 0 = found, -1 = directory missing/unreadable, -2 = file not found
    """

    # Copy dll to app folder if not already there (for development testing)
    location_to_dll = obtain_file_location("libfilesearch.dll")

    if not location_to_dll or not location_to_dll.is_file():
        return -2, None

    try:
        dll = ctypes.CDLL(str(location_to_dll))
    except Exception:
        return -1, None

    dll.search_file.argtypes = [ctypes.c_char_p, ctypes.c_char_p, ctypes.c_char_p, ctypes.c_char_p, ctypes.c_size_t]
    dll.search_file.restype = ctypes.c_int

    # Use app folder as shared path context
    status, shared_path = obtain_folder_location("shared")
    if not status == 0:
        return -1, shared_path

    project_root = get_app_folder_path()
    if not project_root:
        return -1, project_root

    path_buffer = ctypes.create_string_buffer(260)
    shared_path_str = str(shared_path)
    project_root_str = str(project_root)

    result = dll.search_file(shared_path_str.encode('utf-8'), filename.encode('utf-8'), project_root_str.encode('utf-8'), path_buffer, 260)

    return 0, result


def Execute_server_command(data: str, conn, server_stats, stats_lock) -> tuple:

    #sending to client
    if data.startswith("/ask"):
        parts = data.split(" ", 1)
        if len(parts) < 2:
            conn.send(b"ERROR invalid_command")
            return 1, parts

        filename = parts[1].strip()
        #app_dir = get_app_folder_path()
        #if not app_dir:
        #    return 2, None

        #project_root_str = str(app_dir)

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
            print("error -1")
            return -1, "ERROR file_directory_missing_or_unreadable"
        elif result == -2:
            conn.send(b"ERROR file_not_found")
            return -1, None
        else:
            conn.send(b"ERROR unknown_system_fault")
            return -1, None

    #receiving files from PC
    elif data.startswith("/upload"):
        parts = data.split(" ")
        if len(parts) < 3:
            conn.send(b"ERROR invalid_upload_command")
            return 1, parts

        filename = parts[2].strip()
        filesize = int(parts[1].strip())

        # Use obtain_folder_location to find 'received' subfolder within app folder
        status, save_dir = obtain_folder_location("received")
        if status != 0 or not save_dir.is_dir():
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


if __name__ == '__main__':
    pass