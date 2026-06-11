package types

import "time"

// Port represents a single port discovered on a host (OpenSearch nested type)
type Port struct {
	Port     int       `json:"port"`
	Protocol string    `json:"protocol"`
	Service  string    `json:"service,omitempty"`
	Product  string    `json:"product,omitempty"`
	CPE      string    `json:"cpe,omitempty"`
	Banner   string    `json:"banner,omitempty"`
	LastSeen time.Time `json:"last_seen"`
	HTTP     *HTTPData `json:"http,omitempty"`
	TLS      *TLSData  `json:"tls,omitempty"`
	SSH      *SSHData  `json:"ssh,omitempty"`
}

// HTTPData represents HTTP-specific metadata
type HTTPData struct {
	StatusCode int    `json:"status_code,omitempty"`
	Title      string `json:"title,omitempty"`
	Server     string `json:"server,omitempty"`
}

// TLSData represents TLS-specific metadata
type TLSData struct {
	Version string    `json:"version,omitempty"`
	Cert    *CertData `json:"cert,omitempty"`
}

// CertData represents TLS certificate metadata
type CertData struct {
	SubjectCN   string    `json:"subject_cn,omitempty"`
	IssuerCN    string    `json:"issuer_cn,omitempty"`
	Fingerprint string    `json:"fingerprint,omitempty"`
	NotAfter    time.Time `json:"not_after,omitempty"`
	SAN         []string  `json:"san,omitempty"`
}

// SSHData represents SSH-specific metadata
type SSHData struct {
	HASSH    string   `json:"hassh,omitempty"`
	ServerID string   `json:"server_id,omitempty"`
	KexAlgos []string `json:"kex_algos,omitempty"`
}

// HostDocument represents a host document in OpenSearch (with nested ports)
type HostDocument struct {
	IP       string    `json:"ip"`
	IPInt    uint64    `json:"ip_int"`
	ASN      int       `json:"asn,omitempty"`
	Org      string    `json:"org,omitempty"`
	Country  string    `json:"country,omitempty"`
	City     string    `json:"city,omitempty"`
	RDNS     string    `json:"rdns,omitempty"`
	LastSeen time.Time `json:"last_seen"`
	IsStale  bool      `json:"is_stale"`
	CVEs     []string  `json:"cves,omitempty"`
	Tags     []string  `json:"tags,omitempty"`
	Ports    []Port    `json:"ports,omitempty"`
}
