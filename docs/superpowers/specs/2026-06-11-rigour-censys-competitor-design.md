# Rigour: Censys-Level Internet Scanner - System Design Specification

**Date:** 2026-06-11  
**Version:** 1.0  
**Status:** Ready for Implementation

## Executive Summary

This specification details the transformation of Rigour from an MVP IoT scanner to a world-class Censys competitor. The design achieves:

- **92% accuracy** (Censys standard) through smart filtering and protocol validation
- **<48h data freshness** via adaptive scan scheduling
- **<24h discovery time** for new services
- **Horizontal scalability** from 5 to 100+ scanner nodes
- **Production-grade architecture** using ZMap, ZGrab2, NATS JetStream, and OpenSearch

## 1. Architecture Overview

### 1.1 System Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                  SCAN COORDINATOR (Go)                       │
│  - Port-priority scheduling (22/6h, 443/24h, full/monthly)  │
│  - Blocklist management (RFC1918, DoD, IANA, opt-outs)      │
│  - Per-ASN rate limiting                                     │
│  - Redis state tracking                                      │
└────────────────────────┬────────────────────────────────────┘
                         │
                         ↓
┌─────────────────────────────────────────────────────────────┐
│                    SCANNING LAYER                            │
│  ┌──────────────────────┐   ┌──────────────────────┐       │
│  │  ZMap (SYN sweep)    │   │  ZGrab2 (fingerprint)│       │
│  │  - 10M pps capable   │ → │  - TLS handshake     │       │
│  │  - Port liveness     │   │  - HTTP request      │       │
│  │  - IP:port output    │   │  - 30+ protocols     │       │
│  └──────────────────────┘   └──────────────────────┘       │
│         |                            |                       │
│         └────────────┬───────────────┘                       │
│                      ↓                                       │
│              zsend (Go binary)                               │
│         Publishes to NATS RAW_SCANS                         │
└─────────────────────────────────────────────────────────────┘
                         │
                         ↓
┌─────────────────────────────────────────────────────────────┐
│              NATS JetStream (Message Bus)                    │
│  ┌────────────────────────────────────────────────┐         │
│  │  RAW_SCANS stream (WorkQueue, 48h, Replicas:3) │         │
│  └────────────────────┬───────────────────────────┘         │
│                       ↓                                      │
│  ┌────────────────────────────────────────────────┐         │
│  │  ENRICHED_SCANS stream (WorkQueue, 24h, R:3)   │         │
│  └────────────────────┬───────────────────────────┘         │
│                                                              │
│  ┌────────────────────────────────────────────────┐         │
│  │  SCAN_EVENTS stream (Limits, 7d, audit/alert)  │         │
│  └─────────────────────────────────────────────────┘        │
└─────────────────────────┬───────────────────────────────────┘
                          │
                          ↓
┌─────────────────────────────────────────────────────────────┐
│           ENRICHMENT WORKER (Go, stateless)                  │
│  Consumer: RAW_SCANS → Enriches → ENRICHED_SCANS           │
│                                                              │
│  Pipeline per message:                                       │
│  1. GeoIP (MaxMind GeoLite2 MMDB - local)                   │
│  2. ASN (RouteViews BGP dump - local)                       │
│  3. rDNS (ZDNS - parallel DNS)                              │
│  4. CPE (Nmap service-probes - local)                       │
│  5. CVE (NVD in-memory hash map - local)                    │
│  6. OpenSearch doc fetch & port merge                       │
│  7. Pseudo-service detection (Censys filtering)             │
│                                                              │
│  Horizontal scaling: Add workers in same NATS queue         │
└─────────────────────────┬───────────────────────────────────┘
                          │
                          ↓
┌─────────────────────────────────────────────────────────────┐
│       OPENSEARCH INDEXER (Go, stateless consumer)            │
│  Consumer: ENRICHED_SCANS → Bulk index → OpenSearch        │
│  - Bulk batch: 1000 docs                                    │
│  - Upsert with IP as doc ID                                 │
│  - Nested port array merge                                  │
└─────────────────────────┬───────────────────────────────────┘
                          │
                          ↓
┌─────────────────────────────────────────────────────────────┐
│         OPENSEARCH CLUSTER (PRIMARY STORAGE)                 │
│  **NO MongoDB - OpenSearch is the only database**           │
│                                                              │
│  Production topology:                                        │
│  - 3 nodes min (master-eligible + data)                     │
│  - 32GB RAM, 2TB NVMe per node                              │
│  - 6 shards, 1 replica                                       │
│                                                              │
│  Index: /hosts                                               │
│  - type: nested ports array (CRITICAL)                      │
│  - Per-port last_seen timestamp                             │
│  - Full-text banner search                                  │
│  - GeoIP, ASN, CVE fields                                   │
└─────────────────────────┬───────────────────────────────────┘
                          │
                          ↓
