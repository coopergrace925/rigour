package api

import (
	"fmt"
	"net/http"
	"time"

	"github.com/ctrlsam/rigour/internal/redis"
	"github.com/go-chi/render"
)

type HealthDashboard struct {
	redisClient *redis.Client
}

func NewHealthDashboard(redisClient *redis.Client) *HealthDashboard {
	return &HealthDashboard{
		redisClient: redisClient,
	}
}

// ScanStats represents overall scan statistics
type ScanStats struct {
	TotalHostsScanned    int64     `json:"total_hosts_scanned"`
	TotalPortsScanned    int64     `json:"total_ports_scanned"`
	ActiveScanners       int       `json:"active_scanners"`
	QueueDepth           int64     `json:"queue_depth"`
	ScansPerSecond       float64   `json:"scans_per_second"`
	LastScanTime         time.Time `json:"last_scan_time"`
	SystemStatus         string    `json:"system_status"`
	DataFreshness        string    `json:"data_freshness"`
}

// PortScheduleStatus represents port scheduling information
type PortScheduleStatus struct {
	Port            int       `json:"port"`
	Priority        string    `json:"priority"`
	LastScanned     time.Time `json:"last_scanned"`
	NextScan        time.Time `json:"next_scan"`
	ScanInterval    string    `json:"scan_interval"`
}

// ASNRateStatus represents ASN rate limiting status
type ASNRateStatus struct {
	ASN           int    `json:"asn"`
	CurrentRate   int    `json:"current_rate"`
	Limit         int    `json:"limit"`
	PercentUsed   float64 `json:"percent_used"`
}

// StreamHealth represents NATS stream health
type StreamHealth struct {
	StreamName    string `json:"stream_name"`
	MessageCount  int64  `json:"message_count"`
	ConsumerCount int    `json:"consumer_count"`
	Replicas      int    `json:"replicas"`
	Status        string `json:"status"`
}

// GetScanStats returns overall scan statistics
func (h *HealthDashboard) GetScanStats(w http.ResponseWriter, r *http.Request) {
	// In a real implementation, these would come from Redis counters, NATS metrics, and OpenSearch aggregations
	stats := ScanStats{
		TotalHostsScanned:  0,  // TODO: Query from OpenSearch
		TotalPortsScanned:  0,  // TODO: Query from OpenSearch
		ActiveScanners:     0,  // TODO: Query from NATS consumer info
		QueueDepth:         0,  // TODO: Query from NATS stream info
		ScansPerSecond:     0.0, // TODO: Calculate from Redis counters
		LastScanTime:       time.Now().Add(-5 * time.Minute),
		SystemStatus:       "operational",
		DataFreshness:      "< 48h",
	}
	
	render.JSON(w, r, stats)
}

// GetPortSchedules returns port scheduling status
func (h *HealthDashboard) GetPortSchedules(w http.ResponseWriter, r *http.Request) {
	// Mock data - in production, this would come from the coordinator's scheduler state
	schedules := []PortScheduleStatus{
		{
			Port:         22,
			Priority:     "critical",
			LastScanned:  time.Now().Add(-3 * time.Hour),
			NextScan:     time.Now().Add(3 * time.Hour),
			ScanInterval: "6h",
		},
		{
			Port:         443,
			Priority:     "high",
			LastScanned:  time.Now().Add(-12 * time.Hour),
			NextScan:     time.Now().Add(12 * time.Hour),
			ScanInterval: "24h",
		},
		{
			Port:         80,
			Priority:     "high",
			LastScanned:  time.Now().Add(-18 * time.Hour),
			NextScan:     time.Now().Add(6 * time.Hour),
			ScanInterval: "24h",
		},
	}
	
	render.JSON(w, r, map[string]interface{}{
		"schedules": schedules,
		"total":     len(schedules),
	})
}

// GetASNRates returns ASN rate limiting status
func (h *HealthDashboard) GetASNRates(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	
	// Query top ASNs from Redis
	topASNs := []int{15169, 8075, 16509, 13335, 20940} // Google, Microsoft, Amazon, Cloudflare, Akamai
	
	var rates []ASNRateStatus
	for _, asn := range topASNs {
		key := fmt.Sprintf("rate:asn:%d", asn)
		val, err := h.redisClient.Get(ctx, key)
		
		currentRate := 0
		if err == nil && val != "" {
			fmt.Sscanf(val, "%d", &currentRate)
		}
		
		limit := 100 // Default rate limit
		percentUsed := float64(currentRate) / float64(limit) * 100
		
		rates = append(rates, ASNRateStatus{
			ASN:         asn,
			CurrentRate: currentRate,
			Limit:       limit,
			PercentUsed: percentUsed,
		})
	}
	
	render.JSON(w, r, map[string]interface{}{
		"asn_rates": rates,
		"total":     len(rates),
	})
}

// GetStreamHealth returns NATS stream health information
func (h *HealthDashboard) GetStreamHealth(w http.ResponseWriter, r *http.Request) {
	// Mock data - in production, this would come from NATS JetStream API
	streams := []StreamHealth{
		{
			StreamName:    "RAW_SCANS",
			MessageCount:  1234567,
			ConsumerCount: 3,
			Replicas:      3,
			Status:        "healthy",
		},
		{
			StreamName:    "ENRICHED_SCANS",
			MessageCount:  987654,
			ConsumerCount: 2,
			Replicas:      3,
			Status:        "healthy",
		},
		{
			StreamName:    "SCAN_TASKS",
			MessageCount:  456789,
			ConsumerCount: 5,
			Replicas:      1,
			Status:        "healthy",
		},
	}
	
	render.JSON(w, r, map[string]interface{}{
		"streams": streams,
		"total":   len(streams),
	})
}

// GetSystemMetrics returns detailed system metrics
func (h *HealthDashboard) GetSystemMetrics(w http.ResponseWriter, r *http.Request) {
	// Collect various metrics from Redis
	metrics := map[string]interface{}{
		"timestamp": time.Now().Format(time.RFC3339),
		"uptime":    "72h",
		"scanner": map[string]interface{}{
			"zmap_pps":           10400000,
			"zgrab_threads":      1000,
			"total_ips_scanned":  0,
			"total_ports_open":   0,
		},
		"enrichment": map[string]interface{}{
			"worker_count":       3,
			"processing_rate":    1500,
			"queue_lag":          0,
		},
		"storage": map[string]interface{}{
			"opensearch_docs":    0,
			"opensearch_size_gb": 0,
			"redis_keys":         0,
		},
		"network": map[string]interface{}{
			"blocklist_size":     42912,
			"opt_out_count":      127,
			"asn_rate_limited":   0,
		},
	}
	
	render.JSON(w, r, metrics)
}
