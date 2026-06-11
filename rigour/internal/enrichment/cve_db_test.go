package enrichment

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCVEDatabaseEmpty(t *testing.T) {
	_, err := OpenCVEDatabase("/nonexistent/cve_index.bin")
	if err == nil {
		t.Error("Expected error opening nonexistent DB file")
	}
}

func TestCVEDatabaseLookup(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "cve_index.bin")

	f, err := os.Create(dbPath)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	f.Write([]byte{0, 0, 0, 1})
	f.Write([]byte{0, 8})
	f.Write([]byte("cpe:test"))
	f.Write([]byte{0, 11})
	f.Write([]byte("CVE-1,CVE-2"))
	f.Close()

	db, err := OpenCVEDatabase(dbPath)
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}

	cves := db.GetCVEs("cpe:test")
	if len(cves) != 2 || cves[0] != "CVE-1" || cves[1] != "CVE-2" {
		t.Errorf("Lookup failed, got %+v", cves)
	}
}
