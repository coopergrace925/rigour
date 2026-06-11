package opensearch

import (
	"net"
	"sort"
	"time"

	"github.com/ctrlsam/rigour/pkg/types"
)

const PortStalenessTTL = 7 * 24 * time.Hour // 7 days

// MergePorts merges a new scan result into an existing host document.
// - Adds new ports
// - Updates existing ports with fresh data
// - Prunes ports not seen in 7+ days
func MergePorts(existing types.HostDocument, newScan types.EnrichedScan) types.HostDocument {
	portMap := make(map[int]types.Port)

	// Load existing ports
	for _, p := range existing.Ports {
		portMap[p.Port] = p
	}

	// Convert ZGrabData types to OpenSearch types
	var httpData *types.HTTPData
	if newScan.ZGrabData.HTTP != nil {
		httpData = &types.HTTPData{
			StatusCode: newScan.ZGrabData.HTTP.StatusCode,
			Title:      newScan.ZGrabData.HTTP.Title,
			Server:     newScan.ZGrabData.HTTP.Server,
		}
	}

	var tlsData *types.TLSData
	if newScan.ZGrabData.TLS != nil {
		var certData *types.CertData
		if newScan.ZGrabData.TLS.Cert != nil {
			certData = &types.CertData{
				SubjectCN:   newScan.ZGrabData.TLS.Cert.SubjectCN,
				IssuerCN:    newScan.ZGrabData.TLS.Cert.IssuerCN,
				Fingerprint: newScan.ZGrabData.TLS.Cert.Fingerprint,
				NotAfter:    newScan.ZGrabData.TLS.Cert.NotAfter,
			}
			if len(newScan.ZGrabData.TLS.Cert.SAN) > 0 {
				certData.SAN = newScan.ZGrabData.TLS.Cert.SAN
			}
		}
		tlsData = &types.TLSData{
			Version: newScan.ZGrabData.TLS.Version,
			Cert:    certData,
		}
	}

	var sshData *types.SSHData
	if newScan.ZGrabData.SSH != nil {
		sshData = &types.SSHData{
			HASSH:    newScan.ZGrabData.SSH.HASSH,
			ServerID: newScan.ZGrabData.SSH.ServerID,
			KexAlgos: newScan.ZGrabData.SSH.KexAlgos,
		}
	}

	// Add or update scanned port
	portMap[newScan.Port] = types.Port{
		Port:     newScan.Port,
		Protocol: newScan.Protocol,
		Service:  newScan.Service,
		Product:  "", // TODO: Extract product from banner or CPE
		Banner:   newScan.Banner,
		CPE:      newScan.CPE,
		LastSeen: time.Now(),
		HTTP:     httpData,
		TLS:      tlsData,
		SSH:      sshData,
	}

	// Prune stale ports
	var activePorts []types.Port
	cutoff := time.Now().Add(-PortStalenessTTL)
	for _, p := range portMap {
		if p.LastSeen.After(cutoff) {
			activePorts = append(activePorts, p)
		}
	}

	existing.Ports = activePorts
	existing.LastSeen = time.Now()
	existing.IsStale = false
	existing.IP = newScan.IP
	existing.IPInt = ipToInt(newScan.IP)
	existing.ASN = newScan.ASN
	existing.Org = newScan.Org
	existing.Country = newScan.Country
	existing.City = newScan.City
	existing.RDNS = newScan.RDNS

	// Deduplicate CVEs across all ports
	cveSet := make(map[string]struct{})
	for _, cve := range existing.CVEs {
		cveSet[cve] = struct{}{}
	}
	for _, cve := range newScan.CVEs {
		cveSet[cve] = struct{}{}
	}
	existing.CVEs = nil
	for cve := range cveSet {
		existing.CVEs = append(existing.CVEs, cve)
	}
	sort.Strings(existing.CVEs) // Ensure deterministic ordering

	return existing
}

// ipToInt converts an IPv4 address string to uint64 for range queries.
// Returns 0 if the IP is invalid or not IPv4.
func ipToInt(ip string) uint64 {
	parsedIP := net.ParseIP(ip)
	if parsedIP == nil {
		return 0
	}

	ipv4 := parsedIP.To4()
	if ipv4 == nil {
		return 0 // Only IPv4 supported
	}

	return uint64(ipv4[0])<<24 | uint64(ipv4[1])<<16 | uint64(ipv4[2])<<8 | uint64(ipv4[3])
}
