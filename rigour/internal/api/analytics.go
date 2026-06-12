package api

import (
	"fmt"
	"net/http"
	"time"

	"github.com/ctrlsam/rigour/internal/clickhouse"
	"github.com/go-chi/render"
)

type AnalyticsHandler struct {
	clickhouse *clickhouse.Client
}

func NewAnalyticsHandler(ch *clickhouse.Client) *AnalyticsHandler {
	return &AnalyticsHandler{
		clickhouse: ch,
	}
}

type AnalyticsOverview struct {
	TotalScans         int64     `json:"total_scans"`
	UniqueHosts        int64     `json:"unique_hosts"`
	TopServices        []ServiceStat `json:"top_services"`
	RecentCVEs         []CVETrend `json:"recent_cves"`
	TopASNs            []ASNStat `json:"top_asns"`
	ScanTrend          []ScanTrendPoint `json:"scan_trend"`
}

type ServiceStat struct {
	Service    string `json:"service"`
	Port       uint16 `json:"port"`
	HostCount  uint64 `json:"host_count"`
	TotalScans uint64 `json:"total_scans"`
}

type CVETrend struct {
	Date          time.Time `json:"date"`
	CVEID         string    `json:"cve_id"`
	AffectedHosts uint64    `json:"affected_hosts"`
}

type ASNStat struct {
	ASN       uint32 `json:"asn"`
	Country   string `json:"country"`
	ScanCount uint64 `json:"scan_count"`
	UniqueIPs uint32 `json:"unique_ips"`
}

type ScanTrendPoint struct {
	Date      time.Time `json:"date"`
	Scans     uint64    `json:"scans"`
	NewHosts  uint64    `json:"new_hosts"`
}

func (h *AnalyticsHandler) GetOverview(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	overview := AnalyticsOverview{
		TotalScans:  0,
		UniqueHosts: 0,
		TopServices: []ServiceStat{},
		RecentCVEs:  []CVETrend{},
		TopASNs:     []ASNStat{},
		ScanTrend:   []ScanTrendPoint{},
	}

	if h.clickhouse != nil {
		serviceStats, _ := h.clickhouse.GetServiceStats(ctx, time.Now().AddDate(0, 0, -7), time.Now())
		for _, s := range serviceStats {
			overview.TopServices = append(overview.TopServices, ServiceStat{
				Service:    s.Service,
				Port:       s.Port,
				HostCount:  s.HostCount,
				TotalScans: s.TotalScans,
			})
		}

		cveTrends, _ := h.clickhouse.GetCVETrends(ctx, 7)
		for _, c := range cveTrends {
			overview.RecentCVEs = append(overview.RecentCVEs, CVETrend{
				Date:          c.Date,
				CVEID:         c.CVEID,
				AffectedHosts: c.AffectedHosts,
			})
		}

		asnStats, _ := h.clickhouse.GetASNStats(ctx, 24)
		for _, a := range asnStats {
			overview.TopASNs = append(overview.TopASNs, ASNStat{
				ASN:       a.ASN,
				Country:   a.Country,
				ScanCount: a.ScanCount,
				UniqueIPs: a.UniqueIPs,
			})
		}
	}

	render.JSON(w, r, overview)
}

func (h *AnalyticsHandler) GetServiceAnalytics(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	days := 7
	if d := r.URL.Query().Get("days"); d != "" {
		fmt.Sscanf(d, "%d", &days)
	}

	stats, err := h.clickhouse.GetServiceStats(ctx, time.Now().AddDate(0, 0, -days), time.Now())
	if err != nil {
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, map[string]string{"error": err.Error()})
		return
	}

	render.JSON(w, r, map[string]interface{}{
		"services": stats,
		"period_days": days,
	})
}

func (h *AnalyticsHandler) GetCVEAnalytics(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	days := 30
	if d := r.URL.Query().Get("days"); d != "" {
		fmt.Sscanf(d, "%d", &days)
	}

	trends, err := h.clickhouse.GetCVETrends(ctx, days)
	if err != nil {
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, map[string]string{"error": err.Error()})
		return
	}

	render.JSON(w, r, map[string]interface{}{
		"cve_trends": trends,
		"period_days": days,
	})
}

func (h *AnalyticsHandler) GetASNAnalytics(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	hours := 24
	if h := r.URL.Query().Get("hours"); h != "" {
		fmt.Sscanf(h, "%d", &hours)
	}

	stats, err := h.clickhouse.GetASNStats(ctx, hours)
	if err != nil {
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, map[string]string{"error": err.Error()})
		return
	}

	render.JSON(w, r, map[string]interface{}{
		"asn_stats": stats,
		"period_hours": hours,
	})
}
