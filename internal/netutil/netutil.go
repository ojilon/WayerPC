// Package netutil holds the few networking helpers the UI needs to explain
// itself: the address a phone should type in, rather than 0.0.0.0.
package netutil

import (
	"net"
	"sort"
	"strings"
)

// LanAddr returns the first usable private IPv4 of this PC ("192.168.1.7"),
// or "" when no interface has one. The phone cannot dial 0.0.0.0, so the UI
// shows this instead of the bind address.
//
// Order: private addresses (10/8, 172.16/12, 192.168/16) on up, non-loopback
// interfaces, preferring ones whose interface name does not look virtual
// (vEthernet/VirtualBox/vmnet/docker/br-/veth) so a hotspot address wins over
// a VM bridge.
func LanAddr() string {
	addrs := candidateAddrs()
	sort.SliceStable(addrs, func(i, j int) bool {
		return rank(addrs[i]) < rank(addrs[j])
	})
	for _, a := range addrs {
		return a.ip
	}
	return ""
}

type candidate struct {
	ip   string
	name string
}

func candidateAddrs() []candidate {
	ifaces, err := net.Interfaces()
	if err != nil {
		return nil
	}
	var out []candidate
	seen := map[string]bool{}
	for _, iface := range ifaces {
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
			continue
		}
		list, err := iface.Addrs()
		if err != nil {
			continue
		}
		for _, addr := range list {
			var ip net.IP
			switch v := addr.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			}
			if ip == nil {
				continue
			}
			ip4 := ip.To4()
			if ip4 == nil || ip4.IsLoopback() || ip4.IsLinkLocalUnicast() {
				continue
			}
			s := ip4.String()
			if seen[s] {
				continue
			}
			seen[s] = true
			out = append(out, candidate{ip: s, name: strings.ToLower(iface.Name)})
		}
	}
	return out
}

// rank orders candidates: private + physical-first, public last.
func rank(c candidate) int {
	score := 100
	if isPrivate(c.ip) {
		score = 0
	}
	switch {
	case strings.Contains(c.name, "vethernet"),
		strings.Contains(c.name, "virtualbox"),
		strings.Contains(c.name, "vmnet"),
		strings.Contains(c.name, "vmware"),
		strings.Contains(c.name, "docker"),
		strings.HasPrefix(c.name, "br-"),
		strings.HasPrefix(c.name, "veth"),
		strings.HasPrefix(c.name, "virbr"):
		score += 10
	}
	return score
}

func isPrivate(ipStr string) bool {
	ip := net.ParseIP(ipStr)
	if ip == nil {
		return false
	}
	ip4 := ip.To4()
	if ip4 == nil {
		return false
	}
	switch {
	case ip4[0] == 10:
		return true
	case ip4[0] == 172 && ip4[1] >= 16 && ip4[1] <= 31:
		return true
	case ip4[0] == 192 && ip4[1] == 168:
		return true
	}
	return false
}

// DialAddr turns a bind host + port into the address shown to the user.
// Falls back to the bind host when no private address exists.
func DialAddr(host string, port int, lanAddr string) string {
	if lanAddr != "" {
		if port > 0 {
			return net.JoinHostPort(lanAddr, itoa(port))
		}
		return lanAddr
	}
	if host == "" || host == "0.0.0.0" || host == "::" {
		if port > 0 {
			return net.JoinHostPort("127.0.0.1", itoa(port))
		}
		return "127.0.0.1"
	}
	if port > 0 {
		return net.JoinHostPort(host, itoa(port))
	}
	return host
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var buf [8]byte
	i := len(buf)
	for n > 0 && i > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	return string(buf[i:])
}
