import socket
import threading
import time
from .ExecuteServerCommand import Execute_server_command

HOST = "0.0.0.0"
PORT = 5000
BUFFER_SIZE = 4096

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
    with stats_lock:
        server_stats["active_connections"] += 1
        server_stats["total_connections_handled"] += 1

    logmessage(f"\n[THREAD-{threading.get_ident()}] Handling connection from {addr}")

    try:
        conn.settimeout(300.0)

        while True:
            try:
                data = conn.recv(1024).decode("utf-8").strip()
                if not data:
                    logmessage(f"[CLIENT] {addr} disconnected")
                    break

                status, detail = Execute_server_command(
                    data, conn, server_stats, stats_lock
                )

                if detail is None:
                    logmessage(f"[SERVER ERROR] {data!r} returned no detail (unexpected)")
                    break

                if status == 0:
                    logmessage(f"[ASK OK] Sent file: {detail}")
                elif status == 3:
                    logmessage(f"[UPLOAD OK] Received and saved: {detail}")
                elif status == 4:
                    logmessage(f"[MATCHES] {detail}")
                elif status == 1:
                    logmessage(f"[PROTOCOL] {detail}")
                elif status == 2:
                    logmessage(f"[STORAGE ERROR] {detail}")
                elif status == -1:
                    logmessage(f"[COMMAND FAIL] {detail}")
                else:
                    logmessage(f"[SERVER] Unexpected status={status} for {data!r}: {detail}")

            except socket.timeout:
                logmessage(f"[TIMEOUT] Connection to {addr} timed out")
                break
            except Exception as e:
                logmessage(f"[HANDLER ERROR] {addr}: {e}...")
                break

    except Exception as e:
        logmessage(f"[THREAD EXCEPTION] {addr}: {e}...")
    finally:
        with stats_lock:
            server_stats["active_connections"] -= 1
        conn.close()
        logmessage(f"[THREAD-{threading.get_ident()}] Connection with {addr} closed.")


def start_server(logmessage=print):
    server = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
    server.setsockopt(socket.SOL_SOCKET, socket.SO_REUSEADDR, 1)

    try:
        server.bind((HOST, PORT))
        server.listen(5)
    except OSError as e:
        logmessage(f"[SERVER] Failed to bind server to {HOST}:{PORT}: {e}")
        return

    with stats_lock:
        server_stats["is_running"] = True
        server_stats["start_time"] = time.time()

    logmessage(f"[SERVER] Listening on {HOST}:{PORT}...")

    try:
        while True:
            conn, addr = server.accept()
            client_thread = threading.Thread(
                target=handle_client,
                args=(conn, addr, logmessage),
                daemon=True,
            )
            client_thread.start()
    except OSError as e:
        logmessage(f"[SERVER] Server main-loop encountered error: {e}")
    finally:
        with stats_lock:
            server_stats["is_running"] = False
        server.close()


if __name__ == "__main__":
    start_server()
