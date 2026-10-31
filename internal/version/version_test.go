package version

import (
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
