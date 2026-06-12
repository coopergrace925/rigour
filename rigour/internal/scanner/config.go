package scanner

import (
	"fmt"
	"strings"
)

// ScanConfig holds configuration for ZMap + ZGrab2 scanning
type ScanConfig struct {
	// ZMap configuration
	ZMapBinary    string
	ZMapRate      int    // packets per second
	ZMapBandwidth string // e.g., "10M"
	ZMapInterface string // network interface

	// ZGrab2 configuration
	ZGrab2Binary string
	ZGrab2Module string // http, https, ssh, etc.

	// Hostname-aware scanning (Shodan standard)
	EnableSNI        bool   // Enable TLS SNI
	EnableHostHeader bool   // Enable HTTP Host header
	Hostname         string // Hostname to use (if known)

	// Protocol detection
	EnableProtocolDetection bool // Auto-detect protocol mismatches

	// Scanner metadata
	ScannerID     string // Unique scanner identifier
	ScannerRegion string // Geographic region (us-east, eu-west, etc.)
}

// DefaultScanConfig returns production-grade defaults aligned with Shodan
func DefaultScanConfig(scannerID string) *ScanConfig {
	return &ScanConfig{
		ZMapBinary:              "/usr/local/bin/zmap",
		ZMapRate:                10000, // 10k pps (conservative)
		ZMapBandwidth:           "10M",
		ZMapInterface:           "eth0",
		ZGrab2Binary:            "/usr/local/bin/zgrab2",
		EnableSNI:               true, // CRITICAL: Shodan enables this
		EnableHostHeader:        true, // CRITICAL: Shodan enables this
		EnableProtocolDetection: true,
		ScannerID:               scannerID,
		ScannerRegion:           "default",
	}
}

// ZMapCommand builds a ZMap command for scanning
func (c *ScanConfig) ZMapCommand(cidr string, port int) []string {
	args := []string{
		c.ZMapBinary,
		"-p", fmt.Sprintf("%d", port),
		"-B", c.ZMapBandwidth,
		"-r", fmt.Sprintf("%d", c.ZMapRate),
		"-i", c.ZMapInterface,
		"-o", "-", // Output to stdout
		"--output-fields=saddr",
		cidr,
	}
	return args
}

// ZGrab2Command builds a ZGrab2 command for banner grabbing
func (c *ScanConfig) ZGrab2Command(port int, service string) []string {
	module := c.inferZGrab2Module(port, service)

	args := []string{
		c.ZGrab2Binary,
		module,
		"--port", fmt.Sprintf("%d", port),
	}

	// Hostname-aware configuration (Shodan standard)
	if c.EnableSNI && (module == "tls" || module == "https") {
		if c.Hostname != "" {
			args = append(args, "--server-name", c.Hostname)
		}
	}

	if c.EnableHostHeader && (module == "http" || module == "https") {
		if c.Hostname != "" {
			args = append(args, "--host", c.Hostname)
		}
	}

	return args
}

// inferZGrab2Module determines the correct ZGrab2 module based on port/service
func (c *ScanConfig) inferZGrab2Module(port int, service string) string {
	// Explicit service mapping
	serviceMap := map[string]string{
		"http":    "http",
		"https":   "https",
		"ssh":     "ssh",
		"ftp":     "ftp",
		"smtp":    "smtp",
		"pop3":    "pop3",
		"imap":    "imap",
		"telnet":  "telnet",
		"mysql":   "mysql",
		"postgres": "postgres",
		"mongodb": "mongodb",
		"redis":   "redis",
		"modbus":  "modbus",
		"bacnet":  "bacnet",
	}

	if module, ok := serviceMap[strings.ToLower(service)]; ok {
		return module
	}

	// Port-based inference
	portMap := map[int]string{
		21:    "ftp",
		22:    "ssh",
		23:    "telnet",
		25:    "smtp",
		80:    "http",
		110:   "pop3",
		143:   "imap",
		443:   "https",
		465:   "smtp",
		587:   "smtp",
		993:   "imap",
		995:   "pop3",
		3306:  "mysql",
		5432:  "postgres",
		6379:  "redis",
		8080:  "http",
		8443:  "https",
		27017: "mongodb",
		// ICS/SCADA
		102:   "s7",
		502:   "modbus",
		47808: "bacnet",
	}

	if module, ok := portMap[port]; ok {
		return module
	}

	// Default to banner grabbing
	return "banner"
}

// DetectProtocolMismatch checks if the banner indicates a protocol mismatch
// Example: SSH running on port 80
func DetectProtocolMismatch(expectedService string, banner string) (actualService string, mismatch bool) {
	banner = strings.ToLower(banner)

	// Protocol signatures
	signatures := map[string][]string{
		"ssh":   {"ssh-", "openssh"},
		"http":  {"http/1", "http/2", "server:"},
		"ftp":   {"220 ", "ftp"},
		"smtp":  {"220 ", "smtp", "esmtp"},
		"mysql": {"mysql"},
		"redis": {"-err", "+ok", "$"},
	}

	for service, patterns := range signatures {
		for _, pattern := range patterns {
			if strings.Contains(banner, pattern) {
				if service != strings.ToLower(expectedService) {
					return service, true
				}
				return service, false
			}
		}
	}

	return expectedService, false
}
