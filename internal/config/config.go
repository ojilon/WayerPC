// Package config owns runtime settings: bind address, ports and — most
// importantly — where the data root lives.
//
// The data root is the single folder that holds shared/, received/ and
// Data/wayerpc.db. Resolution order:
//  1. WAYERPC_DATA_DIR env var (dev stub .data/, tests use t.TempDir()).
//  2. config.json "data_root" (written by the installer / Settings screen).
//  3. Legacy config.json "app_dir" (old Python layout kept the same three
//     folders directly under the app dir, so it maps 1:1 to a data root).
package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

const (
	// AppName is used for the OS config dir and default folder names.
	AppName = "WayerPC"
	// DefaultHost mirrors the Python server (all interfaces, hotspot LAN).
	DefaultHost = "0.0.0.0"
	// DefaultPort mirrors the Python server and the Android client default.
	DefaultPort = 5000
	// DefaultBufferSize mirrors server4.py BUFFER_SIZE.
	DefaultBufferSize = 4096
	// EnvDataDir overrides every other source (dev + tests).
	EnvDataDir = "WAYERPC_DATA_DIR"
)

// Config is the persisted + runtime configuration.
type Config struct {
	Host     string `json:"host"`
	Port     int    `json:"port"`
	DataRoot string `json:"data_root"`
	// InstallDir is informational (where bin/ lives). Empty in dev runs.
	InstallDir string `json:"install_dir,omitempty"`
}

// Defaults returns a Config with sane defaults and no data root.
func Defaults() Config {
	return Config{Host: DefaultHost, Port: DefaultPort}
}

// Addr returns "host:port".
func (c Config) Addr() string { return fmt.Sprintf("%s:%d", c.Host, c.Port) }

// Dir returns the OS-specific config dir, e.g.
// %LOCALAPPDATA%\WayerPC on Windows.
func Dir() (string, error) {
	base, err := os.UserConfigDir()
	if err != nil || base == "" {
		home, herr := os.UserHomeDir()
		if herr != nil {
			return "", fmt.Errorf("config: cannot locate user config dir: %w", err)
		}
		base = filepath.Join(home, ".config")
	}
	return filepath.Join(base, AppName), nil
}

// FilePath returns the config.json path.
func FilePath() (string, error) {
	dir, err := Dir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "config.json"), nil
}

// rawFile mirrors both the new keys and the legacy Python keys
// (app_dir/sub_dir) so old installs migrate transparently.
type rawFile struct {
	Host       string `json:"host"`
	Port       int    `json:"port"`
	DataRoot   string `json:"data_root"`
	InstallDir string `json:"install_dir"`
	AppDir     string `json:"app_dir"`
	SubDir     string `json:"sub_dir"`
}

// Load reads config.json. Missing file → defaults, nil.
// Corrupt file → defaults, nil (legacy load_saved_paths was equally lenient).
func Load() (Config, error) {
	cfg := Defaults()
	path, err := FilePath()
	if err != nil {
		return cfg, nil
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, nil
		}
		return cfg, nil
	}
	var rf rawFile
	if err := json.Unmarshal(raw, &rf); err != nil {
		return cfg, nil
	}
	if rf.Host != "" {
		cfg.Host = rf.Host
	}
	if rf.Port > 0 && rf.Port < 65536 {
		cfg.Port = rf.Port
	}
	cfg.InstallDir = rf.InstallDir
	switch {
	case rf.DataRoot != "":
		cfg.DataRoot = rf.DataRoot
	case rf.AppDir != "":
		cfg.DataRoot = rf.AppDir // legacy layout == data root layout
	}
	return cfg, nil
}

// Save persists cfg (creates the config dir as needed).
func Save(cfg Config) error {
	path, err := FilePath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("config: mkdir: %w", err)
	}
	if cfg.Host == "" {
		cfg.Host = DefaultHost
	}
	if cfg.Port <= 0 {
		cfg.Port = DefaultPort
	}
	raw, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(raw, '\n'), 0o644)
}

// ResolveDataRoot picks the data root following the documented order.
// Returns "" when nothing usable exists yet (caller shows onboarding).
func ResolveDataRoot(cfg Config) string {
	if env := os.Getenv(EnvDataDir); env != "" {
		return env
	}
	if cfg.DataRoot != "" {
		return cfg.DataRoot
	}
	return ""
}

// DevDataStub returns <cwd>/.data when it exists (fresh-clone dev runs).
func DevDataStub() string {
	cwd, err := os.Getwd()
	if err != nil {
		return ""
	}
	cand := filepath.Join(cwd, ".data")
	if st, err := os.Stat(cand); err == nil && st.IsDir() {
		return cand
	}
	return ""
}