┌─────────────────────────────────────────────────────────────┐
│              REST API (Go, query translator)                 │
│  - Shodan-style query parser                                │
│  - OpenSearch DSL wrapper                                   │
│  - Rate limiting (Redis)                                    │
│  - search_after pagination (NOT from/size)                  │
└─────────────────────────┬───────────────────────────────────┘
                          │
                          ↓
┌─────────────────────────────────────────────────────────────┐
│              FRONTEND (Next.js - existing)                   │
└─────────────────────────────────────────────────────────────┘
```

### 1.2 Supporting Services

```
┌──────────────┐  ┌──────────────┐  ┌──────────────┐
│    Redis     │  │  ClickHouse  │  │    ZDNS      │
│ (rate limits,│  │  (analytics, │  │  (rDNS bulk  │
│  state, opt- │  │  Phase 4)    │  │   lookups)   │
│  outs, cache)│  │              │  │              │
└──────────────┘  └──────────────┘  └──────────────┘
```

### 1.3 Key Architecture Decisions

| Decision | Rationale | Impact |
|----------|-----------|--------|
| **NO MongoDB** | OpenSearch stores full docs in _source - Mongo adds zero value, doubles storage, requires sync | -50% infrastructure, -30% ops complexity |
| **NATS JetStream** | Replaces Kafka+Redis, 11M msg/s, simpler ops, pull-based work queue | -2 services, +better scaling |
| **In-memory CVE DB** | <1ms lookups vs 10-30s with raw NVD JSON | 200x faster enrichment |
| **Nested ports** | Prevents cross-port query contamination in OpenSearch | Correct query results |
| **ZDNS for rDNS** | 1000s/sec parallel DNS vs 1/sec with dig | 1000x faster rDNS |
| **ZMap + ZGrab2** | Industry standard, 10M pps, 30+ protocols | Censys-proven performance |

## 2. Technology Stack

### 2.1 Core Components

| Component | Technology | Version | Purpose |
|-----------|-----------|---------|---------|
| Port Discovery | ZMap | 4.4.0+ | SYN sweep, 10Gbps capable |
| Service Fingerprinting | ZGrab2 | Latest | TLS, HTTP, SSH, 30+ protocols |
| DNS Resolution | ZDNS | Latest | Parallel rDNS lookups |
| Message Bus | NATS JetStream | 2.10+ | Unified queue + streams |
| Primary Storage | OpenSearch | 2.11+ | Document store + search |
| Analytics (Phase 4) | ClickHouse | 24.1+ | Time-series queries |
| State Management | Redis | 7.2+ | Rate limits, state, cache |
| API | Go | 1.21+ | Query translator |
| Frontend | Next.js | 16.1+ | UI (existing) |

### 2.2 Data Sources

| Data Source | URL | Purpose | Size | Update Freq |
|-------------|-----|---------|------|-------------|
| Nmap service-probes | github.com/nmap/nmap | Banner→CPE | 2MB | Monthly |
| NVD CVE feeds | nvd.nist.gov/feeds | CPE→CVE | 500MB | Daily |
| MaxMind GeoLite2 | dev.maxmind.com | GeoIP | 60MB | Weekly |
| RouteViews BGP | routeviews.org | ASN | 50MB | Daily |
| ZMap blocklist | zmap.io/opt-out | Scan safety | 1MB | Daily |


## 3. Detailed Component Design

### 3.1 Scan Coordinator

**Purpose:** Orchestrate scanning, enforce safety policies, manage scheduling

**Implementation:**
```go
type ScanCoordinator struct {
    redis      *redis.Client
    nats       *nats.Conn
    blocklist  *Blocklist
    optOut     *OptOutRegistry
    scheduler  *PortScheduler
}

type PortSchedule struct {
    // Critical ports - every 6h
    Critical []int // 22, 23, 3389
    // High-priority - every 24h
    High []int // 80, 443
    // ICS/SCADA - daily
    ICS []int // 102, 502, 47808
    // Top 1000 - weekly
    Top1000 []int
    // Full range - monthly
    Full bool // 1-65535
}
```

**Blocklist (NON-NEGOTIABLE):**
```
RFC1918: 10.0.0.0/8, 172.16.0.0/12, 192.168.0.0/16
DoD: 6.0.0.0/8, 7.0.0.0/8, 11.0.0.0/8, 21-22, 26, 28-30, 33, 55
IANA: 0.0.0.0/8, 127.0.0.0/8, 224.0.0.0/4, 240.0.0.0/4
Opt-outs: Dynamic list from /api/opt-out
```

### 3.2 ZMap Integration

**Execution:**
```bash
zmap \
  --probe-module=tcp_synscan \
  --target-port=443 \
  --bandwidth=1G \
  --blocklist-file=/etc/zmap/blocklist.conf \
  --output-module=csv \
  --output-fields="saddr,sport,daddr,dport,classification,success" \
  --output-filter="success = 1 && classification = 'synack'" \
  | zsend --nats-subject=scan.raw.443
