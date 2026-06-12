package clickhouse

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "github.com/ClickHouse/clickhouse-go/v2"
	"github.com/ctrlsam/rigour/pkg/types"
)

type Client struct {
	db *sql.DB
}

type Config struct {
	Host     string
	Port     int
	Database string
	Username string
	Password string
}

func NewClient(cfg Config) (*Client, error) {
	dsn := fmt.Sprintf("clickhouse://%s:%s@%s:%d/%s?dial_timeout=10s&max_execution_time=60",
		cfg.Username, cfg.Password, cfg.Host, cfg.Port, cfg.Database)

	db, err := sql.Open("clickhouse", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open ClickHouse connection: %w", err)
	}

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("failed to ping ClickHouse: %w", err)
	}

	return &Client{db: db}, nil
}

func (c *Client) Close() error {
	if c.db != nil {
		return c.db.Close()
	}
	return nil
}

// InsertScanEvent inserts a scan event into the time-series database
func (c *Client) InsertScanEvent(ctx context.Context, scan types.EnrichedScan) error {
	query := `
		INSERT INTO rigour_analytics.scan_events (
			event_time, ip, ip_int, asn, country, city, port, protocol, service,
			scanner_id, banner, cpe, product, tls_enabled, rdns, cves
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	ipInt := ipToInt(scan.IP)
	tlsEnabled := uint8(0)
	if scan.Protocol == "https" {
		tlsEnabled = 1
	}

	_, err := c.db.ExecContext(ctx, query,
		scan.ScannedAt,
		scan.IP,
		ipInt,
		scan.ASN,
		scan.Country,
		scan.City,
		scan.Port,
		scan.Protocol,
		scan.Service,
		scan.ScannerID,
		scan.Banner,
		scan.CPE,
		"", // product - TODO: extract from CPE
		tlsEnabled,
		scan.RDNS,
		scan.CVEs,
	)

	if err != nil {
		return fmt.Errorf("failed to insert scan event: %w", err)
	}

	return nil
}

// BatchInsertScanEvents inserts multiple scan events efficiently
func (c *Client) BatchInsertScanEvents(ctx context.Context, scans []types.EnrichedScan) error {
	if len(scans) == 0 {
		return nil
	}

	tx, err := c.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	stmt, err := tx.PrepareContext(ctx, `
		INSERT INTO rigour_analytics.scan_events (
			event_time, ip, ip_int, asn, country, city, port, protocol, service,
			scanner_id, banner, cpe, product, tls_enabled, rdns, cves
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`)
	if err != nil {
		return fmt.Errorf("failed to prepare statement: %w", err)
	}
	defer stmt.Close()

	for _, scan := range scans {
		ipInt := ipToInt(scan.IP)
		tlsEnabled := uint8(0)
		if scan.Protocol == "https" {
			tlsEnabled = 1
		}

		_, err = stmt.ExecContext(ctx,
			scan.ScannedAt,
			scan.IP,
			ipInt,
			scan.ASN,
			scan.Country,
			scan.City,
			scan.Port,
			scan.Protocol,
			scan.Service,
			scan.ScannerID,
			scan.Banner,
			scan.CPE,
			"",
			tlsEnabled,
			scan.RDNS,
			scan.CVEs,
		)

		if err != nil {
			return fmt.Errorf("failed to insert scan event: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// GetServiceStats returns service popularity statistics
func (c *Client) GetServiceStats(ctx context.Context, startDate, endDate time.Time) ([]ServiceStat, error) {
	query := `
		SELECT 
			service,
			port,
			uniq(ip) as host_count,
			count(*) as total_scans
		FROM rigour_analytics.scan_events
		WHERE event_date >= ? AND event_date <= ?
		GROUP BY service, port
		ORDER BY host_count DESC
		LIMIT 100
	`

	rows, err := c.db.QueryContext(ctx, query, startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("failed to query service stats: %w", err)
	}
	defer rows.Close()

	var stats []ServiceStat
	for rows.Next() {
		var stat ServiceStat
		if err := rows.Scan(&stat.Service, &stat.Port, &stat.HostCount, &stat.TotalScans); err != nil {
			return nil, err
		}
		stats = append(stats, stat)
	}

	return stats, nil
}

// GetCVETrends returns CVE detection trends over time
func (c *Client) GetCVETrends(ctx context.Context, days int) ([]CVETrend, error) {
	query := `
		SELECT 
			toDate(event_time) as date,
			arrayJoin(cves) as cve_id,
			count(DISTINCT ip) as affected_hosts
		FROM rigour_analytics.scan_events
		WHERE event_time >= now() - INTERVAL ? DAY
		  AND length(cves) > 0
		GROUP BY date, cve_id
		ORDER BY date DESC, affected_hosts DESC
		LIMIT 1000
	`

	rows, err := c.db.QueryContext(ctx, query, days)
	if err != nil {
		return nil, fmt.Errorf("failed to query CVE trends: %w", err)
	}
	defer rows.Close()

	var trends []CVETrend
	for rows.Next() {
		var trend CVETrend
		if err := rows.Scan(&trend.Date, &trend.CVEID, &trend.AffectedHosts); err != nil {
			return nil, err
		}
		trends = append(trends, trend)
	}

	return trends, nil
}

// GetASNStats returns ASN scan volume statistics
func (c *Client) GetASNStats(ctx context.Context, hours int) ([]ASNStat, error) {
	query := `
		SELECT 
			asn,
			country,
			count(*) as scan_count,
			uniq(ip) as unique_ips
		FROM rigour_analytics.scan_events
		WHERE event_time >= now() - INTERVAL ? HOUR
		GROUP BY asn, country
		ORDER BY scan_count DESC
		LIMIT 100
	`

	rows, err := c.db.QueryContext(ctx, query, hours)
	if err != nil {
		return nil, fmt.Errorf("failed to query ASN stats: %w", err)
	}
	defer rows.Close()

	var stats []ASNStat
	for rows.Next() {
		var stat ASNStat
		if err := rows.Scan(&stat.ASN, &stat.Country, &stat.ScanCount, &stat.UniqueIPs); err != nil {
			return nil, err
		}
		stats = append(stats, stat)
	}

	return stats, nil
}

// Helper types
type ServiceStat struct {
	Service    string
	Port       uint16
	HostCount  uint64
	TotalScans uint64
}

type CVETrend struct {
	Date          time.Time
	CVEID         string
	AffectedHosts uint64
}

type ASNStat struct {
	ASN       uint32
	Country   string
	ScanCount uint64
	UniqueIPs uint32
}

// ipToInt converts IPv4 to uint32
func ipToInt(ipStr string) uint32 {
	// Simple conversion - production should use net.ParseIP
	var a, b, c, d uint32
	fmt.Sscanf(ipStr, "%d.%d.%d.%d", &a, &b, &c, &d)
	return (a << 24) | (b << 16) | (c << 8) | d
}
