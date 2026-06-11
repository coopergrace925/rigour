package blocklist

import (
	"fmt"
	"net"
	"os"
	"sync"
)

// DefaultCIDRs contains mandatory blocklist entries (IPv4 only)
// TODO: Add IPv6 ranges (::1, fe80::/10, fc00::/7, etc.)
var DefaultCIDRs = []string{
	// RFC 1918 - Private networks
	"10.0.0.0/8",
	"172.16.0.0/12",
	"192.168.0.0/16",

	// RFC 5735 - Special use
	"0.0.0.0/8",
	"127.0.0.0/8",
	"169.254.0.0/16",
	"192.0.0.0/24",
	"192.0.2.0/24",
	"198.18.0.0/15",
	"198.51.100.0/24",
	"203.0.113.0/24",
	"224.0.0.0/4",
	"240.0.0.0/4",
	"255.255.255.255/32",

	// Carrier-grade NAT
	"100.64.0.0/10",

	// IPv6 to IPv4 relay
	"192.88.99.0/24",

	// DoD - US Department of Defense
	"6.0.0.0/8",
	"7.0.0.0/8",
	"11.0.0.0/8",
	"21.0.0.0/8",
	"22.0.0.0/8",
	"26.0.0.0/8",
	"28.0.0.0/8",
	"29.0.0.0/8",
	"30.0.0.0/8",
	"33.0.0.0/8",
	"55.0.0.0/8",
	"214.0.0.0/8",
	"215.0.0.0/8",
}

type Blocklist struct {
	mu      sync.RWMutex
	nets    []*net.IPNet
	optOuts map[string]struct{}
}

func NewBlocklist() *Blocklist {
	bl := &Blocklist{
		optOuts: make(map[string]struct{}),
	}

	for _, cidr := range DefaultCIDRs {
		_, ipnet, err := net.ParseCIDR(cidr)
		if err != nil {
			// Should never happen with hardcoded defaults
			panic(fmt.Sprintf("invalid default CIDR %s: %v", cidr, err))
		}
		bl.nets = append(bl.nets, ipnet)
	}

	return bl
}

func (b *Blocklist) IsBlocked(ip net.IP) bool {
	if ip == nil {
		return false
	}

	b.mu.RLock()
	defer b.mu.RUnlock()

	// Check opt-outs
	if _, ok := b.optOuts[ip.String()]; ok {
		return true
	}

	// Check CIDR blocks
	for _, ipnet := range b.nets {
		if ipnet.Contains(ip) {
			return true
		}
	}

	return false
}

func (b *Blocklist) AddOptOut(ip net.IP) {
	if ip == nil {
		return
	}

	b.mu.Lock()
	defer b.mu.Unlock()
	b.optOuts[ip.String()] = struct{}{}
}

func (b *Blocklist) GenerateFile(path string) error {
	b.mu.RLock()
	defer b.mu.RUnlock()

	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("failed to create blocklist file: %w", err)
	}
	defer f.Close()

	fmt.Fprintln(f, "# Rigour Blocklist — Auto-generated, do not edit manually")
	fmt.Fprintln(f, "# RFC1918, DoD, IANA reserved, opt-outs")
	fmt.Fprintln(f)

	for _, cidr := range DefaultCIDRs {
		fmt.Fprintln(f, cidr)
	}

	fmt.Fprintln(f)
	fmt.Fprintln(f, "# Opt-out IPs")
	for ip := range b.optOuts {
		fmt.Fprintf(f, "%s/32\n", ip)
	}

	return nil
}
