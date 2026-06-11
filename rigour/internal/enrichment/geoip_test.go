package enrichment

import (
	"testing"
)

func TestGeoIPLookupResult(t *testing.T) {
	result := GeoResult{
		Country: "US",
		City:    "San Francisco",
		ASN:     15169,
		Org:     "Google LLC",
	}

	if result.Country != "US" {
		t.Errorf("Expected country US, got %s", result.Country)
	}
	if result.ASN != 15169 {
		t.Errorf("Expected ASN 15169, got %d", result.ASN)
	}
}

func TestNewGeoIPLookupReturnsErrorForMissingFile(t *testing.T) {
	_, err := NewGeoIPLookup("/nonexistent/path.mmdb", "/nonexistent/asn.mmdb")
	if err == nil {
		t.Error("Expected error for missing MMDB file")
	}
}