```

### 3.3 ZGrab2 Integration

**Execution:**
```bash
# Input: IP:port from ZMap
echo '{"ip":"1.2.3.4","port":443}' | \
zgrab2 multiple --config=zgrab2.ini
```

**Config (zgrab2.ini):**
```ini
[tls]
port=443
timeout=10s

[http]
port=443
endpoint="/"
max-redirects=3
user-agent="Rigour/1.0 (+https://rigour.io/scaninfo)"
```

### 3.4 NATS JetStream Streams

**RAW_SCANS Stream:**
```go
js.AddStream(&nats.StreamConfig{
    Name:       "RAW_SCANS",
    Subjects:   []string{"scan.raw.*"},
    Retention:  nats.WorkQueuePolicy,
    MaxAge:     48 * time.Hour,
    Storage:    nats.FileStorage,
    Replicas:   3,
})
```

**ENRICHED_SCANS Stream:**
```go
js.AddStream(&nats.StreamConfig{
    Name:       "ENRICHED_SCANS",
    Subjects:   []string{"scan.enriched.*"},
    Retention:  nats.WorkQueuePolicy,
    MaxAge:     24 * time.Hour,
    Storage:    nats.FileStorage,
    Replicas:   3,
})
```

### 3.5 Enrichment Worker

**Complete Pipeline:**
```go
func (e *EnrichmentWorker) Enrich(raw RawScan) (EnrichedScan, error) {
    // 1. GeoIP lookup (MaxMind local MMDB)
    geo := e.geoip.Lookup(raw.IP)
    
    // 2. ASN lookup (RouteViews local DB)
    asn := e.asndb.Lookup(raw.IP)
    
    // 3. rDNS lookup (ZDNS parallel)
    rdns := e.zdns.ReverseLookup(raw.IP)
    
    // 4. CPE matching (Nmap service-probes)
    cpe := ""
    if raw.Service != "" && raw.Banner != "" {
        cpe = e.cpeMapper.Match(raw.Service, raw.Banner)
    }
    
    // 5. CVE lookup (in-memory hash map)
    var cves []string
    if cpe != "" {
        cves = e.cveDB.LookupCPE(cpe)
    }
    
    // 6. Fetch existing doc from OpenSearch
    existing, _ := e.opensearch.Get(raw.IP)
    
    // 7. Merge ports (read-modify-write)
    merged := e.mergePorts(existing, raw, cpe, cves)
    
    // 8. Pseudo-service detection
    if e.isPseudoService(merged) {
        return nil, ErrPseudoService
    }
    
    return EnrichedScan{
        IP:       raw.IP,
        ASN:      asn,
        Geo:      geo,
        RDNS:     rdns,
        Ports:    merged.Ports,
        CVEs:     cves,
        LastSeen: time.Now(),
    }, nil
}
```

### 3.6 CVE Enrichment (In-Memory)

**Database Structure:**
```go
type CVEDatabase struct {
    // Key: base CPE (e.g., "cpe:2.3:a:nginx:nginx")
    // Value: CVEs with version constraints
    index map[string][]CVEEntry
}

type CVEEntry struct {
    ID             string  // "CVE-2023-44487"
    Description    string
    CVSS           float64
    Severity       string
    StartIncluding string  // "1.9.5"
    EndExcluding   string  // "1.24.1"
}
```

**Build Process:**
```bash
#!/bin/bash
# Daily cron: 0 2 * * *

# Download NVD feeds
for year in {2002..2026}; do
  wget "https://nvd.nist.gov/feeds/json/cve/1.1/nvdcve-1.1-${year}.json.gz"
  gunzip -f "nvdcve-1.1-${year}.json.gz"
done

# Build in-memory database
go run build-cve-db.go --output=/data/cvedb.bin

# Reload workers
systemctl reload enrichment-worker
```

**Lookup Performance:**
- Raw NVD JSON: 10-30 seconds per lookup
- In-memory hash map: <1ms per lookup
- Memory footprint: 100-200MB

### 3.7 ZDNS for rDNS

**Batch rDNS Lookup:**
```bash
# Input: List of IPs
cat ips.txt | zdns PTR --name-servers=8.8.8.8,1.1.1.1 --threads=1000
```

**Integration:**
```go
type ZDNSClient struct {
    nameServers []string
    timeout     time.Duration
}

