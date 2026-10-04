package config

import (
	"os"
	"path/filepath"
	"strconv"
	"testing"
)

func TestDefaults(t *testing.T) {
	c := Defaults()
	if c.Host != DefaultHost || c.Port != DefaultPort {
		t.Fatalf("bad defaults: %+v", c)
	}
	if c.Addr() != "0.0.0.0:5000" {
		t.Fatalf("bad addr: %q", c.Addr())
	}
}

func TestEnvOverrides(t *testing.T) {
	t.Setenv(EnvDataDir, t.TempDir())
	cfg := Defaults()
	if got := ResolveDataRoot(cfg); got == "" {
		t.Fatal("env data dir not picked up")
	}
}

func TestLegacyAppDirMigration(t *testing.T) {
	// Simulate a legacy config.json with only app_dir.
	dir := t.TempDir()
	t.Setenv("APPDATA", dir) // Windows: UserConfigDir lives under APPDATA
	legacyRoot := filepath.Join(dir, "D-stub")
	if err := os.MkdirAll(legacyRoot, 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv(EnvDataDir, "") // make sure env does not win
	cfgPath, err := FilePath()
	if err != nil {
		t.Skipf("no config dir on this platform: %v", err)
	}
	if err := os.MkdirAll(filepath.Dir(cfgPath), 0o755); err != nil {
		t.Fatal(err)
	}
	raw := `{"app_dir": ` + quote(legacyRoot) + `}`
	if err := os.WriteFile(cfgPath, []byte(raw), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.DataRoot != legacyRoot {
		t.Fatalf("legacy app_dir not migrated: %+v", cfg)
	}
}

func quote(s string) string { return strconv.Quote(s) }
