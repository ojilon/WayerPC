// Package wire implements the protocol-v2 framing and message types.
//
// v2 is opt-in and recognised by the FIRST byte of a connection: '@' starts a
// v2 frame, 'F' starts a v1 line ("/ask …"). Frame = "@<byteLen>\n" followed
// by exactly byteLen bytes of UTF-8 JSON. Raw file bytes only ever follow a
// frame that declares a size.
//
// Nothing in this package touches the filesystem or the v1 protocol.
package wire

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"strconv"
	"strings"
	"time"
)

const (
	// Marker is the first byte of every v2 frame.
	Marker = '@'
	// MaxFrame caps one JSON frame (4 MiB).
	MaxFrame = 4 << 20
	// ProtoVersion is the protocol this package speaks.
	ProtoVersion = 2
	// DefaultPing is the control-link keepalive period in seconds.
	DefaultPing = 20
	// PongTimeout drops a control link with no pong for this long.
	PongTimeout = 60 * time.Second
	// QueryTTL is how long a numbered match list stays valid.
	QueryTTL = 120 * time.Second
	// MaxMatches caps one numbered match list.
	MaxMatches = 50
	// MaxDataConns caps concurrent data connections.
	MaxDataConns = 4
	// MaxUploadBytes mirrors the v1 upload cap (2 GiB).
	MaxUploadBytes = int64(2 << 30)
)

// Message type names.
const (
	THello       = "hello"
	THelloOK     = "hello_ok"
	TQuery       = "query"
	TMatches     = "matches"
	TPick        = "pick"
	TAbort       = "abort"
	TPush        = "push"
	TReady       = "ready"
	TDone        = "done"
	TFile        = "file"
	TOffer       = "offer"
	TAccept      = "accept"
	TReject      = "reject"
	TPull        = "pull"
	TGet         = "get"
	TManifestReq = "manifest_req"
	TManifest    = "manifest"
	TPing        = "ping"
	TPong        = "pong"
	TError       = "error"
	TOk          = "ok"
)

// Error codes.
const (
	CodeNotFound     = "not_found"
	CodeExpired      = "expired"
	CodeAborted      = "aborted"
	CodeBusy         = "busy"
	CodeDenied       = "denied"
	CodeTooLarge     = "too_large"
	CodeHashMismatch = "hash_mismatch"
	CodeBadRequest   = "bad_request"
	CodeUnsupported  = "unsupported"
	CodeInternal     = "internal"
)

// ErrFrameTooLarge is returned when a peer announces more than MaxFrame.
var ErrFrameTooLarge = errors.New("wire: frame exceeds maxFrame")

// Hello is the mandatory first message of every v2 connection.
type Hello struct {
	T      string   `json:"t"`
	Proto  int      `json:"proto"`
	Role   string   `json:"role"` // "control" | "data"
	Device string   `json:"device,omitempty"`
	DID    string   `json:"did,omitempty"`
	Caps   []string `json:"caps,omitempty"`
}

// HelloOK answers a Hello.
type HelloOK struct {
	T        string   `json:"t"`
	Proto    int      `json:"proto"`
	PC       string   `json:"pc"`
	Caps     []string `json:"caps"`
	MaxFrame int      `json:"maxFrame"`
	Ping     int      `json:"ping"`
}

// Item is one numbered match (phone → PC or PC → phone).
type Item struct {
	N     int    `json:"n"`
	Name  string `json:"name"`
	Size  int64  `json:"size"`
	MTime int64  `json:"mtime"`
	SHA   string `json:"sha256,omitempty"`
}

// Query asks the peer for matches. PC → phone and phone → PC share the shape.
type Query struct {
	T   string `json:"t"`
	ID  string `json:"id"`
	Q   string `json:"q"`
	Max int    `json:"max,omitempty"`
}

// Matches carries a numbered result list (TTL is only sent by the PC).
type Matches struct {
	T     string `json:"t"`
	ID    string `json:"id"`
	TTL   int    `json:"ttl,omitempty"`
	Items []Item `json:"items"`
}

