package enrichment

import (
	"encoding/binary"
	"fmt"
	"os"
	"strings"

	"github.com/ctrlsam/rigour/pkg/types"
)

// CVEDatabase provides fast in-memory CVE lookups with verification status
type CVEDatabase struct {
	file   *os.File
	numCPE uint32
	keys   []string
	cves   map[string][]types.CVEInfo
}

// OpenCVEDatabase loads the CVE database from disk
func OpenCVEDatabase(path string) (*CVEDatabase, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open CVE index: %w", err)
	}
	defer f.Close()

	var numCPE uint32
	if err := binary.Read(f, binary.BigEndian, &numCPE); err != nil {
		return nil, fmt.Errorf("failed to read header: %w", err)
	}

	keys := make([]string, 0, numCPE)
	cves := make(map[string][]types.CVEInfo)

	for i := uint32(0); i < numCPE; i++ {
		var kLen uint16
		if err := binary.Read(f, binary.BigEndian, &kLen); err != nil {
			return nil, err
		}
		kBytes := make([]byte, kLen)
		if _, err := f.Read(kBytes); err != nil {
			return nil, err
		}

		var cveLen uint16
		if err := binary.Read(f, binary.BigEndian, &cveLen); err != nil {
			return nil, err
		}
		cveBytes := make([]byte, cveLen)
		if _, err := f.Read(cveBytes); err != nil {
			return nil, err
		}

		k := string(kBytes)
		cveList := parseCVEList(string(cveBytes))
		keys = append(keys, k)
		cves[k] = cveList
	}

	return &CVEDatabase{
		numCPE: numCPE,
		keys:   keys,
		cves:   cves,
	}, nil
}

// parseCVEList converts comma-separated CVE string to CVEInfo list
func parseCVEList(cveStr string) []types.CVEInfo {
	if cveStr == "" {
		return nil
	}

	ids := strings.Split(cveStr, ",")
	result := make([]types.CVEInfo, len(ids))

	for i, id := range ids {
		result[i] = types.CVEInfo{
			ID:       strings.TrimSpace(id),
			Verified: false, // Will be enriched with verification data
		}
	}

	return result
}

// GetCVEs returns CVEs for a given CPE
func (db *CVEDatabase) GetCVEs(cpe string) []types.CVEInfo {
	if db == nil || db.cves == nil {
		return nil
	}
	return db.cves[cpe]
}

// EnrichWithVerificationData adds verification status from CISA KEV or exploit-db
// This should be called during database initialization with verification data
func (db *CVEDatabase) EnrichWithVerificationData(verifiedCVEs map[string]bool) {
	for cpe, cveList := range db.cves {
		for i := range cveList {
			if verifiedCVEs[cveList[i].ID] {
				cveList[i].Verified = true
			}
		}
		db.cves[cpe] = cveList
	}
}
