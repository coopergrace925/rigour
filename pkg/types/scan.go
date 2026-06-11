package types

import "time"

// RawScan represents output from ZMap + ZGrab2
type RawScan struct {
	IP         string    `json:"ip"`
	Port       int       `json:"port"`
	Protocol   string    `json:"protocol"` // "tcp" or "udp"
	Service    string    `json:"service"`  // "https", "ssh", etc.
	Banner     string    `json:"banner"`
	ZGrabData  ZGrabData `json:"zgrab_data,omitempty"`
	ScannedAt  time.Time `json:"scanned_at"`
	ScannerID  string    `json:"scanner_id"`
}

// ZGrabData holds detailed ZGrab2 output
type ZGrabData struct {
	Status    string                 `json:"status"`
	Protocol  string                 `json:"protocol"`
	Result    map[string]interface{} `json:"result,omitempty"`
	TLS       *TLSInfo               `json:"tls,omitempty"`
	HTTP      *HTTPInfo              `json:"http,omitempty"`
	SSH       *SSHInfo               `json:"ssh,omitempty"`
}

type TLSInfo struct {
	Version string    `json:"version"`
	Cipher  string    `json:"cipher"`
	Cert    *CertInfo `json:"cert,omitempty"`
}

type CertInfo struct {
	SubjectCN   string    `json:"subject_cn"`
	IssuerCN    string    `json:"issuer_cn"`
	Fingerprint string    `json:"fingerprint"`
	NotAfter    time.Time `json:"not_after"`
}

type HTTPInfo struct {
	StatusCode int               `json:"status_code"`
	Title      string            `json:"title"`
	Server     string            `json:"server"`
	Headers    map[string]string `json:"headers,omitempty"`
}

type SSHInfo struct {
	ServerID string   `json:"server_id"`
	HASSH    string   `json:"hassh"`
	KexAlgos []string `json:"kex_algos,omitempty"`
}

// EnrichedScan represents enriched scan data
type EnrichedScan struct {
	RawScan
	ASN         int       `json:"asn"`
	Org         string    `json:"org"`
	Country     string    `json:"country"`
	City        string    `json:"city"`
	RDNS        string    `json:"rdns"`
	CPE         string    `json:"cpe,omitempty"`
	CVEs        []string  `json:"cves,omitempty"`
	EnrichedAt  time.Time `json:"enriched_at"`
}