// Pick selects one number from the previous Matches.
type Pick struct {
	T  string `json:"t"`
	ID string `json:"id"`
	N  int    `json:"n"`
}

// Abort cancels a query.
type Abort struct {
	T  string `json:"t"`
	ID string `json:"id"`
}

// Push announces incoming bytes on a data connection (phone → PC).
type Push struct {
	T      string `json:"t"`
	ID     string `json:"id"`
	Name   string `json:"name"`
	Size   int64  `json:"size"`
	MTime  int64  `json:"mtime,omitempty"`
	SHA    string `json:"sha256,omitempty"`
	Offset int64  `json:"offset"`
}

// Ready answers a Push with the offset the receiver already has.
type Ready struct {
	T      string `json:"t"`
	ID     string `json:"id"`
	Offset int64  `json:"offset"`
}

// File announces bytes the PC is about to send (phone-initiated query).
type File struct {
	T     string `json:"t"`
	ID    string `json:"id"`
	Name  string `json:"name"`
	Size  int64  `json:"size"`
	MTime int64  `json:"mtime,omitempty"`
	SHA   string `json:"sha256,omitempty"`
}

// Pull asks the PC to start sending an accepted offer.
type Pull struct {
	T      string `json:"t"`
	ID     string `json:"id"`
	Offset int64  `json:"offset"`
}

// Offer proposes a file to the phone.
type Offer struct {
	T     string `json:"t"`
	ID    string `json:"id"`
	Name  string `json:"name"`
	Size  int64  `json:"size"`
	MTime int64  `json:"mtime,omitempty"`
	SHA   string `json:"sha256,omitempty"`
}

// Accept / Reject answer an Offer.
type Accept struct {
	T  string `json:"t"`
	ID string `json:"id"`
}

type Reject struct {
	T    string `json:"t"`
	ID   string `json:"id"`
	Code string `json:"code"`
}

// Done closes a transfer, carrying the byte count and the sender's hash.
type Done struct {
	T     string `json:"t"`
	ID    string `json:"id"`
	Bytes int64  `json:"bytes"`
	SHA   string `json:"sha256,omitempty"`
}

// Get asks the phone for one known file by path/name.
type Get struct {
	T    string `json:"t"`
	ID   string `json:"id"`
	Path string `json:"path"`
}

// ManifestReq asks for a manifest of a phone scope.
type ManifestReq struct {
	T     string `json:"t"`
	ID    string `json:"id"`
	Scope string `json:"scope"`
}

// ManifestItem is one phone file in a manifest.
type ManifestItem struct {
	P     string `json:"p"`
	Size  int64  `json:"size"`
	MTime int64  `json:"mtime"`
	SHA   string `json:"sha256,omitempty"`
}

// Manifest is one part of a manifest response.
type Manifest struct {
	T     string         `json:"t"`
	ID    string         `json:"id"`
	Part  int            `json:"part"`
	Last  bool           `json:"last"`
	Items []ManifestItem `json:"items"`
}

// Ping / Pong keep the control link alive.
type Ping struct {
	T string `json:"t"`
}

type Pong struct {
	T string `json:"t"`
}

// Error is the uniform failure message.
type Error struct {
	T    string `json:"t"`
	ID   string `json:"id,omitempty"`
	Code string `json:"code"`
	Msg  string `json:"msg,omitempty"`
}

// Ok acknowledges an abort.
type Ok struct {
	T  string `json:"t"`
	ID string `json:"id"`
}

// Conn is a v2 connection: framing + typed helpers over a net.Conn.
type Conn struct {
	rwc net.Conn
	br  *bufio.Reader

	// ReadDeadline/WriteDeadline are applied per operation by callers.
}

// NewConn wraps a connection (buffered reader shared with the caller so the
// first byte used for v1/v2 detection is not lost).
func NewConn(c net.Conn, br *bufio.Reader) *Conn {
	if br == nil {
		br = bufio.NewReaderSize(c, 64*1024)
	}
	return &Conn{rwc: c, br: br}
}

// Reader exposes the buffered reader (for the v1 detector).
func (c *Conn) Reader() *bufio.Reader { return c.br }

