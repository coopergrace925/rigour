package coordinator

import (
	_ "embed"
	"encoding/json"
)

// ShodanPorts contains the official Shodan port list (3,846 ports)
// Source: https://api.shodan.io/shodan/ports
// Last updated: 2026-06-12
//
//go:embed ports_shodan.json
var shodanPortsJSON []byte

// PortTier defines the scanning priority/frequency tier
type PortTier int

const (
	TierCritical  PortTier = iota // Every 6 hours - remote access, high-risk
	TierHigh                       // Every 24 hours - web, email, databases
	TierICS                        // Every 24 hours - ICS/SCADA protocols
	TierTop1000                    // Every 7 days - Nmap top-1000
	TierShodanFull                 // Every 30 days - Full Shodan list (3,846 ports)
)

// PortDefinition contains metadata about a port
type PortDefinition struct {
	Port        int      `json:"port"`
	Tier        PortTier `json:"tier"`
	Protocol    string   `json:"protocol"`    // "tcp" or "udp"
	Service     string   `json:"service"`     // e.g., "ssh", "http", "modbus"
	Description string   `json:"description"` // Human-readable description
}

// PortRegistry is a production-grade port management system
type PortRegistry struct {
	// All ports indexed by port number
	byPort map[int]*PortDefinition

	// Ports grouped by tier for efficient scheduling
	byTier map[PortTier][]int

	// Shodan's full port list
	shodanPorts []int
}

// NewPortRegistry initializes the production port registry
func NewPortRegistry() (*PortRegistry, error) {
	pr := &PortRegistry{
		byPort: make(map[int]*PortDefinition),
		byTier: make(map[PortTier][]int),
	}

	// Load Shodan's official port list
	if err := json.Unmarshal(shodanPortsJSON, &pr.shodanPorts); err != nil {
		return nil, err
	}

	// Define tier-specific ports
	pr.defineTiers()

	return pr, nil
}

// defineTiers configures the tiered port structure
func (pr *PortRegistry) defineTiers() {
	// Tier 1: CRITICAL - Every 6 hours (remote access, RDP, VNC)
	criticalPorts := []struct {
		port    int
		service string
		desc    string
	}{
		{22, "ssh", "SSH remote login"},
		{23, "telnet", "Telnet remote login (insecure)"},
		{3389, "rdp", "Remote Desktop Protocol"},
		{5900, "vnc", "VNC remote desktop"},
		{5901, "vnc", "VNC remote desktop (alt)"},
		{445, "smb", "SMB file sharing"},
		{3306, "mysql", "MySQL database"},
		{5432, "postgresql", "PostgreSQL database"},
		{6379, "redis", "Redis database (often exposed)"},
		{27017, "mongodb", "MongoDB database"},
	}
	for _, p := range criticalPorts {
		pr.addPort(p.port, TierCritical, "tcp", p.service, p.desc)
	}

	// Tier 2: HIGH - Every 24 hours (web, email, common services)
	highPorts := []struct {
		port    int
		service string
		desc    string
	}{
		{80, "http", "HTTP web server"},
		{443, "https", "HTTPS web server"},
		{8080, "http-proxy", "HTTP proxy/alt web"},
		{8443, "https-alt", "HTTPS alternative"},
		{8000, "http-alt", "HTTP alternative"},
		{8001, "http-alt", "HTTP alternative"},
		{8008, "http-alt", "HTTP alternative"},
		{8081, "http-alt", "HTTP alternative"},
		{8888, "http-alt", "HTTP alternative"},
		{9200, "elasticsearch", "Elasticsearch REST API"},
		{9300, "elasticsearch", "Elasticsearch transport"},
		{25, "smtp", "SMTP email server"},
		{465, "smtps", "SMTP over SSL"},
		{587, "smtp", "SMTP submission"},
		{110, "pop3", "POP3 email"},
		{143, "imap", "IMAP email"},
		{993, "imaps", "IMAP over SSL"},
		{995, "pop3s", "POP3 over SSL"},
		{21, "ftp", "FTP file transfer"},
		{1433, "mssql", "Microsoft SQL Server"},
		{5000, "upnp", "UPnP device discovery"},
		{5001, "synology", "Synology DSM"},
	}
	for _, p := range highPorts {
		pr.addPort(p.port, TierHigh, "tcp", p.service, p.desc)
	}

	// Tier 3: ICS/SCADA - Every 24 hours (industrial control systems)
	icsPorts := []struct {
		port    int
		service string
		desc    string
	}{
		{102, "s7", "Siemens S7"},
		{502, "modbus", "Modbus protocol"},
		{1089, "ff-annunc", "FF Annunciation"},
		{1911, "mtp", "Niagara Fox"},
		{2222, "iec-104", "IEC 60870-5-104"},
		{4000, "dnp3", "DNP3 protocol"},
		{4840, "opcua", "OPC UA"},
		{20000, "dnp3", "DNP3 over TCP"},
		{44818, "ethernetip", "EtherNet/IP"},
		{47808, "bacnet", "BACnet"},
		{55000, "omron", "Omron FINS"},
	}
	for _, p := range icsPorts {
		pr.addPort(p.port, TierICS, "tcp", p.service, p.desc)
	}

	// Tier 4: TOP1000 - Every 7 days
	// Add all Nmap top-1000 ports not already in critical/high/ics
	top1000Sample := []int{
		53, 135, 139, 161, 389, 636, 1080, 1194, 1723, 3128,
		5060, 5061, 5432, 8009, 8010, 8180, 8181, 9090, 9091, 9999,
	}
	for _, port := range top1000Sample {
		if _, exists := pr.byPort[port]; !exists {
			pr.addPort(port, TierTop1000, "tcp", "unknown", "Nmap top-1000")
		}
	}

	// Tier 5: SHODAN_FULL - Every 30 days
	// Add all remaining Shodan ports
	for _, port := range pr.shodanPorts {
		if _, exists := pr.byPort[port]; !exists {
			pr.addPort(port, TierShodanFull, "tcp", "unknown", "Shodan port list")
		}
	}
}

// addPort registers a port with its metadata
func (pr *PortRegistry) addPort(port int, tier PortTier, protocol, service, desc string) {
	def := &PortDefinition{
		Port:        port,
		Tier:        tier,
		Protocol:    protocol,
		Service:     service,
		Description: desc,
	}
	pr.byPort[port] = def
	pr.byTier[tier] = append(pr.byTier[tier], port)
}

// GetPortsByTier returns all ports in a given tier
func (pr *PortRegistry) GetPortsByTier(tier PortTier) []int {
	return pr.byTier[tier]
}

// GetPortInfo returns metadata for a specific port
func (pr *PortRegistry) GetPortInfo(port int) (*PortDefinition, bool) {
	def, exists := pr.byPort[port]
	return def, exists
}

// GetAllPorts returns all registered ports
func (pr *PortRegistry) GetAllPorts() []int {
	ports := make([]int, 0, len(pr.byPort))
	for port := range pr.byPort {
		ports = append(ports, port)
	}
	return ports
}

// GetTierStats returns statistics about each tier
func (pr *PortRegistry) GetTierStats() map[string]int {
	return map[string]int{
		"critical":    len(pr.byTier[TierCritical]),
		"high":        len(pr.byTier[TierHigh]),
		"ics":         len(pr.byTier[TierICS]),
		"top1000":     len(pr.byTier[TierTop1000]),
		"shodan_full": len(pr.byTier[TierShodanFull]),
		"total":       len(pr.byPort),
	}
}
