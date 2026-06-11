package types

import (
	"encoding/json"
	"testing"
	"time"
)

func TestRawScanJSON(t *testing.T) {
	scan := RawScan{
		IP:         "1.2.3.4",
		Port:       443,
		Protocol:   "tcp",
		Service:    "https",
		Banner:     "nginx/1.24.0",
		ScannedAt:  time.Now(),
		ScannerID:  "scanner-01",
	}

	data, err := json.Marshal(scan)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	var decoded RawScan
	err = json.Unmarshal(data, &decoded)
	if err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if decoded.IP != scan.IP {
		t.Errorf("IP mismatch: got %s, want %s", decoded.IP, scan.IP)
	}
}
