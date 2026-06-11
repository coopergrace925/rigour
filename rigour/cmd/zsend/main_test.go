package main

import (
	"testing"
)

func TestParseZMapCSVLine(t *testing.T) {
	line := "1.2.3.4,443,synack,1"
	result, err := parseZMapCSVLine(line)
	if err != nil {
		t.Fatalf("parseZMapCSVLine failed: %v", err)
	}
	if result.IP != "1.2.3.4" {
		t.Errorf("IP mismatch: got %s, want 1.2.3.4", result.IP)
	}
	if result.Port != 443 {
		t.Errorf("Port mismatch: got %d, want 443", result.Port)
	}
}

func TestParseZGrab2JSONLine(t *testing.T) {
	input := `{"ip":"1.2.3.4","data":{"http":{"status":"success","result":{"response":{"status_code":200}}}}}`
	result, err := parseZGrab2JSONLine([]byte(input))
	if err != nil {
		t.Fatalf("parseZGrab2JSONLine failed: %v", err)
	}
	if result.IP != "1.2.3.4" {
		t.Errorf("IP mismatch: got %s, want 1.2.3.4", result.IP)
	}
}

func TestBuildRawScan(t *testing.T) {
	scan := buildRawScan("1.2.3.4", 443, "tcp", "scanner-01")
	if scan.IP != "1.2.3.4" {
		t.Errorf("IP mismatch")
	}
	if scan.Port != 443 {
		t.Errorf("Port mismatch")
	}
	if scan.ScannerID != "scanner-01" {
		t.Errorf("ScannerID mismatch")
	}
}
