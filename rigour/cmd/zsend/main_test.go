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

func TestParseZGrab2JSONLineWithPort(t *testing.T) {
	input := `{"ip":"1.2.3.4","port":8080,"data":{"http":{"status":"success","result":{"response":{"status_code":200}}}}}`
	result, err := parseZGrab2JSONLine([]byte(input))
	if err != nil {
		t.Fatalf("parseZGrab2JSONLine failed: %v", err)
	}
	if result.IP != "1.2.3.4" {
		t.Errorf("IP mismatch: got %s, want 1.2.3.4", result.IP)
	}
	if result.Port != 8080 {
		t.Errorf("Port mismatch: got %d, want 8080", result.Port)
	}
}

func TestParseZMapCSVLineInvalidIP(t *testing.T) {
	line := "999.999.999.999,443,synack,1"
	_, err := parseZMapCSVLine(line)
	if err == nil {
		t.Fatal("Expected error for invalid IP, got nil")
	}
}

func TestParseZGrab2JSONLineInvalidIP(t *testing.T) {
	input := `{"ip":"not.an.ip.address","port":8080,"data":{"http":{"status":"success"}}}`
	_, err := parseZGrab2JSONLine([]byte(input))
	if err == nil {
		t.Fatal("Expected error for invalid IP, got nil")
	}
}

func TestIsValidIPv4(t *testing.T) {
	tests := []struct {
		ip    string
		valid bool
	}{
		{"1.2.3.4", true},
		{"192.168.1.1", true},
		{"255.255.255.255", true},
		{"0.0.0.0", true},
		{"256.1.1.1", false},
		{"1.2.3", false},
		{"1.2.3.4.5", false},
		{"not.an.ip", false},
		{"", false},
	}

	for _, tt := range tests {
		result := isValidIPv4(tt.ip)
		if result != tt.valid {
			t.Errorf("isValidIPv4(%q) = %v, want %v", tt.ip, result, tt.valid)
		}
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
