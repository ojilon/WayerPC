package version

import (
	"bytes"
	"os"
	"strings"
	"testing"
)

func TestCurrentParses(t *testing.T) {
	in := Current()
	if in.AppName == "" {
		t.Fatal("AppName empty")
	}
	if in.Version == "" {
		t.Fatal("Version empty")
	}
	parts := strings.Split(in.Version, ".")
	if len(parts) != 3 {
		t.Fatalf("Version %q is not semver MAJOR.MINOR.PATCH", in.Version)
	}
	if in.ProtocolVersion < 1 {
		t.Fatalf("ProtocolVersion = %d, want >= 1", in.ProtocolVersion)
	}
	t.Logf("version: %s", in.String())
}

// The root file is edited for releases; this copy is embedded by Go.
func TestEmbeddedVersionMatchesRoot(t *testing.T) {
	root, err := os.ReadFile("../../version.json")
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(root, raw) {
		t.Fatal("embedded version is stale: copy version.json to internal/version/version.json")
	}
}
