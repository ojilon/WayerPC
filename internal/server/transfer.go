package server

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"os"
	"time"
)

// sendLine writes a line terminated with \n (protocol replies use LF;
// legacy used \n via explicit encodes, matching here).
func sendLine(conn net.Conn, line string) error {
	_, err := fmt.Fprint(conn, line+"\n")
	return err
}

// sendFile streams path to conn: "FOUND <size>\n" + raw bytes.
// The write deadline is refreshed per chunk, so a slow-but-progressing
// phone never trips the control writeTimeout mid-download.
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
	_ = conn.SetWriteDeadline(time.Now().Add(writeTimeout))
	if _, err := fmt.Fprintf(conn, "FOUND %d\n", st.Size()); err != nil {
		return 0, err
	}
	var sent int64
	buf := make([]byte, 32*1024)
	for {
		nr, rerr := f.Read(buf)
		if nr > 0 {
			_ = conn.SetWriteDeadline(time.Now().Add(writeTimeout))
			nw, werr := conn.Write(buf[:nr])
			sent += int64(nw)
			if werr != nil {
				return sent, werr
			}
			if nw != nr {
				return sent, io.ErrShortWrite
			}
		}
		if rerr == io.EOF {
			return sent, nil
		}
		if rerr != nil {
			return sent, rerr
		}
	}
}

// recvExact reads exactly n bytes from r into a temp file at partPath.
// The read deadline is refreshed per chunk received, so a slow-but-alive
// upload survives; a stall longer than idleTimeout aborts it.
func recvExact(r *bufio.Reader, conn net.Conn, partPath string, n int64) (int64, error) {
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
			// Progress! The peer is alive — slide the stall window.
			_ = conn.SetReadDeadline(time.Now().Add(idleTimeout))
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
