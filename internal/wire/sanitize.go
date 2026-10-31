package wire

import (
	"errors"
	"fmt"
	"strings"
)

// ErrBadName is returned for names that must never reach the filesystem.
var ErrBadName = errors.New("wire: unsafe file name")

// reservedNames are the Windows device names that cannot be used as a file
// name, with or without an extension ("CON", "con.txt", "LPT1.log", …).
var reservedNames = map[string]bool{
	"con": true, "prn": true, "aux": true, "nul": true,
	"com1": true, "com2": true, "com3": true, "com4": true,
	"com5": true, "com6": true, "com7": true, "com8": true, "com9": true,
	"lpt1": true, "lpt2": true, "lpt3": true, "lpt4": true,
	"lpt5": true, "lpt6": true, "lpt7": true, "lpt8": true, "lpt9": true,
}

// SanitizeName turns an untrusted, phone-supplied name into a safe basename:
// directory components are dropped, traversal, separators, control characters
// and Windows reserved device names are rejected outright. Callers still join
// the result onto their own directory (never trust this as a path).
func SanitizeName(raw string) (string, error) {
	if strings.Contains(raw, "..") {
		// Reject traversal attempts instead of silently basenaming them away.
		return "", fmt.Errorf("%w: traversal in %q", ErrBadName, raw)
	}
	for _, r := range raw {
		if r < 0x20 || r == 0x7f {
			return "", fmt.Errorf("%w: control character", ErrBadName)
		}
	}
	name := strings.TrimSpace(raw)
	if name == "" {
		return "", fmt.Errorf("%w: empty", ErrBadName)
	}
	// Drop any directory component, whichever separator the phone used.
	name = name[strings.LastIndexAny(name, `/\`)+1:]
	if name == "" || name == "." || name == ".." {
		return "", fmt.Errorf("%w: %q", ErrBadName, raw)
	}
	if strings.ContainsAny(name, `/\`) {
		return "", fmt.Errorf("%w: separator in %q", ErrBadName, raw)
	}
	if strings.HasPrefix(name, ".") {
		// Keeps internal folders (.partial) unreachable from the wire.
		return "", fmt.Errorf("%w: hidden name %q", ErrBadName, name)
	}
	if strings.HasSuffix(name, ".") || strings.HasSuffix(name, " ") {
		return "", fmt.Errorf("%w: trailing dot/space", ErrBadName)
	}
	if len(name) > 200 {
		return "", fmt.Errorf("%w: too long", ErrBadName)
	}
	base := name
	if i := strings.IndexByte(base, '.'); i > 0 {
		base = base[:i]
	}
	if reservedNames[strings.ToLower(base)] {
		return "", fmt.Errorf("%w: reserved device name %q", ErrBadName, name)
	}
	for _, r := range name {
		switch r {
		case '<', '>', ':', '"', '|', '?', '*':
			return "", fmt.Errorf("%w: illegal character %q", ErrBadName, r)
		}
	}
	return name, nil
}

// SanitizePath is SanitizeName for manifest paths ("trip-plan.pdf"): the
// phone may send a flat relative path, but never a traversal.
func SanitizePath(raw string) (string, error) {
	return SanitizeName(raw)
}
