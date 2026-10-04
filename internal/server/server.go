// Package server is the phone ↔ PC TCP server (port of server4.py).
//
// Wire protocol (byte-compatible with the Python server):
//
//	/ask <name>             → FOUND <size>\n + raw bytes
//	                        → MATCHES <n>\n<name>\n… (related names)
//	                        → ERROR file_not_found | ERROR file_access_denied
//	/upload <size> <name>   → READY … raw bytes … DONE (saved to received/)
//	/upload <query>         → MATCHES <n>\n<name>\n… (catalog search only)
package server

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"sync"
	"time"

	"wayerpc/internal/catalog"
	"wayerpc/internal/storage"
)

const (
	// StatusAskOK mirrors Python status 0 (file streamed to phone).
	StatusAskOK = 0
	// StatusUploadOK mirrors Python status 3 (upload saved).
	StatusUploadOK = 3
	// StatusMatches mirrors Python status 4 (MATCHES list sent).
	StatusMatches = 4
	// StatusProtocol mirrors Python status 1 (protocol/validation).
	StatusProtocol = 1
	// StatusStorage mirrors Python status 2 (storage problem).
	StatusStorage = 2
	// StatusFail mirrors Python status -1 (failure already sent to client).
	StatusFail = -1

	// idleTimeout is the max silence between phone commands (mirrors the
	// Python conn.settimeout(300.0)). It is refreshed on every chunk of an
	// active transfer, so slow-but-alive uploads/downloads survive while
	// truly stalled ones still time out.
	idleTimeout = 300 * time.Second
	// writeTimeout bounds control replies (READY/DONE/FOUND/MATCHES/ERROR).
	// Bulk transfer chunks refresh their own deadline per chunk (see
	// transfer.go), so this never aborts a progressing transfer.
	writeTimeout = 60 * time.Second
	// maxLine is the longest command line accepted (legacy recv(1024) was
	// smaller; 64 KiB keeps long filenames safe without unbounded growth).
	maxLine = 64 * 1024
	// DefaultMaxUploadBytes caps one upload (legacy had no cap).
	DefaultMaxUploadBytes = int64(2 << 30)
)

// Logger receives human-readable server lines (wired to the UI log).
type Logger func(line string)

// Stats is a point-in-time snapshot for the Console tab.
type Stats struct {
	Running       bool   `json:"running"`
	Addr          string `json:"addr"`
	Active        int    `json:"active"`
	Total         int64  `json:"total"`
	BytesSent     int64  `json:"bytesSent"`
	BytesReceived int64  `json:"bytesReceived"`
	StartTime     int64  `json:"startTimeUnix"`
	UptimeSecs    int64  `json:"uptimeSecs"`
}

// Server is a TCP phone-link server. Zero value is not usable; use New.
type Server struct {
	host      string
	port      int
	store     *storage.Store
	cat       *catalog.DB
	logf      Logger
	maxUpload int64

	mu            sync.Mutex
	listener      net.Listener
	running       bool
	active        int
	total         int64
	bytesSent     int64
	bytesReceived int64
	startTime     time.Time

	// backendMu guards store/cat so the app can relocate the data root
	// (Settings screen / installer) without restarting the listener.
	backendMu sync.RWMutex
}

// New builds a server. logf may be nil (lines are dropped).
func New(host string, port int, store *storage.Store, cat *catalog.DB, logf Logger) *Server {
	if logf == nil {
		logf = func(string) {}
	}
	return &Server{
		host: host, port: port,
		store: store, cat: cat, logf: logf,
		maxUpload: DefaultMaxUploadBytes,
	}
}

// Addr returns "host:port".
func (s *Server) Addr() string { return fmt.Sprintf("%s:%d", s.host, s.port) }

// SetMaxUploadBytes overrides the per-upload cap (tests use a small value).
func (s *Server) SetMaxUploadBytes(n int64) { s.maxUpload = n }

// Attach hot-swaps the storage + catalog backends (data-root relocate).
func (s *Server) Attach(store *storage.Store, cat *catalog.DB) {
	s.backendMu.Lock()
	defer s.backendMu.Unlock()
	s.store = store
	s.cat = cat
}

// backends returns the current storage + catalog handles.
func (s *Server) backends() (*storage.Store, *catalog.DB) {
	s.backendMu.RLock()
	defer s.backendMu.RUnlock()
	return s.store, s.cat
}

