package netutil

import "testing"

func TestIsPrivate(t *testing.T) {
	private := []string{"10.1.2.3", "172.16.0.1", "172.31.255.254", "192.168.43.21", "192.168.1.7"}
	for _, ip := range private {
		if !isPrivate(ip) {
			t.Errorf("isPrivate(%q) = false, want true", ip)
		}
	}
	others := []string{"", "8.8.8.8", "172.32.0.1", "172.15.0.1", "192.169.1.1", "127.0.0.1", "::1"}
	for _, ip := range others {
		if isPrivate(ip) {
			t.Errorf("isPrivate(%q) = true, want false", ip)
		}
	}
}

func TestDialAddr(t *testing.T) {
	cases := []struct {
		host string
		port int
		lan  string
		want string
		name string
	}{
		{"0.0.0.0", 5000, "192.168.43.21", "192.168.43.21:5000", "private address wins over bind host"},
		{"0.0.0.0", 5000, "", "127.0.0.1:5000", "no private address falls back to loopback"},
		{"192.168.1.5", 5000, "", "192.168.1.5:5000", "explicit host kept"},
		{"0.0.0.0", 0, "10.0.0.2", "10.0.0.2", "port 0 omitted"},
	}
	for _, c := range cases {
		if got := DialAddr(c.host, c.port, c.lan); got != c.want {
			t.Errorf("%s: DialAddr(%q,%d,%q) = %q, want %q", c.name, c.host, c.port, c.lan, got, c.want)
		}
	}
}

func TestLanAddrShape(t *testing.T) {
	// A sandbox may have no private interface; when one exists it must be a
	// bare IPv4 string, never a host:port pair or 0.0.0.0.
	got := LanAddr()
	if got == "" {
		t.Skip("no private IPv4 on this host")
	}
	if got == "0.0.0.0" || got == "127.0.0.1" {
		t.Fatalf("LanAddr returned a non-dialable address: %q", got)
	}
	if !isPrivate(got) {
		t.Fatalf("LanAddr = %q, want a private IPv4", got)
	}
}
