package coordinator

import (
	"testing"
)

func TestSplitSubnets(t *testing.T) {
	cidr := "192.168.0.0/22" // Contains 4 /24 subnets
	subnets, err := SplitInto24(cidr)
	if err != nil {
		t.Fatalf("Split failed: %v", err)
	}
	if len(subnets) != 4 {
		t.Errorf("Expected 4 subnets, got %d", len(subnets))
	}
}
