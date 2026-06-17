import socket
import os
import ctypes
import threading
import time
from .transfer import send_file
from .config import *
from .Locate import nail_file_location, get_project_root, search_root_subfolder

# Global dictionary for monitoring stats (shared across threads)
# Python dictionaries are thread-safe for simple reads/writes due to the GIL,
# but using a Lock is best practice for consistency and future C3/Zig translations!
stats_lock = threading.Lock()
server_stats = {
    "is_running": False,
    "active_connections": 0,
    "total_connections_handled": 0,
    "bytes_sent": 0,
    "bytes_received": 0,
    "start_time": None
}

# Locate dll and folder setup 
location_to_dll = nail_file_location("libfilesearch.dll")
if not location_to_dll:
    print("Failed to get the 'libfilesearch.dll' file ")
    exit(1)

location_to_shared = search_root_subfolder("shared")
if not location_to_shared.is_dir():
    print(f"Failed to create the folder: {location_to_shared}")
    exit(1)

dll = ctypes.CDLL(location_to_dll)
dll.search_file.argtypes = [ctypes.c_char_p, ctypes.c_char_p, ctypes.c_char_p, ctypes.c_char_p, ctypes.c_size_t]
dll.search_file.restype = ctypes.c_int


def handle_client(conn, addr):
    """Handles an individual client connection inside its own dedicated thread."""
    # Increment active connections safely using a Mutex (Lock)
    with stats_lock:
        server_stats["active_connections"] += 1
        server_stats["total_connections_handled"] += 1

    print(f"\n[THREAD-{threading.get_ident()}] Handling connection from {addr}")

    try:
        data = conn.recv(1024).decode('utf-8').strip()
        if not data:
            return

        # =========================================================
        # CASE 1: PHONE IS DOWNLOADING FROM PC (/ask)
        # =========================================================
        if data.startswith("/ask"):
            parts = data.split(" ", 1)
            if len(parts) < 2:
                conn.send(b"ERROR invalid_command")
                return

            filename = parts[1].strip()
            project_root = get_project_root()
            if not project_root:
                print("server4.py : Failed to get project root.")
                return
            
            path_buffer = ctypes.create_string_buffer(260)
            shared_path_str = str(location_to_shared)
            project_root_str = str(project_root)

            result = dll.search_file(shared_path_str.encode('utf-8'), filename.encode('utf-8'), project_root_str.encode('utf-8'), path_buffer, 260)

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

                except FileNotFoundError:
                    conn.send(b"ERROR file_access_denied")

            elif result == -1:
                conn.send(b"ERROR shared_file_directory_missing_or_unreadable")
            elif result == -2:
                conn.send(b"ERROR file_not_found")
            else:
                conn.send(b"ERROR unknown_system_fault")

        # =========================================================
        # CASE 2: PHONE IS UPLOADING TO PC (/upload)
        # =========================================================
        elif data.startswith("/upload"):
            parts = data.split(" ")
            if len(parts) < 3:
                conn.send(b"ERROR invalid_upload_command")
                return
            
            filename = parts[2].strip()
            filesize = int(parts[1].strip())
            
            save_dir = search_root_subfolder("received")
            if not save_dir.is_dir():
                conn.send(b"ERROR backend_directory_creation_failed")
                return
                                            
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
            else:
                print(f"Upload of {filename} interrupted.")
                
        else:
            conn.send(b"ERROR unknown_protocol_command.")                     

    except Exception as e:
        print(f"Server Thread Exception for {addr}: {e}")
    finally:
        conn.close()
        # Decrement active connections safely when the thread dies
        with stats_lock:
            server_stats["active_connections"] -= 1
        print(f"[THREAD-{threading.get_ident()}] Connection with {addr} closed.")


def start_server():
    """Main loop accepting TCP connections. Runs inside a background thread."""
    server = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
    server.setsockopt(socket.SOL_SOCKET, socket.SO_REUSEADDR, 1)
    
    try:
        server.bind((HOST, PORT))
        server.listen(5)
    except Exception as e:
        print(f"Failed to bind server to {HOST}:{PORT}: {e}")
        return

    with stats_lock:
        server_stats["is_running"] = True
        server_stats["start_time"] = time.time()

    print(f"[SERVER STARTED] Listening on {HOST}:{PORT}...")

    try:
        while True:
            conn, addr = server.accept()
            
            # Instead of processing blocking code inline, spin up a NEW thread for this client
            client_thread = threading.Thread(
                target=handle_client, 
                args=(conn, addr),
                daemon=True # Dies automatically if the main program exits
            )
            client_thread.start()
            
    except Exception as e:
        print(f"Server main-loop encountered error: {e}")
    finally:
        with stats_lock:
            server_stats["is_running"] = False
        server.close()

if __name__ == "__main__":
    # Fallback to run directly if needed
    start_server()