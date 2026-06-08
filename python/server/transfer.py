def send_file(conn, filepath):
    with open(filepath, "rb") as f:

        while True:
            data = f.read(4096)

            if not data:
                break

            conn.sendall(data)

    print("File sent.")
