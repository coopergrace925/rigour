package enrichment

import (
	"context"
	"testing"
)

func TestRDNSResolverInterface(t *testing.T) {
	resolver := NewRDNSResolver()
	ctx := context.Background()
	
	// Should fail on empty
	_, err := resolver.Resolve(ctx, "")
	if err == nil {
		t.Error("Should fail on empty IP")
	}
}
