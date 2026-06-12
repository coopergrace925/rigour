package api

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/ctrlsam/rigour/internal/redis"
	"github.com/go-chi/render"
)

type RateLimiter struct {
	redis *redis.Client
}

func NewRateLimiter(redisClient *redis.Client) *RateLimiter {
	return &RateLimiter{
		redis: redisClient,
	}
}

// RateLimitMiddleware enforces per-IP and per-API-key rate limits
func (rl *RateLimiter) RateLimitMiddleware(requestsPerMinute int) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			
			// Extract API key or IP address
			identifier := rl.getIdentifier(r)
			
			// Check rate limit
			allowed, err := rl.checkRateLimit(ctx, identifier, requestsPerMinute)
			if err != nil {
				render.Status(r, http.StatusInternalServerError)
				render.JSON(w, r, map[string]string{"error": "Rate limit check failed"})
				return
			}
			
			if !allowed {
				render.Status(r, http.StatusTooManyRequests)
				render.JSON(w, r, map[string]string{"error": "Rate limit exceeded. Try again later."})
				return
			}
			
			next.ServeHTTP(w, r)
		})
	}
}

// APIKeyAuthMiddleware validates API keys
func (rl *RateLimiter) APIKeyAuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Extract API key from header or query param
		apiKey := r.Header.Get("X-API-Key")
		if apiKey == "" {
			apiKey = r.URL.Query().Get("api_key")
		}
		
		// If no API key provided, allow anonymous access with stricter rate limits
		if apiKey == "" {
			next.ServeHTTP(w, r)
			return
		}
		
		// Validate API key
		ctx := r.Context()
		valid, err := rl.validateAPIKey(ctx, apiKey)
		if err != nil || !valid {
			render.Status(r, http.StatusUnauthorized)
			render.JSON(w, r, map[string]string{"error": "Invalid API key"})
			return
		}
		
		// Store validated API key in context for downstream use
		ctx = context.WithValue(ctx, "api_key", apiKey)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (rl *RateLimiter) getIdentifier(r *http.Request) string {
	// Try to get API key first
	apiKey := r.Header.Get("X-API-Key")
	if apiKey == "" {
		apiKey = r.URL.Query().Get("api_key")
	}
	
	if apiKey != "" {
		// Hash the API key for privacy
		hash := sha256.Sum256([]byte(apiKey))
		return "apikey:" + hex.EncodeToString(hash[:])
	}
	
	// Fall back to IP address
	ip := getClientIP(r)
	return "ip:" + ip
}

func (rl *RateLimiter) checkRateLimit(ctx context.Context, identifier string, requestsPerMinute int) (bool, error) {
	key := "ratelimit:" + identifier
	
	// Use Redis INCR + EXPIRE for sliding window rate limiting
	count, err := rl.redis.Incr(ctx, key)
	if err != nil {
		return false, err
	}
	
	// Set expiry on first request
	if count == 1 {
		if err := rl.redis.Expire(ctx, key, 60*time.Second); err != nil {
			return false, err
		}
	}
	
	return count <= int64(requestsPerMinute), nil
}

func (rl *RateLimiter) validateAPIKey(ctx context.Context, apiKey string) (bool, error) {
	// Check if API key exists in Redis set
	key := "apikeys:valid"
	exists, err := rl.redis.SIsMember(ctx, key, apiKey)
	if err != nil {
		return false, err
	}
	
	return exists, nil
}

func getClientIP(r *http.Request) string {
	// Try X-Forwarded-For first
	xff := r.Header.Get("X-Forwarded-For")
	if xff != "" {
		ips := strings.Split(xff, ",")
		if len(ips) > 0 {
			return strings.TrimSpace(ips[0])
		}
	}
	
	// Try X-Real-IP
	xri := r.Header.Get("X-Real-IP")
	if xri != "" {
		return xri
	}
	
	// Fall back to RemoteAddr
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	
	return ip
}
