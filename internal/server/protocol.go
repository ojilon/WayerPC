package server

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"

	"wayerpc/internal/catalog"
	"wayerpc/internal/search"
)

// handleCommand implements one protocol line. Always returns
// (status, detail, fatal); error replies are already written to conn.
// fatal=true means the reply could not be delivered (peer gone) and the
// caller should close the session immediately.
func (s *Server) handleCommand(line string, conn net.Conn, r *bufio.Reader) (int, string, bool) {
	if strings.HasPrefix(line, "/ask") {
		rest := strings.TrimSpace(strings.TrimPrefix(line, "/ask"))
		if rest == "" {
			_ = sendLine(conn, "ERROR invalid_command")
			return StatusProtocol, "invalid /ask command (missing filename)", false
		}
		// Tolerate a duplicated prefix ("/ask /ask file") from the phone.
		if strings.HasPrefix(rest, "/ask") {
			rest = strings.TrimSpace(strings.TrimPrefix(rest, "/ask"))
		}
		if rest == "" {
			_ = sendLine(conn, "ERROR invalid_command")
			return StatusProtocol, "invalid /ask command (missing filename)", false
		}
		return s.resolveAndSend(rest, conn)
	}

	if strings.HasPrefix(line, "/upload") {
		return s.handleUpload(line, conn, r)
	}

	_ = sendLine(conn, "ERROR unknown_protocol_command.")
	return StatusProtocol, fmt.Sprintf("unknown protocol command: %q", line), false
}

// resolveAndSend serves an /ask query: catalog first (20%+ related names),
// then the exact-name fallback over shared/ + received/.
func (s *Server) resolveAndSend(query string, conn net.Conn) (int, string, bool) {
	_, cat := s.backends()
	var hits []catalog.Entry
	if cat != nil {
		var err error
		hits, err = cat.FindForRequest(query, catalog.DefaultCutoff)
		if err != nil {
			hits = nil
		}
	}

	lower := strings.ToLower(query)
	var exact []catalog.Entry
	for _, h := range hits {
		if strings.ToLower(h.Name) == lower {
			exact = append(exact, h)
		}
	}
	if len(exact) == 1 {
		return s.streamFile(conn, exact[0].Path)
	}
	if len(hits) == 1 && hits[0].Score >= 0.8 {
		return s.streamFile(conn, hits[0].Path)
	}
	if len(hits) > 1 || (len(hits) == 1 && hits[0].Score < 0.8) {
		return s.sendMatches(conn, hits)
	}

	found := search.SearchAppFolders(query, s.appFolders())
	if len(found) > 0 {
		return s.streamFile(conn, found[0])
	}
	_ = sendLine(conn, "ERROR file_not_found")
	return StatusFail, fmt.Sprintf("file not found: %s", query), false
}

