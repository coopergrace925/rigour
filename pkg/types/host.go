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
	Port     int       `json:"port"`
	Protocol string    `json:"protocol"`
	Service  string    `json:"service"`
	Product  string    `json:"product,omitempty"`
	CPE      string    `json:"cpe,omitempty"`
	Banner   string    `json:"banner,omitempty"`
	LastSeen time.Time `json:"last_seen"`
	HTTP     *HTTPInfo `json:"http,omitempty"`
	TLS      *TLSInfo  `json:"tls,omitempty"`
	SSH      *SSHInfo  `json:"ssh,omitempty"`
}

// IPToInt converts IP string to int64
func IPToInt(ip string) int64 {
	// Split IP into octets
	parts := make([]int64, 4)
	var octet int64
	partIndex := 0

	for i := 0; i < len(ip); i++ {
		if ip[i] == '.' {
			if partIndex >= 3 {
				return 0 // Too many dots
			}
			parts[partIndex] = octet
			partIndex++
			octet = 0
		} else if ip[i] >= '0' && ip[i] <= '9' {
			octet = octet*10 + int64(ip[i]-'0')
			if octet > 255 {
				return 0 // Invalid octet
			}
		} else {
			return 0 // Invalid character
		}
	}

	if partIndex != 3 {
		return 0 // Wrong number of parts
	}
	parts[3] = octet

	// Combine octets into int64
	return (parts[0] << 24) | (parts[1] << 16) | (parts[2] << 8) | parts[3]
}