// Start binds and serves until Stop is called (or bind fails).
func (s *Server) Start() error {
	ln, err := net.Listen("tcp", s.Addr())
	if err != nil {
		s.logf(fmt.Sprintf("[SERVER] Failed to bind server to %s: %v", s.Addr(), err))
		return err
	}
	s.mu.Lock()
	s.listener = ln
	s.running = true
	s.startTime = time.Now()
	s.mu.Unlock()
	s.logf(fmt.Sprintf("[SERVER] Listening on %s...", ln.Addr()))

	for {
		conn, err := ln.Accept()
		if err != nil {
			s.mu.Lock()
			s.running = false
			s.mu.Unlock()
			return nil // closed via Stop
		}
		s.mu.Lock()
		s.active++
		s.total++
		s.mu.Unlock()
		go s.serveConn(conn)
	}
}

// Stop shuts the listener down; running connections drain.
func (s *Server) Stop() error {
	s.mu.Lock()
	ln := s.listener
	s.listener = nil
	s.running = false
	s.mu.Unlock()
	if ln == nil {
		return nil
	}
	return ln.Close()
}

// Snapshot returns current stats for the Console tab.
func (s *Server) Snapshot() Stats {
	s.mu.Lock()
	defer s.mu.Unlock()
	uptime := int64(0)
	if s.running && !s.startTime.IsZero() {
		uptime = int64(time.Since(s.startTime).Seconds())
	}
	return Stats{
		Running: s.running, Addr: s.Addr(),
		Active: s.active, Total: s.total,
		BytesSent: s.bytesSent, BytesReceived: s.bytesReceived,
		StartTime: s.startTime.Unix(), UptimeSecs: uptime,
	}
}

func (s *Server) addSent(n int64) {
	s.mu.Lock()
	s.bytesSent += n
	s.mu.Unlock()
}

func (s *Server) addReceived(n int64) {
	s.mu.Lock()
	s.bytesReceived += n
	s.mu.Unlock()
}

func (s *Server) serveConn(conn net.Conn) {
	defer func() {
		s.mu.Lock()
		s.active--
		s.mu.Unlock()
		conn.Close()
	}()
	remote := conn.RemoteAddr().String()
	s.logf(fmt.Sprintf("[CONNECT] Handling connection from %s", remote))

	r := bufio.NewReaderSize(conn, 32*1024)
	for {
		// Per-operation deadlines, refreshed every command: an absolute
		// deadline set once per iteration could expire between the read
		// and the reply (or mid-transfer), killing live sessions with
		// "i/o timeout". Transfers refresh these per chunk instead.
		_ = conn.SetReadDeadline(time.Now().Add(idleTimeout))
		_ = conn.SetWriteDeadline(time.Now().Add(writeTimeout))
		line, err := readCommand(r)
		if err != nil {
			if err == io.EOF {
				s.logf(fmt.Sprintf("[CLIENT] %s disconnected", remote))
			} else {
				s.logf(fmt.Sprintf("[TIMEOUT] Connection to %s closed: %v", remote, err))
			}
			return
		}
		if line == "" {
			continue
		}
		status, detail, fatal := s.handleCommand(line, conn, r)
		switch status {
		case StatusAskOK:
			s.logf(fmt.Sprintf("[ASK OK] Sent file: %s", detail))
		case StatusUploadOK:
			s.logf(fmt.Sprintf("[UPLOAD OK] Received and saved: %s", detail))
		case StatusMatches:
			s.logf(fmt.Sprintf("[MATCHES] %s", detail))
		case StatusProtocol:
			s.logf(fmt.Sprintf("[PROTOCOL] %s", detail))
		case StatusStorage:
			s.logf(fmt.Sprintf("[STORAGE ERROR] %s", detail))
		case StatusFail:
			s.logf(fmt.Sprintf("[COMMAND FAIL] %s", detail))
		default:
			s.logf(fmt.Sprintf("[SERVER] Unexpected status=%d for %q: %s", status, line, detail))
		}
		if fatal {
			// The reply could not be delivered — the peer is gone.
			// Close now instead of lingering until the next deadline.
			s.logf(fmt.Sprintf("[DISCONNECT] Closing %s (undeliverable reply)", remote))
			return
		}
	}
}

// readCommand reads one trimmed command line (without the newline).
func readCommand(r *bufio.Reader) (string, error) {
	var line []byte
	for {
		chunk, isPrefix, err := r.ReadLine()
		if err != nil {
			if err == io.EOF && len(line) > 0 {
				break // last line without newline
			}
			return "", err
		}
		if len(line)+len(chunk) > maxLine {
			return "", fmt.Errorf("command line too long")
		}
		line = append(line, chunk...)
		if !isPrefix {
			break
		}
	}
	// Trim trailing \r and surrounding whitespace (legacy .strip()).
	out := string(line)
	for len(out) > 0 && (out[len(out)-1] == '\r' || out[len(out)-1] == ' ' || out[len(out)-1] == '\t') {
		out = out[:len(out)-1]
	}
	for len(out) > 0 && (out[0] == ' ' || out[0] == '\t') {
		out = out[1:]
	}
	return out, nil
}
