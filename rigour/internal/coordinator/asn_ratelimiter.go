package coordinator

import (
	"context"
	"fmt"
	"log"
	"net"
	"time"

	"github.com/ctrlsam/rigour/internal/redis"
)

// ASNRateLimiter enforces per-ASN scan rate limits to avoid complaints
type ASNRateLimiter struct {
	redisClient *redis.Client
	defaultRate int // scans per minute per ASN
}

// NewASNRateLimiter creates a new ASN rate limiter
func NewASNRateLimiter(redisClient *redis.Client, defaultRate int) *ASNRateLimiter {
	return &ASNRateLimiter{
		redisClient: redisClient,
		defaultRate: defaultRate,
	}
}

// CheckAndIncrement checks if an ASN can be scanned and increments the counter
// Returns true if the scan is allowed, false if rate limit exceeded
func (r *ASNRateLimiter) CheckAndIncrement(ctx context.Context, asn int) (bool, error) {
	key := fmt.Sprintf("rate:asn:%d", asn)
	
	// Get current count
	count, err := r.redisClient.Get(ctx, key)
	if err != nil && err.Error() != "redis: nil" {
		return false, fmt.Errorf("failed to get rate limit: %w", err)
	}
	
	// Parse count
	var currentCount int
	if count != "" {
		fmt.Sscanf(count, "%d", &currentCount)
	}
	
	// Check if limit exceeded
	if currentCount >= r.defaultRate {
		log.Printf("Rate limit exceeded for ASN %d: %d/%d scans/min", asn, currentCount, r.defaultRate)
		return false, nil
	}
	
	// Increment and set expiry
	newCount := currentCount + 1
	err = r.redisClient.Set(ctx, key, fmt.Sprintf("%d", newCount), 60*time.Second)
	if err != nil {
		return false, fmt.Errorf("failed to set rate limit: %w", err)
	}
	
	return true, nil
}

// GetASNForIP looks up the ASN for an IP address (simplified - in production use BGP data)
func (r *ASNRateLimiter) GetASNForIP(ip string) (int, error) {
	parsedIP := net.ParseIP(ip)
	if parsedIP == nil {
		return 0, fmt.Errorf("invalid IP: %s", ip)
	}
	
	// Simplified ASN mapping (in production, use MaxMind GeoIP2 ASN database or BGP data)
	// For now, derive a pseudo-ASN from the first octet for testing
	ipv4 := parsedIP.To4()
	if ipv4 == nil {
		return 0, fmt.Errorf("IPv6 not supported in simplified ASN lookup")
	}
	
	// Simple heuristic: map first octet ranges to common ASNs
	firstOctet := int(ipv4[0])
	switch {
	case firstOctet >= 1 && firstOctet <= 9:
		return 3356, nil // Level3/Lumen
	case firstOctet >= 10 && firstOctet <= 19:
		return 7018, nil // AT&T
	case firstOctet >= 20 && firstOctet <= 50:
		return 15169, nil // Google
	case firstOctet >= 51 && firstOctet <= 100:
		return 8075, nil // Microsoft
	case firstOctet >= 101 && firstOctet <= 150:
		return 16509, nil // Amazon
	case firstOctet >= 151 && firstOctet <= 192:
		return 13335, nil // Cloudflare
	case firstOctet >= 193 && firstOctet <= 223:
		return 20940, nil // Akamai
	default:
		return 0, nil // Unknown/Private
	}
}

// ResetASNCounter manually resets the rate limit for an ASN (for testing/admin)
func (r *ASNRateLimiter) ResetASNCounter(ctx context.Context, asn int) error {
	key := fmt.Sprintf("rate:asn:%d", asn)
	return r.redisClient.Del(ctx, key)
}

// GetASNStats returns current rate limit stats for an ASN
func (r *ASNRateLimiter) GetASNStats(ctx context.Context, asn int) (int, int, error) {
	key := fmt.Sprintf("rate:asn:%d", asn)
	
	count, err := r.redisClient.Get(ctx, key)
	if err != nil && err.Error() != "redis: nil" {
		return 0, r.defaultRate, fmt.Errorf("failed to get rate limit: %w", err)
	}
	
	var currentCount int
	if count != "" {
		fmt.Sscanf(count, "%d", &currentCount)
	}
	
	return currentCount, r.defaultRate, nil
}
