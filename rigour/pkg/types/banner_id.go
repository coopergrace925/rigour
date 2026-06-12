package types

import (
	"crypto/sha256"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// GenerateBannerID creates a unique banner ID for deduplication
// Format: UUID v4 for uniqueness
// Compatible with Shodan's _shodan.id pattern
func GenerateBannerID() string {
	return uuid.New().String()
}

// GenerateDeterministicBannerID creates a deterministic ID based on scan attributes
// Useful for detecting exact duplicates across multiple scanner nodes
// Format: SHA256(ip:port:timestamp_truncated:scanner_id)
func GenerateDeterministicBannerID(ip string, port int, scannedAt time.Time, scannerID string) string {
	// Truncate timestamp to minute for dedup window
	truncated := scannedAt.Truncate(time.Minute).Unix()
	input := fmt.Sprintf("%s:%d:%d:%s", ip, port, truncated, scannerID)
	hash := sha256.Sum256([]byte(input))
	return fmt.Sprintf("%x", hash[:16]) // Use first 16 bytes (32 hex chars)
}

// GenerateBannerIDWithContext creates a banner ID with optional deterministic mode
// Use deterministic=true for multi-node deduplication, false for unique IDs
func GenerateBannerIDWithContext(ip string, port int, scannedAt time.Time, scannerID string, deterministic bool) string {
	if deterministic {
		return GenerateDeterministicBannerID(ip, port, scannedAt, scannerID)
	}
	return GenerateBannerID()
}
