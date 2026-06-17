import threading
import time
import os
import shutil
# Import our background server logic and state variables
from python.server.server4 import start_server, server_stats, stats_lock
from python.server.ItemImporter import pull_file 



def monitorboard():
    """Reads shared memory stats safely using locks to display a UI dashboard."""
    with stats_lock:
        running = server_stats["is_running"]
        active = server_stats["active_connections"]
        total = server_stats["total_connections_handled"]
        sent = server_stats["bytes_sent"]
        received = server_stats["bytes_received"]
        start = server_stats["start_time"]

    if not running:
        print("\n--- SERVER STATUS: OFFLINE ---")
        return

    uptime = time.time() - start if start else 0
    print("\n==========================================")
    print("        SERVER MONITORING BOARD       ")
    print("==========================================")
    print(" Status:           RUNNING")
    print(f" Uptime:           {uptime:.2f} seconds")
    print(f" Active Links:     {active} concurrent devices")
    print(f" Total Handled:    {total} clients")
    print(f" Total Outbound:   {sent / (1024 * 1024):.2f} MB")
    print(f" Total Inbound:    {received / (1024 * 1024):.2f} MB")
    print("==========================================")

def main():
    print("WayerPC Init!...")

    # 1. Start the server socket listener in a BACKGROUND thread
    server_worker = threading.Thread(target=start_server, name="SocketServerThread")
    server_worker.daemon = True # This ensures when main.py closes, the server thread closes too
    server_worker.start()

    # Give the server thread a brief moment to boot up and bind to its port
    time.sleep(0.5)

    # 2. Main Interactive Loop (The Frontend CLI)
    while True:
        print("\n[Main Menu] Options: \n(1) Check Status \n(2) Import File \n(3) Exit")
        
        choice = input("Enter choice: ").strip()

        if choice == "1":
            monitorboard()

        elif choice == "2":
            # Running independent file copier routine without freezing the socket server            
            # Running in another separate thread so even the CLI menu doesn't freeze!
            task_thread = threading.Thread(
                target=pull_file(),
                daemon=True
            )
            task_thread.start()

        elif choice == "3":
            print("Shutting down....")
            break
        else:
            print("Unknown selection.")

if __name__ == "__main__":
    main()