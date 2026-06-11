package blocklist

import (
	"net"
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultBlocklistContainsRFC1918(t *testing.T) {
	bl := NewBlocklist()

	privateIPs := []string{"10.0.0.1", "172.16.0.1", "192.168.1.1"}
	for _, ip := range privateIPs {
		if !bl.IsBlocked(net.ParseIP(ip)) {
			t.Errorf("Expected %s to be blocked (RFC1918)", ip)
		}
	}
}

func TestDefaultBlocklistAllowsPublicIPs(t *testing.T) {
	bl := NewBlocklist()

	publicIPs := []string{"1.1.1.1", "8.8.8.8", "93.184.216.34"}
	for _, ip := range publicIPs {
		if bl.IsBlocked(net.ParseIP(ip)) {
			t.Errorf("Expected %s to NOT be blocked", ip)
		}
	}
}

func TestBlocklistBlocksLoopback(t *testing.T) {
	bl := NewBlocklist()
	if !bl.IsBlocked(net.ParseIP("127.0.0.1")) {
		t.Errorf("Expected 127.0.0.1 to be blocked")
	}
}

func TestBlocklistBlocksMulticast(t *testing.T) {
	bl := NewBlocklist()
	if !bl.IsBlocked(net.ParseIP("224.0.0.1")) {
		t.Errorf("Expected 224.0.0.1 to be blocked (multicast)")
	}
}

func TestAddOptOut(t *testing.T) {
	bl := NewBlocklist()
	ip := net.ParseIP("93.184.216.34")

	if bl.IsBlocked(ip) {
		t.Fatalf("IP should not be blocked before opt-out")
	}

	bl.AddOptOut(ip)

	if !bl.IsBlocked(ip) {
		t.Errorf("IP should be blocked after opt-out")
	}
}

func TestGenerateFile(t *testing.T) {
	bl := NewBlocklist()
	tmpDir := t.TempDir()
	outFile := filepath.Join(tmpDir, "blocklist.conf")

	err := bl.GenerateFile(outFile)
	if err != nil {
		t.Fatalf("GenerateFile failed: %v", err)
	}

	data, err := os.ReadFile(outFile)
	if err != nil {
		t.Fatalf("ReadFile failed: %v", err)
	}

	content := string(data)
	if len(content) == 0 {
		t.Error("Blocklist file is empty")
	}
}

func TestBlocklistHandlesNilIP(t *testing.T) {
	bl := NewBlocklist()

	// Should not panic
	if bl.IsBlocked(nil) {
		t.Error("Expected nil IP to not be blocked")
	}

	// Should not panic
	bl.AddOptOut(nil)
}
