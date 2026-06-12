-- ClickHouse Schema for Rigour Time-Series Analytics
-- Database: rigour_analytics

CREATE DATABASE IF NOT EXISTS rigour_analytics;

-- Scan Events Table (immutable append-only log)
-- Stores every scan event for historical analysis
CREATE TABLE IF NOT EXISTS rigour_analytics.scan_events (
    event_time DateTime64(3) CODEC(Delta, ZSTD),
    event_date Date DEFAULT toDate(event_time),
    
    -- Host identifiers
    ip IPv4 CODEC(ZSTD),
    ip_int UInt32 CODEC(Delta, ZSTD),
    asn UInt32 CODEC(Delta, ZSTD),
    country FixedString(2) CODEC(ZSTD),
    city LowCardinality(String) CODEC(ZSTD),
    
    -- Port and service
    port UInt16 CODEC(ZSTD),
    protocol LowCardinality(String) CODEC(ZSTD),
    service LowCardinality(String) CODEC(ZSTD),
    
    -- Scan metadata
    scanner_id LowCardinality(String) CODEC(ZSTD),
    scan_duration_ms UInt32 CODEC(Delta, ZSTD),
    
    -- Banner and fingerprint
    banner String CODEC(ZSTD),
    cpe String CODEC(ZSTD),
    product LowCardinality(String) CODEC(ZSTD),
    
    -- TLS/HTTP metadata
    tls_enabled UInt8,
    http_status_code UInt16,
    http_title String CODEC(ZSTD),
    
    -- Enrichment data
    rdns String CODEC(ZSTD),
    cves Array(String) CODEC(ZSTD),
    
    INDEX idx_ip ip TYPE bloom_filter GRANULARITY 4,
    INDEX idx_asn asn TYPE minmax GRANULARITY 4,
    INDEX idx_port port TYPE minmax GRANULARITY 4,
    INDEX idx_service service TYPE bloom_filter GRANULARITY 4
)
ENGINE = MergeTree()
PARTITION BY toYYYYMM(event_date)
ORDER BY (event_date, ip_int, port, event_time)
TTL event_date + INTERVAL 90 DAY
SETTINGS index_granularity = 8192;

-- Port Discovery Timeline (Materialized View)
-- Track when ports first/last appeared
CREATE TABLE IF NOT EXISTS rigour_analytics.port_timeline (
    date Date,
    ip IPv4,
    port UInt16,
    service LowCardinality(String),
    first_seen DateTime,
    last_seen DateTime,
    scan_count UInt32,
    
    INDEX idx_ip ip TYPE bloom_filter GRANULARITY 4,
    INDEX idx_port port TYPE minmax GRANULARITY 4
)
ENGINE = SummingMergeTree()
PARTITION BY toYYYYMM(date)
ORDER BY (date, ip, port)
TTL date + INTERVAL 180 DAY
SETTINGS index_granularity = 8192;

-- CVE Discovery Timeline
-- Track when vulnerabilities were first/last detected
CREATE TABLE IF NOT EXISTS rigour_analytics.cve_timeline (
    date Date,
    cve_id String,
    ip IPv4,
    port UInt16,
    cpe String,
    first_detected DateTime,
    last_detected DateTime,
    detection_count UInt32,
    
    INDEX idx_cve cve_id TYPE bloom_filter GRANULARITY 4,
    INDEX idx_ip ip TYPE bloom_filter GRANULARITY 4
)
ENGINE = SummingMergeTree()
PARTITION BY toYYYYMM(date)
ORDER BY (date, cve_id, ip, port)
TTL date + INTERVAL 365 DAY
SETTINGS index_granularity = 8192;

-- ASN Statistics (Aggregated)
-- Track scan volume per ASN over time
CREATE TABLE IF NOT EXISTS rigour_analytics.asn_stats (
    hour DateTime,
    asn UInt32,
    country FixedString(2),
    scan_count UInt64,
    unique_ips UInt32,
    open_ports UInt32,
    
    INDEX idx_asn asn TYPE minmax GRANULARITY 4
)
ENGINE = SummingMergeTree()
PARTITION BY toYYYYMM(hour)
ORDER BY (hour, asn)
TTL hour + INTERVAL 180 DAY
SETTINGS index_granularity = 8192;

-- Service Popularity Over Time
-- Track which services are most common
CREATE TABLE IF NOT EXISTS rigour_analytics.service_stats (
    date Date,
    service LowCardinality(String),
    port UInt16,
    host_count UInt64,
    total_scans UInt64,
    
    INDEX idx_service service TYPE bloom_filter GRANULARITY 4
)
ENGINE = SummingMergeTree()
PARTITION BY toYYYYMM(date)
ORDER BY (date, service, port)
TTL date + INTERVAL 365 DAY
SETTINGS index_granularity = 8192;

-- Common Queries:
-- 1. Hosts discovered in last 24h:
--    SELECT count(DISTINCT ip) FROM scan_events WHERE event_time >= now() - INTERVAL 1 DAY;
--
-- 2. Top 10 services by unique host count:
--    SELECT service, port, uniq(ip) as hosts FROM scan_events GROUP BY service, port ORDER BY hosts DESC LIMIT 10;
--
-- 3. CVE detections over time:
--    SELECT date, cve_id, sum(detection_count) FROM cve_timeline GROUP BY date, cve_id ORDER BY date;
--
-- 4. ASN scan volume trends:
--    SELECT toDate(hour) as day, asn, sum(scan_count) FROM asn_stats GROUP BY day, asn ORDER BY day;
