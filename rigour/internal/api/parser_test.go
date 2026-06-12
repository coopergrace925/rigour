package api

import (
	"testing"
)

func TestParseShodanQuery(t *testing.T) {
	q := `port:22 country:US asn:AS15169 org:"Google LLC" cve:CVE-2023-44487 apache`
	filters := ParseShodanQuery(q)

	if filters.Port != 22 {
		t.Errorf("Expected port 22, got %d", filters.Port)
	}
	if filters.Country != "US" {
		t.Errorf("Expected country US, got %s", filters.Country)
	}
	if filters.ASN != 15169 {
		t.Errorf("Expected ASN 15169, got %d", filters.ASN)
	}
	if filters.Org != "Google LLC" {
		t.Errorf("Expected org Google LLC, got %s", filters.Org)
	}
	if filters.CVE != "CVE-2023-44487" {
		t.Errorf("Expected CVE CVE-2023-44487, got %s", filters.CVE)
	}
	if len(filters.FreeText) != 1 || filters.FreeText[0] != "apache" {
		t.Errorf("Expected free text [apache], got %v", filters.FreeText)
	}
}
