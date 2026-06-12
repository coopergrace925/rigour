package coordinator

import (
	"fmt"
	"math/rand"
	"net"
)

// RandomizedCIDRScanner provides uniform temporal coverage via randomization
// Implements Shodan's approach: "Generate a random IPv4 address"
type RandomizedCIDRScanner struct {
	cidr   string
	ipnet  *net.IPNet
	rng    *rand.Rand
}

// NewRandomizedCIDRScanner creates a scanner with random IP selection
func NewRandomizedCIDRScanner(cidr string, seed int64) (*RandomizedCIDRScanner, error) {
	_, ipnet, err := net.ParseCIDR(cidr)
	if err != nil {
		return nil, fmt.Errorf("invalid CIDR: %w", err)
	}

	return &RandomizedCIDRScanner{
		cidr:  cidr,
		ipnet: ipnet,
		rng:   rand.New(rand.NewSource(seed)),
	}, nil
}

// NextRandomIP generates a random IP within the CIDR block
// Shodan algorithm: "Generate a random IPv4 address"
func (s *RandomizedCIDRScanner) NextRandomIP() net.IP {
	// Get the network address
	ip := s.ipnet.IP.To4()
	if ip == nil {
		// IPv6 not supported in this implementation
		return nil
	}

	// Calculate the number of addresses in the network
	ones, bits := s.ipnet.Mask.Size()
	numAddrs := 1 << uint(bits-ones)

	// Generate random offset
	offset := s.rng.Intn(numAddrs)

	// Add offset to base IP
	result := make(net.IP, 4)
	copy(result, ip)

	// Add the offset as big-endian
	addr := uint32(result[0])<<24 | uint32(result[1])<<16 | uint32(result[2])<<8 | uint32(result[3])
	addr += uint32(offset)

	result[0] = byte(addr >> 24)
	result[1] = byte(addr >> 16)
	result[2] = byte(addr >> 8)
	result[3] = byte(addr)

	return result
}

// GenerateRandomIPs generates N random IPs within the CIDR
func (s *RandomizedCIDRScanner) GenerateRandomIPs(count int) []string {
	ips := make([]string, 0, count)
	seen := make(map[string]bool)

	for len(ips) < count {
		ip := s.NextRandomIP()
		if ip == nil {
			break
		}

		ipStr := ip.String()
		if !seen[ipStr] {
			ips = append(ips, ipStr)
			seen[ipStr] = true
		}
	}

	return ips
}

// ShufflePortList randomizes port scan order
// Shodan algorithm: "Generate a random port to test from the list"
func ShufflePortList(ports []int, seed int64) []int {
	rng := rand.New(rand.NewSource(seed))
	shuffled := make([]int, len(ports))
	copy(shuffled, ports)

	// Fisher-Yates shuffle
	for i := len(shuffled) - 1; i > 0; i-- {
		j := rng.Intn(i + 1)
		shuffled[i], shuffled[j] = shuffled[j], shuffled[i]
	}

	return shuffled
}

// RandomizedScanStrategy generates randomized scan tasks
type RandomizedScanStrategy struct {
	scanner *RandomizedCIDRScanner
	ports   []int
}

// NewRandomizedScanStrategy creates a production-grade randomized scanner
func NewRandomizedScanStrategy(cidr string, ports []int, seed int64) (*RandomizedScanStrategy, error) {
	scanner, err := NewRandomizedCIDRScanner(cidr, seed)
	if err != nil {
		return nil, err
	}

	// Shuffle ports for temporal uniformity
	shuffledPorts := ShufflePortList(ports, seed)

	return &RandomizedScanStrategy{
		scanner: scanner,
		ports:   shuffledPorts,
	}, nil
}

// NextTask returns the next randomized scan task
// Implements Shodan's algorithm:
// 1. Generate a random IPv4 address
// 2. Generate a random port to test from the list
func (s *RandomizedScanStrategy) NextTask() (ip string, port int) {
	if len(s.ports) == 0 {
		return "", 0
	}

	ip = s.scanner.NextRandomIP().String()
	portIdx := s.scanner.rng.Intn(len(s.ports))
	port = s.ports[portIdx]

	return ip, port
}
