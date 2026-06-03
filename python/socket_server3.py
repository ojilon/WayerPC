import socket
from transfer import send_file
from config import *

import os
import ctypes

dll = ctypes.CDLL("C:\\Users\\ojilong\\Desktop\\StudyAssistant\\WayerPC_V.0.0.0\\build\\c\\libfilesearch.dll")

dll.search_file.argtypes = [ctypes.c_char_p, ctypes.c_char_p, ctypes.c_char_p, ctypes.c_size_t]
dll.search_file.restype = ctypes.c_int


def handle_client(conn):
    try:

        data = conn.recv(1024).decode('utf-8').strip()
        print("Received Command:", data)

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

            #mutable string buffer for C to write the path into
            project_root = "C:\\Users\\ojilong\\Desktop\\StudyAssistant\\WayerPC_V.0.0.0"
            path_buffer = ctypes.create_string_buffer(260)

            #call the c function
            result = dll.search_file(filename.encode('utf-8'),project_root.encode('utf-8'), path_buffer, 260)

            if result == 0: #a success
                filepath = path_buffer.value.decode('utf-8')

                try:
                    filesize = os.path.getsize(filepath)
                    #send metadata and immediately stream the file
                    msg = f"FOUND {filesize}\n"
                    conn.send(msg.encode())

                    #directly stream the file data
                    send_file(conn, filepath)

                except FileNotFoundError:
                    #catches cases where the C found but py lost permissions/access
                    conn.send(b"ERROR file_access_denied")
                    print(f"The denied path -> {filepath}")

            elif result == -1:#STATUS_DIR_ERROR
                conn.send(b"ERROR shared_file_directory_missing_or_unreadable")

            elif result == -2: #STATUS_NOT_FOUND
                conn.send(b"ERROR file_not_found")

            else:
                conn.send(b"ERROR funknown_system_fault")

        elif data.startswith("/upload"):
            # Expecting command format: "/upload filesize filename"
            parts = data.split(" ")
            if len(parts) < 3:
                conn.send(b"ERROR invalid_upload_command")
                return
            
            filename = parts[2].strip()
            filesize = int(parts[1].strip())
            
            # Ensure the target directory exists on your PC
            save_dir = "C:\\Users\\ojilong\\Desktop\\StudyAssistant\\WayerPC_V.0.0.0\\received"
            if not os.path.exists(save_dir):
                os.makedirs(save_dir)
                
            filepath = os.path.join(save_dir, filename)
            
            # Tell the phone the PC is ready to receive bytes
            conn.send(b"READY")
            
            # Start receiving binary stream
            print(f"Receiving {filename} ({filesize} bytes)...")
            with open(filepath, "wb") as f:
                remaining = filesize
                while remaining > 0:
                    # Read in chunks up to 4096 bytes
                    chunk = conn.recv(min(4096, remaining))
                    if not chunk:
                        break
                    f.write(chunk)
                    remaining -= len(chunk)
            
            if remaining == 0:
                print(f"Successfully saved {filename} to received/ folder!")
                conn.send(b"DONE")
            else:
                print("Upload interrupted: connection lost prematurely.")
                
        else:
            conn.send(b"ERROR unknown_protocol_command")                     

    except Exception as e:
        print(f"Server Exception: {e}")
    finally:
        conn.close()


def start_server():
    server = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
    # Allow instant socket address reuse to prevent "Address already in use" errors
    server.setsockopt(socket.SOL_SOCKET, socket.SO_REUSEADDR, 1)
    server.bind((HOST, PORT))
    server.listen(5)

    print(f"Listening on {HOST}:{PORT}")

    while True:
        conn, addr = server.accept()
        print("Connected:", addr)
        handle_client(conn)

if __name__ == "__main__":
    start_server()