// Net returns the underlying connection.
func (c *Conn) Net() net.Conn { return c.rwc }

// IsV2 sniffs the first byte without consuming it.
func IsV2(br *bufio.Reader) (bool, error) {
	b, err := br.Peek(1)
	if err != nil {
		return false, err
	}
	return b[0] == Marker, nil
}

// WriteFrame writes "@<len>\n" + payload in one call where possible.
func (c *Conn) WriteFrame(payload []byte) error {
	if len(payload) > MaxFrame {
		return ErrFrameTooLarge
	}
	header := "@" + strconv.Itoa(len(payload)) + "\n"
	buf := make([]byte, 0, len(header)+len(payload))
	buf = append(buf, header...)
	buf = append(buf, payload...)
	_, err := c.rwc.Write(buf)
	return err
}

// WriteJSON marshals v and writes it as one frame.
func (c *Conn) WriteJSON(v any) error {
	payload, err := json.Marshal(v)
	if err != nil {
		return err
	}
	return c.WriteFrame(payload)
}

// ReadFrame reads one frame and returns its payload.
func (c *Conn) ReadFrame() ([]byte, error) {
	first, err := c.br.ReadByte()
	if err != nil {
		return nil, err
	}
	if first != Marker {
		return nil, fmt.Errorf("wire: expected frame marker '@', got %q", first)
	}
	// Length line: digits until '\n'.
	var digits []byte
	for {
		b, err := c.br.ReadByte()
		if err != nil {
			return nil, err
		}
		if b == '\n' {
			break
		}
		if b < '0' || b > '9' || len(digits) > 12 {
			return nil, fmt.Errorf("wire: bad frame length")
		}
		digits = append(digits, b)
	}
	n, err := strconv.Atoi(string(digits))
	if err != nil || n < 0 {
		return nil, fmt.Errorf("wire: bad frame length")
	}
	if n > MaxFrame {
		return nil, ErrFrameTooLarge
	}
	payload := make([]byte, n)
	if _, err := io.ReadFull(c.br, payload); err != nil {
		return nil, err
	}
	return payload, nil
}

// ReadMessage reads one frame and returns its type + raw JSON.
func (c *Conn) ReadMessage() (string, []byte, error) {
	payload, err := c.ReadFrame()
	if err != nil {
		return "", nil, err
	}
	var envelope struct {
		T string `json:"t"`
	}
	if err := json.Unmarshal(payload, &envelope); err != nil {
		return "", nil, fmt.Errorf("wire: invalid json: %w", err)
	}
	if strings.TrimSpace(envelope.T) == "" {
		return "", nil, fmt.Errorf("wire: frame without type")
	}
	return envelope.T, payload, nil
}

// Decode unmarshals a raw message payload into v.
func Decode(payload []byte, v any) error { return json.Unmarshal(payload, v) }

// ReadRaw copies exactly n bytes from the connection into w, calling onChunk
// after each chunk (progress + deadlines). It returns the bytes copied.
func (c *Conn) ReadRaw(w io.Writer, n int64, onChunk func(int64)) (int64, error) {
	buf := make([]byte, 32*1024)
	var done int64
	for done < n {
		want := int64(len(buf))
		if n-done < want {
			want = n - done
		}
		m, err := io.ReadFull(c.br, buf[:want])
		if m > 0 {
			if _, werr := w.Write(buf[:m]); werr != nil {
				return done, werr
			}
			done += int64(m)
			if onChunk != nil {
				onChunk(int64(m))
			}
		}
		if err != nil {
			return done, err
		}
	}
	return done, nil
}

// WriteRaw streams r to the connection, calling onChunk after each chunk.
func (c *Conn) WriteRaw(r io.Reader, onChunk func(int64)) (int64, error) {
	buf := make([]byte, 32*1024)
	var sent int64
	for {
		nr, rerr := r.Read(buf)
		if nr > 0 {
			nw, werr := c.rwc.Write(buf[:nr])
			sent += int64(nw)
			if nw > 0 && onChunk != nil {
				onChunk(int64(nw))
			}
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
