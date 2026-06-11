package enrichment

import (
	"crypto/sha256"
	"fmt"
	"strings"
)

type PseudoServiceDetector struct {
	threshold int // Number of identical banners to trigger detection
}

func NewPseudoServiceDetector(threshold int) *PseudoServiceDetector {
	if threshold <= 0 {
		threshold = 20
	}
	return &PseudoServiceDetector{threshold: threshold}
}

// IsPseudoService checks if a host is a fake responder.
// The ip parameter is reserved for future logging/reporting capabilities.
// Returns true if more than `threshold` ports respond with identical banners.
func (d *PseudoServiceDetector) IsPseudoService(ip string, ports []Port) bool {
	if len(ports) < d.threshold {
		return false
	}

	bannerCounts := make(map[string]int)
	for _, p := range ports {
		hash := hashBanner(p.Banner)
		bannerCounts[hash]++
	}

	for _, count := range bannerCounts {
		if count >= d.threshold {
			return true
		}
	}

	return false
}

func hashBanner(banner string) string {
	normalized := strings.TrimSpace(strings.ToLower(banner))
	if normalized == "" {
		normalized = "__empty__"
	}
	h := sha256.Sum256([]byte(normalized))
	return fmt.Sprintf("%x", h[:8])
}