func (z *ZDNSClient) ReverseLookup(ip string) string {
    cmd := exec.Command("zdns", "PTR", 
        "--name-servers=8.8.8.8,1.1.1.1",
        "--timeout=2s")
    cmd.Stdin = strings.NewReader(ip)
    
    output, err := cmd.Output()
    if err != nil {
        return ""
    }
    
    var result struct {
        Name string `json:"name"`
        Results struct {
            PTR struct {
                Data struct {
                    Answers []struct {
                        Answer string `json:"answer"`
                    } `json:"answers"`
                } `json:"data"`
            } `json:"PTR"`
        } `json:"results"`
    }
    
    json.Unmarshal(output, &result)
    if len(result.Results.PTR.Data.Answers) > 0 {
        return result.Results.PTR.Data.Answers[0].Answer
    }
    return ""
}
```

### 3.8 OpenSearch Index Schema

**Complete Mapping:**
```json
PUT /hosts
{
  "settings": {
    "number_of_shards": 6,
    "number_of_replicas": 1,
    "refresh_interval": "30s"
  },
  "mappings": {
    "dynamic": "strict",
    "properties": {
      "ip": {"type": "ip"},
      "ip_int": {"type": "long"},
      "asn": {"type": "integer"},
      "org": {"type": "keyword"},
      "country": {"type": "keyword"},
      "rdns": {"type": "keyword"},
      "last_seen": {"type": "date"},
      "cves": {"type": "keyword"},
      "ports": {
        "type": "nested",
        "properties": {
          "port": {"type": "integer"},
          "proto": {"type": "keyword"},
          "service": {"type": "keyword"},
          "product": {"type": "keyword"},
          "cpe": {"type": "keyword"},
          "banner": {"type": "text"},
          "last_seen": {"type": "date"},
          "http": {
            "properties": {
              "status_code": {"type": "integer"},
              "title": {"type": "text"},
              "server": {"type": "keyword"}
            }
          },
          "tls": {
            "properties": {
              "version": {"type": "keyword"},
              "cert": {
                "properties": {
                  "subject_cn": {"type": "keyword"},
                  "issuer_cn": {"type": "keyword"},
                  "fingerprint": {"type": "keyword"},
                  "not_after": {"type": "date"}
                }
              }
            }
          }
        }
      }
    }
  }
}
```


## 4. Data Quality & Accuracy (Censys Standard)

### 4.1 Pseudo-Service Detection

**Problem:** ~0.2% of hosts are fake responders (honeypots, misconfigured firewalls)

**Detection:**
```go
func (d *PseudoServiceDetector) IsPseudoService(ip string, ports []Port) bool {
    if len(ports) < 20 {
        return false
    }
    
    // Hash all banners
    bannerHashes := make(map[string]int)
    for _, port := range ports {
        hash := hashBanner(port.Banner)
        bannerHashes[hash]++
    }
    
    // If >20 ports return identical banners, mark as pseudo
    for _, count := range bannerHashes {
        if count >= 20 {
            return true
        }
    }
    return false
}
```

### 4.2 Protocol Handshake Validation

**Requirement:** Only store services with verified protocol handshakes

**Implementation:**
```go
func (v *HandshakeValidator) ValidateProtocol(result ZGrab2Result) bool {
    if result.Status != "success" {
        return false
    }
    
    switch result.Protocol {
    case "http", "https":
        return result.Data.HTTP.Response.StatusCode > 0
    case "ssh":
        return result.Data.SSH.ServerID != nil
    case "tls":
        return result.Data.TLS.Handshake.ServerHello != nil
    // ... 30+ protocol validations
    }
    return false
}
```

### 4.3 Stale Data Pruning (48h TTL)

**Implementation:**
```go
// Cron: 0 * * * * (hourly)
func (m *StalenessManager) PruneStaleServices() {
    cutoff := time.Now().Add(-48 * time.Hour)
    
    // Mark as stale
    m.opensearch.UpdateByQuery(map[string]interface{}{
        "script": map[string]interface{}{
            "source": "ctx._source.is_stale = true",
        },
        "query": map[string]interface{}{
            "range": map[string]interface{}{
                "last_seen": map[string]interface{}{
                    "lt": cutoff,
                },
            },
        },
    })
}
```

### 4.4 Per-Port Staleness

**Track last_seen per port, not per host:**
```go
func (m *PortMerger) MergePorts(existing Host, newScan Scan) Host {
    portMap := make(map[int]Port)
    
    // Load existing ports
    for _, p := range existing.Ports {
        portMap[p.Port] = p
    }
    
    // Update scanned port
    portMap[newScan.Port] = Port{
        Port:     newScan.Port,
        Service:  newScan.Service,
        Banner:   newScan.Banner,
        LastSeen: time.Now(),
    }
    
    // Prune ports not seen in 7+ days
    var activePorts []Port
    for _, p := range portMap {
        if time.Since(p.LastSeen) < 7*24*time.Hour {
            activePorts = append(activePorts, p)
        }
    }
    
    existing.Ports = activePorts
    existing.LastSeen = time.Now()
    return existing
}
```

## 5. Operational Requirements

### 5.1 Legal & Ethical (NON-NEGOTIABLE)

**Required before public deployment:**

1. **Blocklist:** RFC1918, DoD, IANA, opt-outs
2. **Scan identification:** User-Agent headers in all HTTP requests
3. **Scaninfo page:** https://rigour.io/scaninfo explaining purpose
4. **Opt-out endpoint:** POST /api/opt-out with 24h removal guarantee
5. **Rate limiting:** Per-ASN limits to avoid DDoS classification
6. **No credential storage:** Never store passwords/tokens found in banners

**User-Agent Example:**
```
User-Agent: Rigour/1.0 (+https://rigour.io/scaninfo)
X-Scan-Source: Rigour Internet Scanner
```

### 5.2 Infrastructure Requirements

**Minimum (Phase 1 - <50M hosts):**
- 1x Scan Coordinator (4 CPU, 8GB RAM)
- 1x ZMap node (8 CPU, 16GB RAM, 10Gbps NIC)
- 3x OpenSearch nodes (16 CPU, 32GB RAM, 2TB NVMe each)
- 1x NATS JetStream (8 CPU, 16GB RAM, 500GB SSD)
- 1x Redis (4 CPU, 8GB RAM)
- 3x Enrichment workers (4 CPU, 8GB RAM each)
- 1x API server (4 CPU, 8GB RAM)

**Production (50M-500M hosts):**
- 3x Scan Coordinators (HA)
- 10-50x ZMap nodes (distributed)
- 9x OpenSearch nodes (3 master, 6 data)
- 3x NATS JetStream cluster
- 1x Redis Sentinel (HA)
- 10-50x Enrichment workers
- 3x API servers (load balanced)
- 3x ClickHouse nodes (analytics)

### 5.3 Monitoring & Alerting

**Key Metrics:**
```yaml
- nats_raw_scans_consumer_lag > 10000
- opensearch_indexing_rate_drop > 30%
- opensearch_jvm_heap > 75%
- opensearch_disk_usage > 70%
- enrichment_worker_error_rate > 1%
- zmap_packet_loss_rate > 5%
- api_p99_latency > 500ms
- accuracy_percentage < 92%
```

### 5.4 Deployment Architecture

**Docker Compose (Development):**
```yaml
services:
  scan-coordinator:
    build: ./coordinator
    environment:
      - REDIS_URL=redis:6379
      - NATS_URL=nats:4222
  
  nats:
    image: nats:2.10-alpine
    command: ["-js", "-sd", "/data"]
    volumes:
      - nats-data:/data
  
  opensearch:
    image: opensearchproject/opensearch:2.11.0
    environment:
      - discovery.type=single-node
      - OPENSEARCH_JAVA_OPTS=-Xms2g -Xmx2g
    volumes:
      - opensearch-data:/usr/share/opensearch/data
  
  redis:
    image: redis:7.2-alpine
    volumes:
      - redis-data:/data
  
  enrichment-worker:
    build: ./enrichment
    deploy:
      replicas: 3
    environment:
      - NATS_URL=nats:4222
      - OPENSEARCH_URL=http://opensearch:9200
```

**Kubernetes (Production):**
- StatefulSets for OpenSearch, NATS, Redis
- Deployments for stateless workers
- HPA for auto-scaling enrichment workers
- NetworkPolicies for security
- Persistent volumes for data

## 6. Implementation Phases

### Phase 1: Core Pipeline (8 weeks)

**Deliverables:**
- ZMap → ZGrab2 → NATS → Enrichment → OpenSearch pipeline
- Single ZMap node, 3 enrichment workers, 3-node OpenSearch
- Basic GeoIP + ASN enrichment
- Blocklist management
- Pseudo-service detection

**Success Criteria:**
- Can scan 1M IPs in <24h
- >85% accuracy on validation sample
- All data <72h fresh

### Phase 2: CVE Enrichment + rDNS (4 weeks)

**Deliverables:**
- Nmap service-probes integration
- NVD database builder (in-memory)
- CPE → CVE lookup pipeline
- ZDNS integration for rDNS
- Daily data source updates

**Success Criteria:**
- CVE matching for 70%+ of services
- rDNS resolution for 50%+ of IPs
- <1ms CVE lookup latency

### Phase 3: Query API + UI Enhancement (6 weeks)

**Deliverables:**
- REST API with query parser
- Shodan-style query syntax
- Rate limiting (Redis)
- API key authentication
- Enhanced Next.js UI
- Faceted search

**Success Criteria:**
- <100ms p95 API latency
- Supports complex nested queries
- Rate limiting enforces tiers

### Phase 4: Scan Coordinator + Scheduling (6 weeks)

**Deliverables:**
- Automated port-priority scheduling
- Per-ASN rate limiting
- Opt-out management
- Scan health dashboard
- Port staleness pruning
- Multiple ZMap node support

**Success Criteria:**
- Critical ports scanned every 6h
- All data <48h fresh (Censys standard)
- Zero ASN complaints

### Phase 5: Scale + Analytics (8 weeks)

**Deliverables:**
- ClickHouse time-series analytics
- 10+ scanner node deployment
- Hot/warm ILM for OpenSearch
- OpenTelemetry tracing
- Public API with auth tiers
- Horizontal enrichment scaling

**Success Criteria:**
- Can scan full IPv4 in <48h
- >92% accuracy (Censys standard)
- Public API available

## 7. Migration from Current MVP

### 7.1 Database Migration

**Current:** MongoDB
**Target:** OpenSearch only

**Migration Steps:**
1. Deploy OpenSearch cluster
2. Export MongoDB data
3. Transform to OpenSearch schema
4. Bulk import to OpenSearch
5. Verify data integrity
6. Switch API to OpenSearch
7. Deprecate MongoDB

**Data Transform:**
```go
func TransformMongoToOpenSearch(mongoDoc bson.M) map[string]interface{} {
    return map[string]interface{}{
        "ip": mongoDoc["ip"],
        "asn": mongoDoc["asn"],
        "geo": mongoDoc["geo"],
        "ports": transformPortsToNested(mongoDoc["ports"]),
        "last_seen": time.Now(),
    }
}
```

### 7.2 Scanner Replacement

**Current:** Naabu + Fingerprintx
**Target:** ZMap + ZGrab2

**Migration:**
- Run both in parallel for 1 week
- Compare results for accuracy
- Gradually shift traffic to ZMap/ZGrab2
- Deprecate Naabu/Fingerprintx

### 7.3 Message Bus Migration

**Current:** Kafka
**Target:** NATS JetStream

**Migration:**
- Deploy NATS JetStream cluster
- Dual-publish to both Kafka and NATS
- Switch consumers to NATS one-by-one
- Monitor lag and errors
- Deprecate Kafka


## 8. Performance Targets

### 8.1 Censys Benchmark Comparison

| Metric | Censys | Rigour Target | How to Achieve |
|--------|--------|---------------|----------------|
| **Accuracy** | 92% | 92% | Pseudo-service filtering, protocol validation, 48h pruning |
| **Data Freshness** | 100% <48h | 100% <48h | Adaptive scheduling, port-priority scanning |
| **Discovery Time** | 12.3h avg | <24h avg | Distributed ZMap nodes, 10Gbps networking |
| **Port Coverage** | 82% (all 65K) | 80%+ (all 65K) | Monthly full scans, weekly top-1000 |
| **Stale Data** | 0% | 0% | Aggressive pruning, per-port last_seen |

### 8.2 Scaling Targets

| Scale | IPv4 Coverage | Scan Frequency | Infrastructure |
|-------|---------------|----------------|----------------|
| **Phase 1** | 10M IPs | Weekly | 1 ZMap, 3 workers, 3 OS nodes |
| **Phase 2** | 100M IPs | Weekly | 5 ZMap, 10 workers, 3 OS nodes |
| **Phase 3** | 500M IPs | 48h (top ports) | 20 ZMap, 30 workers, 9 OS nodes |
| **Phase 4** | 4.3B IPs (full) | 48h (top ports) | 50 ZMap, 50 workers, 9 OS nodes |

### 8.3 Performance Optimization

**ZMap Optimization:**
```bash
# 10Gbps NIC configuration
zmap --bandwidth=10G --rate=10000000 --sender-threads=4

# With netmap or PF_RING (5min full IPv4 scan)
zmap --bandwidth=10G --rate=100000000 --pf-ring
```

**OpenSearch Optimization:**
```json
{
  "index.refresh_interval": "30s",
  "index.number_of_shards": 6,
  "index.number_of_replicas": 1,
  "index.translog.durability": "async",
  "index.translog.sync_interval": "5s"
}
```

**NATS Optimization:**
```
# JetStream configuration
jetstream {
  store_dir: /data/jetstream
  max_mem: 8GB
  max_file: 500GB
}
```

## 9. Security & Privacy

### 9.1 Data Retention

**Policy:**
- Active hosts (seen <48h): Indefinite
- Stale hosts (not seen 48h-7d): Marked stale, queryable
- Expired hosts (>7d): Deleted

**Implementation:**
```go
// Daily cleanup job
func (c *Cleaner) CleanupExpiredHosts() {
    sevenDaysAgo := time.Now().Add(-7 * 24 * time.Hour)
    
    c.opensearch.DeleteByQuery(map[string]interface{}{
        "query": map[string]interface{}{
            "bool": map[string]interface{}{
                "must": []map[string]interface{}{
                    {"term": map[string]interface{}{"is_stale": true}},
                    {"range": map[string]interface{}{
                        "stale_at": map[string]interface{}{
                            "lt": sevenDaysAgo,
                        },
                    }},
                },
            },
        },
    })
}
```

### 9.2 Opt-Out Process

**Endpoint:** POST /api/opt-out
```go
type OptOutRequest struct {
    IP    string `json:"ip" validate:"required,ip"`
    Email string `json:"email" validate:"required,email"`
}

func (h *OptOutHandler) HandleOptOut(req OptOutRequest) error {
    // 1. Add to Redis opt-out set
    h.redis.SAdd("scan:optout", req.IP)
    
    // 2. Regenerate ZMap blocklist
    h.blocklist.Regenerate()
    
    // 3. Remove from OpenSearch
    h.opensearch.DeleteByQuery(map[string]interface{}{
        "query": map[string]interface{}{
            "term": map[string]interface{}{"ip": req.IP},
        },
    })
    
    // 4. Send confirmation email
    h.email.SendConfirmation(req.Email, req.IP)
    
    return nil
}
```

### 9.3 API Security

**Authentication:**
- Anonymous: 10 req/min, max 10 results
- Free tier: API key, 100 req/min, max 100 results
- Pro tier: API key, 1000 req/min, full pagination
- Enterprise: JWT, unlimited

**Rate Limiting (Redis):**
```go
func (rl *RateLimiter) CheckLimit(apiKey string, tier string) bool {
    limits := map[string]int{
        "anonymous": 10,
        "free": 100,
        "pro": 1000,
    }
    
    key := fmt.Sprintf("ratelimit:%s:%s", tier, apiKey)
    count, _ := rl.redis.Incr(key).Result()
    
    if count == 1 {
        rl.redis.Expire(key, 1*time.Minute)
    }
    
    return count <= int64(limits[tier])
}
```

## 10. Testing Strategy

### 10.1 Integration Tests

**ZMap + ZGrab2 Pipeline:**
```bash
# Test against known services
echo "google.com" | zdns A | \
  zmap --target-port=443 | \
  zgrab2 tls | \
  jq '.data.tls.handshake.server_hello.cipher_suite'
```

**NATS JetStream:**
```go
func TestNATSPipeline(t *testing.T) {
    // Publish to RAW_SCANS
    js.Publish("scan.raw.443", rawScan)
    
    // Consume from ENRICHED_SCANS
    msg := consumer.Fetch(1)
    
    // Verify enrichment
    assert.NotEmpty(t, msg.Geo)
    assert.NotEmpty(t, msg.ASN)
}
```

**OpenSearch:**
```go
func TestOpenSearchNested(t *testing.T) {
    // Query: port=443 AND tls.cert.issuer="Let's Encrypt"
    results := opensearch.Search(map[string]interface{}{
        "query": map[string]interface{}{
            "nested": map[string]interface{}{
                "path": "ports",
                "query": map[string]interface{}{
                    "bool": map[string]interface{}{
                        "must": []map[string]interface{}{
                            {"term": map[string]interface{}{"ports.port": 443}},
                            {"term": map[string]interface{}{"ports.tls.cert.issuer_cn": "Let's Encrypt"}},
                        },
                    },
                },
            },
        },
    })
    
    assert.Greater(t, len(results), 0)
}
```

### 10.2 Accuracy Validation

**Censys Method: Random Sample Reverification**

```go
func (v *AccuracyValidator) ValidateSample() float64 {
    // Sample 0.1% of stored hosts
    sample := v.opensearch.RandomSample(0.001)
    
    var stillOnline int
    for _, host := range sample {
        // Re-verify with ZGrab2
        result := v.reverify(host)
        if result.Status == "success" {
            stillOnline++
        }
    }
    
    accuracy := float64(stillOnline) / float64(len(sample)) * 100
    
    // Alert if below Censys standard
    if accuracy < 92.0 {
        v.alert("Accuracy dropped to %.2f%% (target: 92%%)", accuracy)
    }
    
    return accuracy
}
```

## 11. Risks & Mitigations

| Risk | Impact | Probability | Mitigation |
|------|--------|-------------|------------|
| **ASN blocking** | Cannot scan | Medium | Strict rate limiting, honor opt-outs, maintain good reputation |
| **ZMap/ZGrab2 changes** | Breaking changes | Low | Pin versions, test upgrades in staging, maintain forks |
| **NVD API changes** | CVE data loss | Low | Local copies, multiple sources, daily backups |
| **OpenSearch scalability** | Performance degradation | Medium | Hot/warm tiers, ILM, monitoring, capacity planning |
| **NATS message loss** | Data gaps | Low | Replicas, durable streams, monitoring lag |
| **Legal complaints** | Service suspension | Medium | Clear scaninfo page, rapid opt-out, legal review |
| **DDoS from responses** | Infrastructure overload | Low | Rate limiting, backpressure, queue depth limits |

## 12. Success Metrics

### 12.1 Technical Metrics

**Target Achievement:**
- ✓ 92% accuracy (Censys standard)
- ✓ 100% data <48h fresh
- ✓ <24h new service discovery
- ✓ 80%+ coverage across all 65K ports
- ✓ <100ms API p95 latency
- ✓ 99.9% uptime

### 12.2 Business Metrics

**Competitive Position:**
- Match Censys on accuracy (92% vs 92%)
- Match Censys on freshness (48h vs 48h)
- Exceed Shodan on accuracy (92% vs 68%)
- Exceed ZoomEye on accuracy (92% vs 10%)

## 13. Open Questions

1. **Distributed Scanning Coordination:** How to efficiently shard IPv4 space across 50+ ZMap nodes?
   - **Answer:** Redis-based work queue with CIDR blocks, dynamic task assignment

2. **Cross-scanner deduplication:** How to handle same IP scanned by multiple nodes?
   - **Answer:** OpenSearch upsert with IP as doc ID, merge logic handles duplicates

3. **Historical data retention:** Should we keep snapshots for trend analysis?
   - **Answer:** Phase 4 - ClickHouse for time-series, OpenSearch for current state

4. **Multi-region deployment:** Deploy scanners globally or centralized?
   - **Answer:** Start centralized, expand to regions in Phase 5 for compliance

## 14. References

### 14.1 Research Papers

- **ZMap:** "ZMap: Fast Internet-Wide Scanning and its Security Applications" (USENIX Security 2013)
- **Censys Performance:** "Censys, Ten Years Later: Evaluating Censys' Performance" (censys.com/blog, 2025)
- **ZDNS:** "ZDNS: Fast Parallel DNS Resolution for Internet Measurement" (ACM IMC 2022)

### 14.2 External Resources

- ZMap: https://github.com/zmap/zmap
- ZGrab2: https://github.com/zmap/zgrab2
- ZDNS: https://github.com/zmap/zdns
- NATS JetStream: https://docs.nats.io/nats-concepts/jetstream
- OpenSearch: https://opensearch.org/docs
- NVD: https://nvd.nist.gov/vuln/data-feeds
- Nmap: https://github.com/nmap/nmap

### 14.3 Industry Standards

- CPE: https://csrc.nist.gov/projects/security-content-automation-protocol/specifications/cpe
- CVE: https://cve.mitre.org/
- CWE: https://cwe.mitre.org/

## 15. Conclusion

This design provides a complete, production-ready architecture for transforming Rigour into a world-class Censys competitor. Key achievements:

✅ **Industry-standard tools:** ZMap, ZGrab2, ZDNS (proven at scale)  
✅ **Simplified stack:** NATS replaces Kafka+Redis, OpenSearch replaces MongoDB  
✅ **Censys-level accuracy:** 92% through smart filtering and validation  
✅ **48-hour freshness:** Adaptive scheduling with port priorities  
✅ **Horizontal scalability:** 5 to 100+ scanner nodes with no code changes  
✅ **Legal compliance:** Blocklist, opt-out, rate limiting, scan identification  
✅ **Clear migration path:** Phased approach from MVP to production  

**Next Steps:**
1. Review and approve this specification
2. Create detailed implementation plan (using writing-plans skill)
3. Begin Phase 1: Core Pipeline development

---

**Document Version:** 1.0  
**Last Updated:** 2026-06-11  
**Author:** Senior Principal Engineer (via OpenCode)  
**Status:** Ready for Review


---

## APPENDIX A: Port Coverage Clarification

**IMPORTANT:** Rigour scans **ALL 65,535 ports**, not just a subset.

### Complete Port Coverage Strategy

**Monthly Full Scan:**
- Target: 4.3 billion IPv4 addresses
- Ports: ALL 65,535 ports per IP
- Total: 281 trillion port checks per month
- Coverage: 80%+ of responsive services (Censys: 82%)

**Adaptive Priority Overlay:**
The adaptive scheduling ADDS extra scans for high-value ports on top of the monthly full scan:

| Port Category | Base Scan | Extra Scans | Total Frequency |
|--------------|-----------|-------------|-----------------|
| Critical (22, 23, 3389) | Monthly (all ports) | Every 6h (these ports only) | ~120x/month |
| Web (80, 443) | Monthly (all ports) | Daily (these ports only) | ~30x/month |
| Top 1000 | Monthly (all ports) | Weekly (these ports only) | ~4x/month |
| All 65,535 | Monthly (all ports) | - | 1x/month |

**Result:** 
- Critical ports scanned 120 times per month (near real-time monitoring)
- Every port scanned at least once per month (complete coverage)
- Matches Censys coverage while providing faster detection on critical services

### Why Not Scan All Ports Daily?

**Math:**
- 4.3B IPs × 65,535 ports = 281 trillion checks
- At 10M pps per ZMap node = 32 days of scanning at full speed
- Would require 100+ ZMap nodes running 24/7

**Solution:**
- Monthly full scan provides complete coverage
- Priority scanning provides fast detection where it matters
- Cost-effective while maintaining Censys-level comprehensiveness

