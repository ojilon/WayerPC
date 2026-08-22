import socket
import os
import threading
import time
from .ExecuteServerCommand import Execute_server_command

# Configuration - HOST and PORT
HOST = "0.0.0.0"
PORT = 5000
BUFFER_SIZE = 4096

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
    "start_time": None,
}


def handle_client(conn, addr, logmessage=print):
    """Handles an individual client connection inside its own dedicated thread."""
    # Increment active connections safely using a Mutex (Lock)
    with stats_lock:
        server_stats["active_connections"] += 1
        server_stats["total_connections_handled"] += 1

    logmessage(f"\n[THREAD-{threading.get_ident()}] Handling connection from {addr}")

    try:
        # Set connection timeout so client can't hang forever
        conn.settimeout(300.0)

        while True:
            try:
                data = conn.recv(1024).decode("utf-8").strip()
                if not data:
                    # Client disconnected
                    logmessage(f"[CLIENT] {addr} disconnected")
                    break

                status, detail = Execute_server_command(
                    data, conn, server_stats, stats_lock
                )

                # detail is always a non-None string from Execute_server_command
                if detail is None:
                    # Defensive: should not happen after the return-value cleanup
                    logmessage(
                        f"[SERVER ERROR] {data!r} returned no detail (unexpected)"
                    )
                    break

                # Log according to status. Client-facing messages were already
                # sent inside Execute_server_command; we only log here.
                if status == 0:
                    # /ask success — file streamed
                    logmessage(f"[ASK OK] Sent file: {detail}")
                elif status == 3:
                    # /upload success — file saved under received/
                    logmessage(f"[UPLOAD OK] Received and saved: {detail}")
                elif status == 1:
                    # Protocol / validation / incomplete transfer
                    logmessage(f"[PROTOCOL] {detail}")
                elif status == 2:
                    # Storage / directory problem
                    logmessage(f"[STORAGE ERROR] {detail}")
                elif status == -1:
                    # Logical failure already reported to client (e.g. not found)
                    logmessage(f"[COMMAND FAIL] {detail}")
                else:
                    logmessage(
                        f"[SERVER] Unexpected status={status} for {data!r}: {detail}"
                    )

            except socket.timeout:
                logmessage(f"[TIMEOUT] Connection to {addr} timed out")
                break
            except Exception as e:
                logmessage(f"[HANDLER ERROR] {addr}: {e}...")
                break

    except Exception as e:
        logmessage(f"[THREAD EXCEPTION] {addr}: {e}...")
    finally:
        # Decrement active connections safely when the thread dies
        with stats_lock:
            server_stats["active_connections"] -= 1
        conn.close()
        logmessage(f"[THREAD-{threading.get_ident()}] Connection with {addr} closed.")


def start_server(logmessage=print):
    """Main loop accepting TCP connections. Runs inside a background thread."""
    server = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
    server.setsockopt(socket.SOL_SOCKET, socket.SO_REUSEADDR, 1)

    try:
        server.bind((HOST, PORT))
        server.listen(5)
    except Exception as e:
        logmessage(f"[SERVER] Failed to bind server to {HOST}:{PORT}: {e}")
        return

    with stats_lock:
        server_stats["is_running"] = True
        server_stats["start_time"] = time.time()

    logmessage(f"[SERVER] Listening on {HOST}:{PORT}...")

    try:
        while True:
            conn, addr = server.accept()

            # Spin up a NEW thread for this client
            client_thread = threading.Thread(
                target=handle_client,
                args=(conn, addr, logmessage),
                daemon=True,  # Dies automatically if the main program exits
            )
            client_thread.start()

    except Exception as e:
        logmessage(f"[SERVER] Server main-loop encountered error: {e}")
    finally:
        with stats_lock:
            server_stats["is_running"] = False
        server.close()


if __name__ == "__main__":
    start_server()
