package enrichment

import (
	"context"
	"fmt"
	"net"
	"sync"
	"time"
)

type LocalRDNSResolver struct {
	mu    sync.RWMutex
	cache map[string]cacheEntry
}

type cacheEntry struct {
	hostname  string
	expiresAt time.Time
}

func NewRDNSResolver() *LocalRDNSResolver {
	return &LocalRDNSResolver{
		cache: make(map[string]cacheEntry),
	}
}

func (r *LocalRDNSResolver) Resolve(ctx context.Context, ipStr string) (string, error) {
	if ipStr == "" {
		return "", fmt.Errorf("empty IP address")
	}

	r.mu.RLock()
	entry, ok := r.cache[ipStr]
	r.mu.RUnlock()

	if ok && time.Now().Before(entry.expiresAt) {
		return entry.hostname, nil
	}

	// For Phase 2, we implement standard Go net lookup with parallel workers
	// rather than compiling external C libraries for raw DNS packets.
	names, err := net.LookupAddr(ipStr)
	if err != nil {
		return "", fmt.Errorf("failed to lookup addr: %w", err)
	}

	var hostname string
	if len(names) > 0 {
		hostname = names[0]
	}

	r.mu.Lock()
	r.cache[ipStr] = cacheEntry{
		hostname:  hostname,
		expiresAt: time.Now().Add(1 * time.Hour),
	}
	r.mu.Unlock()

	return hostname, nil
}
