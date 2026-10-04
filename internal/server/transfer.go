package server

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"os"
)

// sendLine writes a line terminated with \n (protocol replies use LF;
// legacy used \n via explicit encodes, matching here).
func sendLine(conn net.Conn, line string) error {
	_, err := fmt.Fprint(conn, line+"\n")
	return err
}

// sendFile streams path to conn. Returns bytes sent.
func sendFile(conn net.Conn, filePath string) (int64, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return 0, err
	}
	defer f.Close()
	st, err := f.Stat()
	if err != nil {
		return 0, err
	}
	if _, err := fmt.Fprintf(conn, "FOUND %d\n", st.Size()); err != nil {
		return 0, err
	}
	n, err := io.CopyBuffer(conn, f, make([]byte, 32*1024))
	return n, err
}

// recvExact reads exactly n bytes from r into a temp file at partPath.
func recvExact(r *bufio.Reader, partPath string, n int64) (int64, error) {
	f, err := os.Create(partPath)
	if err != nil {
		return 0, err
	}
	defer f.Close()
	var received int64
	buf := make([]byte, 32*1024)
	for received < n {
		want := int64(len(buf))
		if n-received < want {
			want = n - received
		}
		m, rerr := io.ReadFull(r, buf[:want])
		if m > 0 {
			if _, werr := f.Write(buf[:m]); werr != nil {
				return received, werr
			}
			received += int64(m)
		}
		if rerr != nil {
			if rerr == io.EOF || rerr == io.ErrUnexpectedEOF {
				break
			}
			return received, rerr
		}
	}
	if err := f.Sync(); err != nil {
		return received, err
	}
	return received, nil
}
