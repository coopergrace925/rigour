package api

import (
	"encoding/json"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/ctrlsam/rigour/internal/blocklist"
	"github.com/ctrlsam/rigour/internal/redis"
)

// OptOutRequest represents an opt-out submission
type OptOutRequest struct {
	IPOrCIDR string `json:"ip_or_cidr"` // Can be "1.2.3.4" or "1.2.3.0/24"
	Email    string `json:"email"`      // Contact email for verification
	Reason   string `json:"reason"`     // Optional reason for opt-out
}

// OptOutResponse represents the API response
type OptOutResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	OptOut  struct {
		IPOrCIDR  string    `json:"ip_or_cidr"`
		AddedAt   time.Time `json:"added_at"`
		ExpiresAt time.Time `json:"expires_at,omitempty"` // Future: time-limited opt-outs
	} `json:"opt_out,omitempty"`
}

// OptOutHandler handles opt-out requests
type OptOutHandler struct {
	blocklist   *blocklist.Blocklist
	redisClient *redis.Client
}

// NewOptOutHandler creates a new opt-out handler
func NewOptOutHandler(bl *blocklist.Blocklist, rc *redis.Client) *OptOutHandler {
	return &OptOutHandler{
		blocklist:   bl,
		redisClient: rc,
	}
}

// HandleOptOut processes POST /api/opt-out requests
func (h *OptOutHandler) HandleOptOut(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req OptOutRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, "Invalid JSON payload", http.StatusBadRequest)
		return
	}

	// Validate input
	if req.IPOrCIDR == "" {
		writeError(w, "ip_or_cidr is required", http.StatusBadRequest)
		return
	}

	if req.Email == "" {
		writeError(w, "email is required for verification", http.StatusBadRequest)
		return
	}

	// Validate email format (basic check)
	if !strings.Contains(req.Email, "@") || !strings.Contains(req.Email, ".") {
		writeError(w, "invalid email format", http.StatusBadRequest)
		return
	}

	// Parse and validate IP or CIDR
	isCIDR := strings.Contains(req.IPOrCIDR, "/")
	
	if isCIDR {
		// Validate CIDR
		_, ipnet, err := net.ParseCIDR(req.IPOrCIDR)
		if err != nil {
			writeError(w, "invalid CIDR format", http.StatusBadRequest)
			return
		}

		// Security check: prevent opt-out of entire internet
		ones, _ := ipnet.Mask.Size()
		if ones < 8 {
			writeError(w, "CIDR block too large (minimum /8)", http.StatusBadRequest)
			return
		}

		// Add to blocklist
		if err := h.blocklist.AddOptOutCIDR(req.IPOrCIDR); err != nil {
			writeError(w, "failed to add CIDR opt-out", http.StatusInternalServerError)
			return
		}

		// Persist to Redis (simplified - use basic Set instead of SetJSON)
		if h.redisClient != nil {
			key := "optout:cidr:" + ipnet.String()
			value := req.Email + "|" + req.Reason
			_ = h.redisClient.Set(r.Context(), key, value, 0) // No expiration
		}

	} else {
		// Validate IP
		ip := net.ParseIP(req.IPOrCIDR)
		if ip == nil {
			writeError(w, "invalid IP address", http.StatusBadRequest)
			return
		}

		// Add to blocklist
		h.blocklist.AddOptOut(ip)

		// Persist to Redis (simplified - use basic Set instead of SetJSON)
		if h.redisClient != nil {
			key := "optout:ip:" + ip.String()
			value := req.Email + "|" + req.Reason
			_ = h.redisClient.Set(r.Context(), key, value, 0) // No expiration
		}
	}

	// Success response
	resp := OptOutResponse{
		Success: true,
		Message: "Successfully added to opt-out list. Scans will stop within 24 hours.",
	}
	resp.OptOut.IPOrCIDR = req.IPOrCIDR
	resp.OptOut.AddedAt = time.Now()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(resp)
}

// HandleOptOutList returns all current opt-outs (GET /api/opt-out)
func (h *OptOutHandler) HandleOptOutList(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	ips, cidrs := h.blocklist.GetOptOuts()

	resp := map[string]interface{}{
		"total_ips":   len(ips),
		"total_cidrs": len(cidrs),
		"ips":         ips,
		"cidrs":       cidrs,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func writeError(w http.ResponseWriter, message string, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(OptOutResponse{
		Success: false,
		Message: message,
	})
}
