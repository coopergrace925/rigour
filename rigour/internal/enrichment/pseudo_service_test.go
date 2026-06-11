package enrichment

import (
	"fmt"
	"testing"
)

// Port represents a simple port/banner pair for pseudo-service detection
type Port struct {
	Port   int
	Banner string
}

func TestIsPseudoServiceReturnsFalseForFewPorts(t *testing.T) {
	detector := NewPseudoServiceDetector(20)

	ports := []Port{
		{Port: 80, Banner: "nginx/1.24"},
		{Port: 443, Banner: "nginx/1.24"},
		{Port: 22, Banner: "OpenSSH_8.9"},
	}

	if detector.IsPseudoService("1.2.3.4", ports) {
		t.Error("Should NOT be pseudo-service with 3 ports")
	}
}

func TestIsPseudoServiceReturnsTrueForManyIdenticalBanners(t *testing.T) {
	detector := NewPseudoServiceDetector(20)

	// Simulate a honeypot: 25 ports with identical banners
	var ports []Port
	for i := 1; i <= 25; i++ {
		ports = append(ports, Port{
			Port:   i,
			Banner: "honeypot-identical-response",
		})
	}

	if !detector.IsPseudoService("1.2.3.4", ports) {
		t.Error("Should BE pseudo-service with 25 identical banners")
	}
}

func TestIsPseudoServiceReturnsFalseForDiverseBanners(t *testing.T) {
	detector := NewPseudoServiceDetector(20)

	var ports []Port
	for i := 1; i <= 25; i++ {
		ports = append(ports, Port{
			Port:   i,
			Banner: fmt.Sprintf("service-%d/v%d.0", i, i),
		})
	}

	if detector.IsPseudoService("1.2.3.4", ports) {
		t.Error("Should NOT be pseudo-service with diverse banners")
	}
}
