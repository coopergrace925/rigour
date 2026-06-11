package enrichment

import (
	"encoding/binary"
	"fmt"
	"os"
	"strings"
)

type CVEDatabase struct {
	file   *os.File
	numCPE uint32
	keys   []string
	cves   map[string][]string
}

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
	cves := make(map[string][]string)

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
		cveList := strings.Split(string(cveBytes), ",")
		keys = append(keys, k)
		cves[k] = cveList
	}

	return &CVEDatabase{
		numCPE: numCPE,
		keys:   keys,
		cves:   cves,
	}, nil
}

func (db *CVEDatabase) GetCVEs(cpe string) []string {
	if db == nil || db.cves == nil {
		return nil
	}
	return db.cves[cpe]
}
