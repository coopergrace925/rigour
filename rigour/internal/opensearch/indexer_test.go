package opensearch

import (
	"testing"
	"time"

	"github.com/ctrlsam/rigour/pkg/types"
)

func TestMergePorts(t *testing.T) {
	existing := types.HostDocument{
		IP:       "192.168.1.100",
		ASN:      12345,
		Org:      "Old Org",
		Country:  "US",
		RDNS:     "old.example.com",
		LastSeen: time.Now().Add(-48 * time.Hour),
		IsStale:  true,
		Ports: []types.Port{
			{Port: 80, Protocol: "tcp", Service: "http", LastSeen: time.Now().Add(-24 * time.Hour)},
		},
		CVEs: []string{"CVE-2021-1234"},
	}

	newScan := types.EnrichedScan{
		RawScan: types.RawScan{
			IP:       "192.168.1.100",
			Port:     443,
			Protocol: "tcp",
			Service:  "https",
			Banner:   "nginx/1.18.0",
			ZGrabData: types.ZGrabData{
				HTTP: &types.HTTPInfo{StatusCode: 200, Title: "Welcome"},
			},
		},
		ASN:     12345,
		Org:     "New Org",
		Country: "US",
		RDNS:    "new.example.com",
		CVEs:    []string{"CVE-2022-5678"},
	}

	merged := MergePorts(existing, newScan)

	// Should have both ports
	if len(merged.Ports) != 2 {
		t.Errorf("expected 2 ports, got %d", len(merged.Ports))
	}

	// Should update host metadata
	if merged.Org != "New Org" {
		t.Errorf("expected Org 'New Org', got '%s'", merged.Org)
	}
	if merged.RDNS != "new.example.com" {
		t.Errorf("expected RDNS 'new.example.com', got '%s'", merged.RDNS)
	}
	if merged.IsStale {
		t.Error("expected IsStale to be false after merge")
	}

	// Should deduplicate CVEs
	if len(merged.CVEs) != 2 {
		t.Errorf("expected 2 CVEs, got %d", len(merged.CVEs))
	}

	// Check new port was added
	var found443 bool
	for _, p := range merged.Ports {
		if p.Port == 443 {
			found443 = true
			if p.Service != "https" {
				t.Errorf("expected service 'https', got '%s'", p.Service)
			}
			if p.Banner != "nginx/1.18.0" {
				t.Errorf("expected banner 'nginx/1.18.0', got '%s'", p.Banner)
			}
			if p.HTTP == nil || p.HTTP.StatusCode != 200 {
				t.Error("expected HTTP data to be preserved")
			}
			if p.HTTP.Title != "Welcome" {
				t.Errorf("expected HTTP title 'Welcome', got '%s'", p.HTTP.Title)
			}
		}
	}
	if !found443 {
		t.Error("port 443 not found in merged result")
	}
}

func TestMergePortsUpdatesExistingPort(t *testing.T) {
	existing := types.HostDocument{
		IP: "10.0.0.1",
		Ports: []types.Port{
			{Port: 22, Protocol: "tcp", Service: "ssh", Banner: "OpenSSH 7.4", LastSeen: time.Now().Add(-2 * time.Hour)},
		},
	}

	newScan := types.EnrichedScan{
		RawScan: types.RawScan{
			IP:       "10.0.0.1",
			Port:     22,
			Protocol: "tcp",
			Service:  "ssh",
			Banner:   "OpenSSH 8.0",
		},
	}

	merged := MergePorts(existing, newScan)

	if len(merged.Ports) != 1 {
		t.Errorf("expected 1 port, got %d", len(merged.Ports))
	}

	port := merged.Ports[0]
	if port.Banner != "OpenSSH 8.0" {
		t.Errorf("expected updated banner 'OpenSSH 8.0', got '%s'", port.Banner)
	}

	if time.Since(port.LastSeen) > 5*time.Second {
		t.Error("expected LastSeen to be updated to now")
	}
}

func TestMergePortsPrunesStale(t *testing.T) {
	existing := types.HostDocument{
		IP: "172.16.0.1",
		Ports: []types.Port{
			{Port: 80, Protocol: "tcp", Service: "http", LastSeen: time.Now().Add(-8 * 24 * time.Hour)},  // 8 days ago - stale
			{Port: 443, Protocol: "tcp", Service: "https", LastSeen: time.Now().Add(-3 * 24 * time.Hour)}, // 3 days ago - fresh
		},
	}

	newScan := types.EnrichedScan{
		RawScan: types.RawScan{
			IP:       "172.16.0.1",
			Port:     22,
			Protocol: "tcp",
			Service:  "ssh",
		},
	}

	merged := MergePorts(existing, newScan)

	// Should have port 443 (fresh) and port 22 (new), but not port 80 (stale)
	if len(merged.Ports) != 2 {
		t.Errorf("expected 2 ports after pruning, got %d", len(merged.Ports))
	}

	var has80, has443, has22 bool
	for _, p := range merged.Ports {
		switch p.Port {
		case 80:
			has80 = true
		case 443:
			has443 = true
		case 22:
			has22 = true
		}
	}

	if has80 {
		t.Error("port 80 should have been pruned (stale)")
	}
	if !has443 {
		t.Error("port 443 should still be present (fresh)")
	}
	if !has22 {
		t.Error("port 22 should be present (newly scanned)")
	}
}
