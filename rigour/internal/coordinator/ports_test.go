package coordinator

import (
	"testing"
)

func TestPortRegistry(t *testing.T) {
	pr, err := NewPortRegistry()
	if err != nil {
		t.Fatalf("Failed to create port registry: %v", err)
	}

	// Test total port count (should be 3,846 from Shodan)
	allPorts := pr.GetAllPorts()
	if len(allPorts) < 3800 {
		t.Errorf("Expected ~3,846 ports, got %d", len(allPorts))
	}

	// Test tier statistics
	stats := pr.GetTierStats()
	t.Logf("Port distribution: %+v", stats)

	if stats["critical"] < 5 {
		t.Errorf("Expected at least 5 critical ports, got %d", stats["critical"])
	}

	if stats["total"] != len(allPorts) {
		t.Errorf("Total mismatch: stats=%d, actual=%d", stats["total"], len(allPorts))
	}

	// Test specific port lookup
	port22, exists := pr.GetPortInfo(22)
	if !exists {
		t.Errorf("Port 22 (SSH) should exist in registry")
	}
	if port22.Tier != TierCritical {
		t.Errorf("Port 22 should be TierCritical, got %v", port22.Tier)
	}
	if port22.Service != "ssh" {
		t.Errorf("Port 22 service should be 'ssh', got %s", port22.Service)
	}

	// Test Shodan ports are loaded
	port443, exists := pr.GetPortInfo(443)
	if !exists {
		t.Errorf("Port 443 (HTTPS) should exist in registry")
	}
	if port443.Tier != TierHigh {
		t.Errorf("Port 443 should be TierHigh, got %v", port443.Tier)
	}
}

func TestSchedulerWithRegistry(t *testing.T) {
	s := NewScheduler()

	summary := s.GetScheduleSummary()
	t.Logf("Scheduler summary: %+v", summary)

	totalPorts := summary["total_ports"].(int)
	if totalPorts < 3800 {
		t.Errorf("Expected ~3,846 ports in scheduler, got %d", totalPorts)
	}

	// Verify all tiers are represented
	if summary["critical_ports"].(int) == 0 {
		t.Errorf("No critical ports in scheduler")
	}
	if summary["high_ports"].(int) == 0 {
		t.Errorf("No high priority ports in scheduler")
	}
	if summary["shodan_full_ports"].(int) == 0 {
		t.Errorf("No Shodan ports in scheduler")
	}
}
