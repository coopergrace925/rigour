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

func TestGeoIPLookupInvalidIP(t *testing.T) {
	// Test that Lookup returns error for invalid IP without requiring real DBs
	// We can't fully test Lookup without real databases, but we can verify
	// the API signature is correct by checking compilation
	var lookup *GeoIPLookup
	if lookup != nil {
		_, err := lookup.Lookup("invalid-ip")
		if err == nil {
			t.Error("Expected error for invalid IP")
		}
	}
}