// handleUpload serves both /upload forms:
//   - "/upload <size> <name>" → binary receive into received/
//   - "/upload <query>"       → catalog MATCHES search (no numeric size)
func (s *Server) handleUpload(line string, conn net.Conn, r *bufio.Reader) (int, string, bool) {
	parts := strings.Split(line, " ")
	if len(parts) < 2 {
		_ = sendLine(conn, "ERROR invalid_upload_command")
		return StatusProtocol, "invalid /upload command", false
	}
	sizeToken := strings.TrimSpace(parts[1])
	if !isDigits(sizeToken) || len(parts) < 3 {
		query := strings.TrimSpace(strings.TrimPrefix(line, "/upload"))
		if query == "" {
			_ = sendLine(conn, "ERROR invalid_upload_command")
			return StatusProtocol, "invalid /upload command (need size+name or a search query)", false
		}
		_, cat := s.backends()
		var hits []catalog.Entry
		if cat != nil {
			var err error
			hits, err = cat.FindForRequest(query, catalog.DefaultCutoff)
			if err != nil {
				hits = nil
			}
		}
		if len(hits) == 0 {
			_ = sendLine(conn, "ERROR file_not_found")
			return StatusFail, fmt.Sprintf("no imported files matching %q", query), false
		}
		return s.sendMatches(conn, hits)
	}

	var filesize int64
	if _, err := fmt.Sscanf(sizeToken, "%d", &filesize); err != nil || filesize < 0 {
		_ = sendLine(conn, "ERROR invalid_upload_command")
		return StatusProtocol, fmt.Sprintf("invalid filesize in /upload: %q", sizeToken), false
	}
	if filesize > s.maxUpload {
		_ = sendLine(conn, "ERROR upload_too_large")
		return StatusProtocol, fmt.Sprintf("upload too large: %d bytes", filesize), false
	}

	// Join the rest so names with spaces work (legacy took parts[2] only).
	filename := filepath.Base(strings.TrimSpace(strings.Join(parts[2:], " ")))
	if filename == "" || filename == "." || filename == "/" {
		_ = sendLine(conn, "ERROR invalid_upload_command")
		return StatusProtocol, "empty filename in /upload", false
	}

	saveDir := ""
	if store, _ := s.backends(); store != nil {
		saveDir = store.ReceivedDir()
	}
	if saveDir == "" {
		_ = sendLine(conn, "ERROR: Failed to obtain directory this side, to store the file to receive")
		return StatusStorage, "received folder unavailable (storage not initialized)", false
	}
	if err := os.MkdirAll(saveDir, 0o755); err != nil {
		_ = sendLine(conn, "ERROR: Failed to obtain directory this side, to store the file to receive")
		return StatusStorage, fmt.Sprintf("received folder unavailable: %v", err), false
	}
	finalPath := filepath.Join(saveDir, filename)
	partPath := finalPath + ".part"

	if _, err := fmt.Fprint(conn, "READY"); err != nil {
		return StatusProtocol, fmt.Sprintf(
			"failed to send READY for upload of %q: %v (phone likely asleep/disconnected — it can retry on reconnect)",
			filename, err), true
	}

	received, werr := recvExact(r, conn, partPath, filesize)
	if werr != nil {
		_ = os.Remove(partPath)
		fatal := sendLine(conn, "ERROR write_failed") != nil
		return StatusStorage, fmt.Sprintf("write failed for %q: %v", filename, werr), fatal
	}
	if received != filesize {
		_ = os.Remove(partPath)
		fatal := sendLine(conn, "ERROR upload_incomplete") != nil
		return StatusProtocol, fmt.Sprintf("upload incomplete for %q (%d/%d bytes)", filename, received, filesize), fatal
	}
	if err := os.Rename(partPath, finalPath); err != nil {
		_ = os.Remove(partPath)
		fatal := sendLine(conn, "ERROR write_failed") != nil
		return StatusStorage, fmt.Sprintf("write failed for %q: %v", filename, err), fatal
	}
	_ = sendLine(conn, "DONE")
	s.addReceived(filesize)
	if _, cat := s.backends(); cat != nil {
		_, _ = cat.AddPath(finalPath)
	}
	return StatusUploadOK, filename, false
}

// streamFile sends FOUND <size>\n + bytes. Mirrors _stream_file.
func (s *Server) streamFile(conn net.Conn, filePath string) (int, string, bool) {
	size, err := sendFile(conn, filePath)
	if err != nil {
		fatal := sendLine(conn, "ERROR file_access_denied") != nil
		return StatusFail, fmt.Sprintf("file not accessible: %s (%v)", filePath, err), fatal
	}
	s.addSent(size)
	return StatusAskOK, filepath.Base(filePath), false
}

// sendMatches sends "MATCHES <n>\n<name>\n…" (names only, max 50 — legacy parity).
func (s *Server) sendMatches(conn net.Conn, hits []catalog.Entry) (int, string, bool) {
	n := len(hits)
	if n > 50 {
		hits = hits[:50]
	}
	var b strings.Builder
	fmt.Fprintf(&b, "MATCHES %d\n", n)
	for _, h := range hits {
		b.WriteString(h.Name)
		b.WriteByte('\n')
	}
	if _, err := conn.Write([]byte(b.String())); err != nil {
		return StatusProtocol, "failed to send MATCHES list", true
	}
	preview := make([]string, 0, 8)
	for i, h := range hits {
		if i >= 8 {
			break
		}
		preview = append(preview, h.Name)
	}
	return StatusMatches, fmt.Sprintf("related names (%d): %s", n, strings.Join(preview, ", ")), false
}

func (s *Server) appFolders() []string {
	store, _ := s.backends()
	if store == nil {
		return nil
	}
	return []string{store.SharedDir(), store.ReceivedDir()}
}

func isDigits(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}
