// Package version is the single source of truth for app versioning.
// Edit version.json at the repo root BEFORE building; everything else
// (Go binary, frontend, installer) is generated/embedded from it.
package version

import (
	_ "embed"
	"encoding/json"
	"fmt"
)

//go:embed version.json
var raw []byte

// Info mirrors version.json.
type Info struct {
	AppName         string `json:"appName"`
	Version         string `json:"version"`
	ProtocolVersion int    `json:"protocolVersion"`
	Description     string `json:"description"`
	// BuildDate is stamped at build time via -ldflags when available.
	BuildDate string `json:"buildDate,omitempty"`
}

// Current parses the embedded version.json.
func Current() Info {
	var in Info
	if err := json.Unmarshal(raw, &in); err != nil {
		return Info{AppName: "WayerPC", Version: "0.0.0-dev"}
	}
	return in
}

// String returns "WayerPC v0.2.0 (proto 1)".
func (i Info) String() string {
	if i.BuildDate != "" {
		return fmt.Sprintf("%s v%s (proto %d, built %s)", i.AppName, i.Version, i.ProtocolVersion, i.BuildDate)
	}
	return fmt.Sprintf("%s v%s (proto %d)", i.AppName, i.Version, i.ProtocolVersion)
}
