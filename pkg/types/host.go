package types

import "time"

// Host represents the OpenSearch document structure
type Host struct {
	IP       string    `json:"ip"`
	IPInt    int64     `json:"ip_int"`
	ASN      int       `json:"asn"`
	Org      string    `json:"org"`
	Country  string    `json:"country"`
	City     string    `json:"city,omitempty"`
	RDNS     string    `json:"rdns,omitempty"`
	LastSeen time.Time `json:"last_seen"`
	IsStale  bool      `json:"is_stale"`
	Ports    []Port    `json:"ports"`
	CVEs     []string  `json:"cves,omitempty"`
}

// Port represents a single port entry (nested in OpenSearch)
type Port struct {
	Port      int       `json:"port"`
	Protocol  string    `json:"protocol"`
	Service   string    `json:"service"`
	Product   string    `json:"product,omitempty"`
	CPE       string    `json:"cpe,omitempty"`
	Banner    string    `json:"banner,omitempty"`
	LastSeen  time.Time `json:"last_seen"`
	HTTP      *HTTPInfo `json:"http,omitempty"`
	TLS       *TLSInfo  `json:"tls,omitempty"`
	SSH       *SSHInfo  `json:"ssh,omitempty"`
}

// IPToInt converts IP string to int64
func IPToInt(ip string) int64 {
	// Simple implementation for IPv4
	// TODO: Handle IPv6
	return 0 // Placeholder
}
