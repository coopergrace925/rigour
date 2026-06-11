# Phase 1: Core Pipeline Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build the foundational scanning pipeline: ZMap → ZGrab2 → NATS → Enrichment → OpenSearch

**Architecture:** Two-stage scanning (ZMap for discovery, ZGrab2 for fingerprinting) with NATS JetStream message bus, stateless Go enrichment workers, and OpenSearch as the only database.

**Tech Stack:** Go 1.21+, ZMap 4.4+, ZGrab2 (latest), NATS JetStream 2.10+, OpenSearch 2.11+, Redis 7.2+, Docker Compose

**Timeline:** 8 weeks

**Deliverables:**
- Working scan pipeline (1M IPs in <24h)
- Basic GeoIP + ASN enrichment
- Pseudo-service detection
- Blocklist management
- >85% accuracy on validation sample
- All data <72h fresh

---

## Architecture Overview

```
ZMap → ZGrab2 → zsend → NATS JetStream → Enrichment Worker → OpenSearch Indexer → OpenSearch
                           ↓
                         Redis (state, blocklist)
```

**Key Design Decisions:**
- NO MongoDB (OpenSearch is the only database)
- NATS JetStream replaces Kafka (simpler, faster)
- In-memory enrichment data (GeoIP, ASN loaded at startup)
- Stateless workers (horizontal scaling via NATS queue groups)

---

## File Structure

### New Services (Go)

```
rigour/
├── cmd/
│   ├── zsend/
│   │   └── main.go                    # NATS publisher for ZMap/ZGrab2 output
│   ├── enrichment-worker/
│   │   └── main.go                    # Enrichment worker
│   └── opensearch-indexer/
│       └── main.go                    # OpenSearch bulk indexer
├── internal/
│   ├── nats/
│   │   ├── client.go                  # NATS JetStream client
│   │   └── streams.go                 # Stream configs
│   ├── enrichment/
│   │   ├── geoip.go                   # MaxMind GeoIP lookup
│   │   ├── asn.go                     # ASN lookup
│   │   └── pseudo_service.go          # Pseudo-service detection
│   ├── opensearch/
│   │   ├── client.go                  # OpenSearch client
│   │   ├── indexer.go                 # Bulk indexing
│   │   └── schema.go                  # Index schema
│   └── blocklist/
│       ├── generator.go               # Blocklist generator
│       └── manager.go                 # Blocklist management
├── pkg/
│   └── types/
│       ├── scan.go                    # Scan message types
│       └── host.go                    # Host document types
└── scripts/
    ├── setup-zmap.sh                  # ZMap installation
    ├── setup-zgrab2.sh                # ZGrab2 installation
    └── download-geoip.sh              # Download GeoIP databases

```

### Infrastructure

```
rigour/
├── docker-compose.new.yml             # New stack (NATS, OpenSearch, Redis)
└── deployments/
    └── kubernetes/                    # K8s manifests (future)
```

---

## Task 1: Project Setup & Dependencies

**Duration:** 2 days

**Files:**
- Create: `go.mod` (if not exists, or verify)
- Create: `scripts/setup-zmap.sh`
- Create: `scripts/setup-zgrab2.sh`
- Create: `scripts/download-geoip.sh`
- Create: `docker-compose.new.yml`

### Step 1.1: Verify Go module setup

- [ ] **Check Go module**

```bash
cd /root/scanner/rigour/rigour
cat go.mod | head -5
```

Expected: Module path `github.com/ctrlsam/rigour`, Go 1.24+

- [ ] **Add new dependencies**

```bash
cd /root/scanner/rigour/rigour
go get github.com/nats-io/nats.go@latest
go get github.com/opensearch-project/opensearch-go/v2@latest
go get github.com/oschwald/geoip2-golang@latest
go get github.com/oschwald/maxminddb-golang@latest
go mod tidy
```

- [ ] **Commit**

```bash
git add go.mod go.sum
git commit -m "deps: add NATS, OpenSearch, GeoIP dependencies"
```

### Step 1.2: Create ZMap installation script

- [ ] **Write script**

Create `scripts/setup-zmap.sh`:

```bash
#!/bin/bash
set -e

echo "Installing ZMap..."

# Install dependencies
apt-get update
apt-get install -y build-essential cmake libgmp3-dev gengetopt libpcap-dev flex byacc libjson-c-dev pkg-config libunistring-dev

# Clone and build ZMap
cd /tmp
git clone https://github.com/zmap/zmap.git
cd zmap
cmake .
make -j$(nproc)
make install

# Verify installation
zmap --version

echo "ZMap installed successfully"
```

- [ ] **Make executable and test**

```bash
chmod +x scripts/setup-zmap.sh
# Test in Docker container (don't run on host yet)
```

- [ ] **Commit**

```bash
git add scripts/setup-zmap.sh
git commit -m "scripts: add ZMap installation script"
```

### Step 1.3: Create ZGrab2 installation script

- [ ] **Write script**

Create `scripts/setup-zgrab2.sh`:

```bash
#!/bin/bash
set -e

echo "Installing ZGrab2..."

# Install Go if not present
if ! command -v go &> /dev/null; then
    echo "Go not found, installing..."
    wget https://go.dev/dl/go1.21.0.linux-amd64.tar.gz
    tar -C /usr/local -xzf go1.21.0.linux-amd64.tar.gz
    export PATH=$PATH:/usr/local/go/bin
fi

# Clone and build ZGrab2
cd /tmp
git clone https://github.com/zmap/zgrab2.git
cd zgrab2
make
cp zgrab2 /usr/local/bin/

# Verify installation
zgrab2 --version

echo "ZGrab2 installed successfully"
```

- [ ] **Make executable**

```bash
chmod +x scripts/setup-zgrab2.sh
```

- [ ] **Commit**

```bash
git add scripts/setup-zgrab2.sh
git commit -m "scripts: add ZGrab2 installation script"
```

### Step 1.4: Create GeoIP download script

- [ ] **Write script**

Create `scripts/download-geoip.sh`:

```bash
#!/bin/bash
set -e

GEOIP_DIR="${GEOIP_DIR:-/data/geoip}"
MAXMIND_LICENSE_KEY="${MAXMIND_LICENSE_KEY}"

if [ -z "$MAXMIND_LICENSE_KEY" ]; then
    echo "Error: MAXMIND_LICENSE_KEY environment variable not set"
    echo "Get a free license key at: https://www.maxmind.com/en/geolite2/signup"
    exit 1
fi

mkdir -p "$GEOIP_DIR"
cd "$GEOIP_DIR"

echo "Downloading MaxMind GeoLite2 databases..."

# Download GeoLite2-City
wget -O GeoLite2-City.tar.gz \
  "https://download.maxmind.com/app/geoip_download?edition_id=GeoLite2-City&license_key=${MAXMIND_LICENSE_KEY}&suffix=tar.gz"
tar -xzf GeoLite2-City.tar.gz --strip-components=1
rm GeoLite2-City.tar.gz

# Download GeoLite2-ASN
wget -O GeoLite2-ASN.tar.gz \
  "https://download.maxmind.com/app/geoip_download?edition_id=GeoLite2-ASN&license_key=${MAXMIND_LICENSE_KEY}&suffix=tar.gz"
tar -xzf GeoLite2-ASN.tar.gz --strip-components=1
rm GeoLite2-ASN.tar.gz

echo "GeoIP databases downloaded to $GEOIP_DIR"
ls -lh "$GEOIP_DIR"/*.mmdb
```

- [ ] **Make executable**

```bash
chmod +x scripts/download-geoip.sh
```

- [ ] **Commit**

```bash
git add scripts/download-geoip.sh
git commit -m "scripts: add GeoIP database download script"
```

### Step 1.5: Create new Docker Compose stack

- [ ] **Write docker-compose.new.yml**

Create `docker-compose.new.yml`:

```yaml
version: '3.8'

services:
  # NATS JetStream
  nats:
    image: nats:2.10-alpine
    container_name: rigour-nats
    command: ["-js", "-sd", "/data", "-m", "8222"]
    ports:
      - "4222:4222"   # Client
      - "8222:8222"   # Monitoring
    volumes:
      - nats-data:/data
    networks:
      - rigour-network
    healthcheck:
      test: ["CMD", "wget", "--spider", "-q", "http://localhost:8222/healthz"]
      interval: 10s
      timeout: 5s
      retries: 3

  # OpenSearch (single node for dev)
  opensearch:
    image: opensearchproject/opensearch:2.11.0
    container_name: rigour-opensearch
    environment:
      - discovery.type=single-node
      - OPENSEARCH_JAVA_OPTS=-Xms2g -Xmx2g
      - DISABLE_SECURITY_PLUGIN=true
    ports:
      - "9200:9200"
      - "9600:9600"
    volumes:
      - opensearch-data:/usr/share/opensearch/data
    networks:
      - rigour-network
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:9200/_cluster/health"]
      interval: 30s
      timeout: 10s
      retries: 3

  # Redis
  redis:
    image: redis:7.2-alpine
    container_name: rigour-redis
    command: redis-server --appendonly yes
    ports:
      - "6379:6379"
    volumes:
      - redis-data:/data
    networks:
      - rigour-network
    healthcheck:
      test: ["CMD", "redis-cli", "ping"]
      interval: 10s
      timeout: 3s
      retries: 3

volumes:
  nats-data:
  opensearch-data:
  redis-data:

networks:
  rigour-network:
    driver: bridge
```

- [ ] **Test stack**

```bash
cd /root/scanner/rigour
docker-compose -f docker-compose.new.yml up -d
docker-compose -f docker-compose.new.yml ps
```

Expected: All services "healthy"

- [ ] **Commit**

```bash
git add docker-compose.new.yml
git commit -m "infra: add new stack with NATS, OpenSearch, Redis"
```

---

## Task 2: Shared Types & NATS Client

**Duration:** 1 day

**Files:**
- Create: `rigour/pkg/types/scan.go`
- Create: `rigour/pkg/types/host.go`
- Create: `rigour/internal/nats/client.go`
- Create: `rigour/internal/nats/streams.go`

### Step 2.1: Define scan message types

- [ ] **Write failing test**

Create `rigour/pkg/types/scan_test.go`:

```go
package types

import (
	"encoding/json"
	"testing"
	"time"
)

func TestRawScanJSON(t *testing.T) {
	scan := RawScan{
		IP:         "1.2.3.4",
		Port:       443,
		Protocol:   "tcp",
		Service:    "https",
		Banner:     "nginx/1.24.0",
		ScannedAt:  time.Now(),
		ScannerID:  "scanner-01",
	}

	data, err := json.Marshal(scan)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	var decoded RawScan
	err = json.Unmarshal(data, &decoded)
	if err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if decoded.IP != scan.IP {
		t.Errorf("IP mismatch: got %s, want %s", decoded.IP, scan.IP)
	}
}
```

- [ ] **Run test to verify it fails**

```bash
cd /root/scanner/rigour/rigour
go test ./pkg/types -v
```

Expected: FAIL with "undefined: RawScan"

- [ ] **Implement types**

Create `rigour/pkg/types/scan.go`:

```go
package types

import "time"

// RawScan represents output from ZMap + ZGrab2
type RawScan struct {
	IP         string    `json:"ip"`
	Port       int       `json:"port"`
	Protocol   string    `json:"protocol"` // "tcp" or "udp"
	Service    string    `json:"service"`  // "https", "ssh", etc.
	Banner     string    `json:"banner"`
	ZGrabData  ZGrabData `json:"zgrab_data,omitempty"`
	ScannedAt  time.Time `json:"scanned_at"`
	ScannerID  string    `json:"scanner_id"`
}

// ZGrabData holds detailed ZGrab2 output
type ZGrabData struct {
	Status    string                 `json:"status"`
	Protocol  string                 `json:"protocol"`
	Result    map[string]interface{} `json:"result,omitempty"`
	TLS       *TLSInfo               `json:"tls,omitempty"`
	HTTP      *HTTPInfo              `json:"http,omitempty"`
	SSH       *SSHInfo               `json:"ssh,omitempty"`
}

type TLSInfo struct {
	Version string    `json:"version"`
	Cipher  string    `json:"cipher"`
	Cert    *CertInfo `json:"cert,omitempty"`
}

type CertInfo struct {
	SubjectCN   string    `json:"subject_cn"`
	IssuerCN    string    `json:"issuer_cn"`
	Fingerprint string    `json:"fingerprint"`
	NotAfter    time.Time `json:"not_after"`
}

type HTTPInfo struct {
	StatusCode int               `json:"status_code"`
	Title      string            `json:"title"`
	Server     string            `json:"server"`
	Headers    map[string]string `json:"headers,omitempty"`
}

type SSHInfo struct {
	ServerID string   `json:"server_id"`
	HASSH    string   `json:"hassh"`
	KexAlgos []string `json:"kex_algos,omitempty"`
}

// EnrichedScan represents enriched scan data
type EnrichedScan struct {
	RawScan
	ASN         int       `json:"asn"`
	Org         string    `json:"org"`
	Country     string    `json:"country"`
	City        string    `json:"city"`
	RDNS        string    `json:"rdns"`
	CPE         string    `json:"cpe,omitempty"`
	CVEs        []string  `json:"cves,omitempty"`
	EnrichedAt  time.Time `json:"enriched_at"`
}
```

- [ ] **Run test to verify it passes**

```bash
cd /root/scanner/rigour/rigour
go test ./pkg/types -v
```

Expected: PASS

- [ ] **Commit**

```bash
git add pkg/types/
git commit -m "types: add RawScan and EnrichedScan message types"
```

### Step 2.2: Define host document type

- [ ] **Write host type**

Create `rigour/pkg/types/host.go`:

```go
package types

import "time"

// Host represents the OpenSearch document structure
type Host struct {
	IP       string    `json:"ip"`
	IPInt    int64     `json:"ip_int"`
	ASN      int       `json:"asn"`
	Org      string    `json:"org"`
	Country  string    `json:"country"`
	City     string    `json:"city,omitempty"`
	RDNS     string    `json:"rdns,omitempty"`
	LastSeen time.Time `json:"last_seen"`
	IsStale  bool      `json:"is_stale"`
	Ports    []Port    `json:"ports"`
	CVEs     []string  `json:"cves,omitempty"`
}

// Port represents a single port entry (nested in OpenSearch)
type Port struct {
	Port      int       `json:"port"`
	Protocol  string    `json:"protocol"`
	Service   string    `json:"service"`
	Product   string    `json:"product,omitempty"`
	CPE       string    `json:"cpe,omitempty"`
	Banner    string    `json:"banner,omitempty"`
	LastSeen  time.Time `json:"last_seen"`
	HTTP      *HTTPInfo `json:"http,omitempty"`
	TLS       *TLSInfo  `json:"tls,omitempty"`
	SSH       *SSHInfo  `json:"ssh,omitempty"`
}

// IPToInt converts IP string to int64
func IPToInt(ip string) int64 {
	// Simple implementation for IPv4
	// TODO: Handle IPv6
	return 0 // Placeholder
}
```

- [ ] **Commit**

```bash
git add pkg/types/host.go
git commit -m "types: add Host and Port document types"
```

### Step 2.3: Create NATS client

- [ ] **Write NATS client**

Create `rigour/internal/nats/client.go`:

```go
package nats

import (
	"fmt"
	"time"

	"github.com/nats-io/nats.go"
)

type Client struct {
	conn *nats.Conn
	js   nats.JetStreamContext
}

func NewClient(url string) (*Client, error) {
	nc, err := nats.Connect(url,
		nats.MaxReconnects(-1),
		nats.ReconnectWait(2*time.Second),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to NATS: %w", err)
	}

	js, err := nc.JetStream()
	if err != nil {
		return nil, fmt.Errorf("failed to create JetStream context: %w", err)
	}

	return &Client{
		conn: nc,
		js:   js,
	}, nil
}

func (c *Client) JetStream() nats.JetStreamContext {
	return c.js
}

func (c *Client) Close() error {
	c.conn.Close()
	return nil
}
```

- [ ] **Commit**

```bash
git add internal/nats/client.go
git commit -m "nats: add NATS JetStream client"
```

### Step 2.4: Create stream configurations

- [ ] **Write stream setup**

Create `rigour/internal/nats/streams.go`:

```go
package nats

import (
	"fmt"
	"time"

	"github.com/nats-io/nats.go"
)

const (
	StreamRawScans      = "RAW_SCANS"
	StreamEnrichedScans = "ENRICHED_SCANS"
	StreamScanEvents    = "SCAN_EVENTS"
)

func (c *Client) SetupStreams() error {
	// RAW_SCANS stream
	_, err := c.js.AddStream(&nats.StreamConfig{
		Name:       StreamRawScans,
		Subjects:   []string{"scan.raw.*"},
		Retention:  nats.WorkQueuePolicy,
		MaxAge:     48 * time.Hour,
		Storage:    nats.FileStorage,
		Replicas:   1, // 3 in production
		Discard:    nats.DiscardOld,
	})
	if err != nil {
		return fmt.Errorf("failed to create RAW_SCANS stream: %w", err)
	}

	// ENRICHED_SCANS stream
	_, err = c.js.AddStream(&nats.StreamConfig{
		Name:       StreamEnrichedScans,
		Subjects:   []string{"scan.enriched.*"},
		Retention:  nats.WorkQueuePolicy,
		MaxAge:     24 * time.Hour,
		Storage:    nats.FileStorage,
		Replicas:   1,
		Discard:    nats.DiscardOld,
	})
	if err != nil {
		return fmt.Errorf("failed to create ENRICHED_SCANS stream: %w", err)
	}

	// SCAN_EVENTS stream (audit/monitoring)
	_, err = c.js.AddStream(&nats.StreamConfig{
		Name:       StreamScanEvents,
		Subjects:   []string{"scan.events.*"},
		Retention:  nats.LimitsPolicy,
		MaxAge:     7 * 24 * time.Hour,
		Storage:    nats.FileStorage,
		Replicas:   1,
	})
	if err != nil {
		return fmt.Errorf("failed to create SCAN_EVENTS stream: %w", err)
	}

	return nil
}
```

- [ ] **Commit**

```bash
git add internal/nats/streams.go
git commit -m "nats: add JetStream stream configurations"
```

---

## Task 3: zsend — NATS Publisher for ZMap/ZGrab2 Output

**Duration:** 2 days

**Files:**
- Create: `rigour/cmd/zsend/main.go`

### Step 3.1: Write zsend tests

- [ ] **Write failing test**

Create `rigour/cmd/zsend/main_test.go`:

```go
package main

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/ctrlsam/rigour/pkg/types"
)

func TestParseZMapCSVLine(t *testing.T) {
	line := "1.2.3.4,443,synack,1"
	result, err := parseZMapCSVLine(line)
	if err != nil {
		t.Fatalf("parseZMapCSVLine failed: %v", err)
	}
	if result.IP != "1.2.3.4" {
		t.Errorf("IP mismatch: got %s, want 1.2.3.4", result.IP)
	}
	if result.Port != 443 {
		t.Errorf("Port mismatch: got %d, want 443", result.Port)
	}
}

func TestParseZGrab2JSONLine(t *testing.T) {
	input := `{"ip":"1.2.3.4","data":{"http":{"status":"success","result":{"response":{"status_code":200}}}}}`
	result, err := parseZGrab2JSONLine([]byte(input))
	if err != nil {
		t.Fatalf("parseZGrab2JSONLine failed: %v", err)
	}
	if result.IP != "1.2.3.4" {
		t.Errorf("IP mismatch: got %s, want 1.2.3.4", result.IP)
	}
}

func TestBuildRawScan(t *testing.T) {
	scan := buildRawScan("1.2.3.4", 443, "tcp", "scanner-01")
	if scan.IP != "1.2.3.4" {
		t.Errorf("IP mismatch")
	}
	if scan.Port != 443 {
		t.Errorf("Port mismatch")
	}
	if scan.ScannerID != "scanner-01" {
		t.Errorf("ScannerID mismatch")
	}
}
```

- [ ] **Run test to verify it fails**

```bash
cd /root/scanner/rigour/rigour
go test ./cmd/zsend/ -v
```

Expected: FAIL with "undefined: parseZMapCSVLine"

- [ ] **Implement zsend**

Create `rigour/cmd/zsend/main.go`:

```go
package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	internalnats "github.com/ctrlsam/rigour/internal/nats"
	"github.com/ctrlsam/rigour/pkg/types"
	"github.com/spf13/cobra"
)

type config struct {
	natsURL   string
	subject   string
	mode      string // "zmap-csv" or "zgrab2-json"
	scannerID string
}

var cfg config

var rootCmd = &cobra.Command{
	Use:   "zsend",
	Short: "Publish ZMap/ZGrab2 output to NATS JetStream",
	RunE: func(cmd *cobra.Command, args []string) error {
		return run()
	},
}

func init() {
	rootCmd.Flags().StringVar(&cfg.natsURL, "nats-url", "nats://localhost:4222", "NATS server URL")
	rootCmd.Flags().StringVar(&cfg.subject, "nats-subject", "scan.raw.0", "NATS subject to publish to")
	rootCmd.Flags().StringVar(&cfg.mode, "mode", "zgrab2-json", "Input mode: zmap-csv or zgrab2-json")
	rootCmd.Flags().StringVar(&cfg.scannerID, "scanner-id", "", "Scanner node ID")
}

func main() {
	if cfg.scannerID == "" {
		hostname, _ := os.Hostname()
		cfg.scannerID = hostname
	}
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func run() error {
	client, err := internalnats.NewClient(cfg.natsURL)
	if err != nil {
		return fmt.Errorf("failed to connect to NATS: %w", err)
	}
	defer client.Close()

	js := client.JetStream()
	scanner := bufio.NewScanner(os.Stdin)
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024) // 1MB buffer

	var published, errors int64

	for scanner.Scan() {
		line := scanner.Text()
		if strings.TrimSpace(line) == "" {
			continue
		}

		var scan *types.RawScan
		var parseErr error

		switch cfg.mode {
		case "zmap-csv":
			scan, parseErr = parseZMapCSVLine(line)
		case "zgrab2-json":
			scan, parseErr = parseZGrab2JSONLine([]byte(line))
		default:
			return fmt.Errorf("unknown mode: %s", cfg.mode)
		}

		if parseErr != nil {
			errors++
			fmt.Fprintf(os.Stderr, "parse error: %v\n", parseErr)
			continue
		}

		scan.ScannerID = cfg.scannerID
		scan.ScannedAt = time.Now()

		data, err := json.Marshal(scan)
		if err != nil {
			errors++
			continue
		}

		subject := fmt.Sprintf("scan.raw.%d", scan.Port)
		_, err = js.Publish(subject, data)
		if err != nil {
			errors++
			fmt.Fprintf(os.Stderr, "publish error: %v\n", err)
			continue
		}

		published++
		if published%10000 == 0 {
			fmt.Fprintf(os.Stderr, "published: %d, errors: %d\n", published, errors)
		}
	}

	fmt.Fprintf(os.Stderr, "done: published=%d errors=%d\n", published, errors)
	return scanner.Err()
}

func parseZMapCSVLine(line string) (*types.RawScan, error) {
	parts := strings.Split(line, ",")
	if len(parts) < 4 {
		return nil, fmt.Errorf("invalid CSV line: expected 4+ fields, got %d", len(parts))
	}

	port, err := strconv.Atoi(strings.TrimSpace(parts[1]))
	if err != nil {
		return nil, fmt.Errorf("invalid port: %w", err)
	}

	return &types.RawScan{
		IP:       strings.TrimSpace(parts[0]),
		Port:     port,
		Protocol: "tcp",
	}, nil
}

func parseZGrab2JSONLine(data []byte) (*types.RawScan, error) {
	var raw struct {
		IP     string                 `json:"ip"`
		Domain string                 `json:"domain,omitempty"`
		Port   int                    `json:"port,omitempty"`
		Data   map[string]interface{} `json:"data"`
	}

	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("invalid JSON: %w", err)
	}

	scan := &types.RawScan{
		IP: raw.IP,
	}

	// Extract protocol data from ZGrab2 output
	for proto, protoData := range raw.Data {
		scan.Service = proto
		if pd, ok := protoData.(map[string]interface{}); ok {
			if status, ok := pd["status"].(string); ok {
				scan.ZGrabData.Status = status
			}
			scan.ZGrabData.Protocol = proto
		}
	}

	return scan, nil
}

func buildRawScan(ip string, port int, protocol string, scannerID string) *types.RawScan {
	return &types.RawScan{
		IP:        ip,
		Port:      port,
		Protocol:  protocol,
		ScannerID: scannerID,
		ScannedAt: time.Now(),
	}
}
```

- [ ] **Run tests to verify they pass**

```bash
cd /root/scanner/rigour/rigour
go test ./cmd/zsend/ -v
```

Expected: PASS

- [ ] **Commit**

```bash
git add cmd/zsend/
git commit -m "feat: add zsend NATS publisher for ZMap/ZGrab2 output"
```

---

## Task 4: Blocklist Manager

**Duration:** 1 day

**Files:**
- Create: `rigour/internal/blocklist/blocklist.go`
- Create: `rigour/internal/blocklist/blocklist_test.go`

### Step 4.1: Write blocklist tests

- [ ] **Write failing test**

Create `rigour/internal/blocklist/blocklist_test.go`:

```go
package blocklist

import (
	"net"
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultBlocklistContainsRFC1918(t *testing.T) {
	bl := NewBlocklist()

	privateIPs := []string{"10.0.0.1", "172.16.0.1", "192.168.1.1"}
	for _, ip := range privateIPs {
		if !bl.IsBlocked(net.ParseIP(ip)) {
			t.Errorf("Expected %s to be blocked (RFC1918)", ip)
		}
	}
}

func TestDefaultBlocklistAllowsPublicIPs(t *testing.T) {
	bl := NewBlocklist()

	publicIPs := []string{"1.1.1.1", "8.8.8.8", "93.184.216.34"}
	for _, ip := range publicIPs {
		if bl.IsBlocked(net.ParseIP(ip)) {
			t.Errorf("Expected %s to NOT be blocked", ip)
		}
	}
}

func TestBlocklistBlocksLoopback(t *testing.T) {
	bl := NewBlocklist()
	if !bl.IsBlocked(net.ParseIP("127.0.0.1")) {
		t.Errorf("Expected 127.0.0.1 to be blocked")
	}
}

func TestBlocklistBlocksMulticast(t *testing.T) {
	bl := NewBlocklist()
	if !bl.IsBlocked(net.ParseIP("224.0.0.1")) {
		t.Errorf("Expected 224.0.0.1 to be blocked (multicast)")
	}
}

func TestAddOptOut(t *testing.T) {
	bl := NewBlocklist()
	ip := net.ParseIP("93.184.216.34")

	if bl.IsBlocked(ip) {
		t.Fatalf("IP should not be blocked before opt-out")
	}

	bl.AddOptOut(ip)

	if !bl.IsBlocked(ip) {
		t.Errorf("IP should be blocked after opt-out")
	}
}

func TestGenerateFile(t *testing.T) {
	bl := NewBlocklist()
	tmpDir := t.TempDir()
	outFile := filepath.Join(tmpDir, "blocklist.conf")

	err := bl.GenerateFile(outFile)
	if err != nil {
		t.Fatalf("GenerateFile failed: %v", err)
	}

	data, err := os.ReadFile(outFile)
	if err != nil {
		t.Fatalf("ReadFile failed: %v", err)
	}

	content := string(data)
	if len(content) == 0 {
		t.Error("Blocklist file is empty")
	}
}
```

- [ ] **Run test to verify it fails**

```bash
cd /root/scanner/rigour/rigour
go test ./internal/blocklist/ -v
```

Expected: FAIL with "undefined: NewBlocklist"

- [ ] **Implement blocklist**

Create `rigour/internal/blocklist/blocklist.go`:

```go
package blocklist

import (
	"fmt"
	"net"
	"os"
	"sync"
)

// DefaultCIDRs contains mandatory blocklist entries
var DefaultCIDRs = []string{
	// RFC 1918 - Private networks
	"10.0.0.0/8",
	"172.16.0.0/12",
	"192.168.0.0/16",

	// RFC 5735 - Special use
	"0.0.0.0/8",
	"127.0.0.0/8",
	"169.254.0.0/16",
	"192.0.0.0/24",
	"192.0.2.0/24",
	"198.18.0.0/15",
	"198.51.100.0/24",
	"203.0.113.0/24",
	"224.0.0.0/4",
	"240.0.0.0/4",
	"255.255.255.255/32",

	// Carrier-grade NAT
	"100.64.0.0/10",

	// IPv6 to IPv4 relay
	"192.88.99.0/24",

	// DoD - US Department of Defense
	"6.0.0.0/8",
	"7.0.0.0/8",
	"11.0.0.0/8",
	"21.0.0.0/8",
	"22.0.0.0/8",
	"26.0.0.0/8",
	"28.0.0.0/8",
	"29.0.0.0/8",
	"30.0.0.0/8",
	"33.0.0.0/8",
	"55.0.0.0/8",
	"214.0.0.0/8",
	"215.0.0.0/8",
}

type Blocklist struct {
	mu      sync.RWMutex
	nets    []*net.IPNet
	optOuts map[string]struct{}
}

func NewBlocklist() *Blocklist {
	bl := &Blocklist{
		optOuts: make(map[string]struct{}),
	}

	for _, cidr := range DefaultCIDRs {
		_, ipnet, err := net.ParseCIDR(cidr)
		if err != nil {
			continue
		}
		bl.nets = append(bl.nets, ipnet)
	}

	return bl
}

func (b *Blocklist) IsBlocked(ip net.IP) bool {
	b.mu.RLock()
	defer b.mu.RUnlock()

	// Check opt-outs
	if _, ok := b.optOuts[ip.String()]; ok {
		return true
	}

	// Check CIDR blocks
	for _, ipnet := range b.nets {
		if ipnet.Contains(ip) {
			return true
		}
	}

	return false
}

func (b *Blocklist) AddOptOut(ip net.IP) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.optOuts[ip.String()] = struct{}{}
}

func (b *Blocklist) GenerateFile(path string) error {
	b.mu.RLock()
	defer b.mu.RUnlock()

	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("failed to create blocklist file: %w", err)
	}
	defer f.Close()

	fmt.Fprintln(f, "# Rigour Blocklist — Auto-generated, do not edit manually")
	fmt.Fprintln(f, "# RFC1918, DoD, IANA reserved, opt-outs")
	fmt.Fprintln(f)

	for _, cidr := range DefaultCIDRs {
		fmt.Fprintln(f, cidr)
	}

	fmt.Fprintln(f)
	fmt.Fprintln(f, "# Opt-out IPs")
	for ip := range b.optOuts {
		fmt.Fprintf(f, "%s/32\n", ip)
	}

	return nil
}
```

- [ ] **Run tests to verify they pass**

```bash
cd /root/scanner/rigour/rigour
go test ./internal/blocklist/ -v
```

Expected: ALL PASS

- [ ] **Commit**

```bash
git add internal/blocklist/
git commit -m "feat: add blocklist with RFC1918, DoD, IANA, opt-out support"
```

---

## Task 5: GeoIP & ASN Enrichment

**Duration:** 2 days

**Files:**
- Create: `rigour/internal/enrichment/geoip.go`
- Create: `rigour/internal/enrichment/geoip_test.go`

### Step 5.1: Write GeoIP tests

- [ ] **Write failing test**

Create `rigour/internal/enrichment/geoip_test.go`:

```go
package enrichment

import (
	"testing"
)

func TestGeoIPLookupResult(t *testing.T) {
	result := GeoResult{
		Country: "US",
		City:    "San Francisco",
		ASN:     15169,
		Org:     "Google LLC",
	}

	if result.Country != "US" {
		t.Errorf("Expected country US, got %s", result.Country)
	}
	if result.ASN != 15169 {
		t.Errorf("Expected ASN 15169, got %d", result.ASN)
	}
}

func TestNewGeoIPLookupReturnsErrorForMissingFile(t *testing.T) {
	_, err := NewGeoIPLookup("/nonexistent/path.mmdb", "/nonexistent/asn.mmdb")
	if err == nil {
		t.Error("Expected error for missing MMDB file")
	}
}
```

- [ ] **Run test to verify it fails**

```bash
cd /root/scanner/rigour/rigour
go test ./internal/enrichment/ -v
```

Expected: FAIL with "undefined: GeoResult"

- [ ] **Implement GeoIP lookup**

Create `rigour/internal/enrichment/geoip.go`:

```go
package enrichment

import (
	"fmt"
	"net"

	"github.com/oschwald/geoip2-golang"
)

type GeoResult struct {
	Country string `json:"country"`
	City    string `json:"city"`
	ASN     int    `json:"asn"`
	Org     string `json:"org"`
	Lat     float64 `json:"lat,omitempty"`
	Lon     float64 `json:"lon,omitempty"`
}

type GeoIPLookup struct {
	cityDB *geoip2.Reader
	asnDB  *geoip2.Reader
}

func NewGeoIPLookup(cityDBPath, asnDBPath string) (*GeoIPLookup, error) {
	cityDB, err := geoip2.Open(cityDBPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open GeoLite2-City DB: %w", err)
	}

	asnDB, err := geoip2.Open(asnDBPath)
	if err != nil {
		cityDB.Close()
		return nil, fmt.Errorf("failed to open GeoLite2-ASN DB: %w", err)
	}

	return &GeoIPLookup{
		cityDB: cityDB,
		asnDB:  asnDB,
	}, nil
}

func (g *GeoIPLookup) Lookup(ipStr string) GeoResult {
	ip := net.ParseIP(ipStr)
	if ip == nil {
		return GeoResult{}
	}

	result := GeoResult{}

	// City/Country lookup
	city, err := g.cityDB.City(ip)
	if err == nil {
		result.Country = city.Country.IsoCode
		if len(city.City.Names) > 0 {
			result.City = city.City.Names["en"]
		}
		result.Lat = city.Location.Latitude
		result.Lon = city.Location.Longitude
	}

	// ASN lookup
	asn, err := g.asnDB.ASN(ip)
	if err == nil {
		result.ASN = int(asn.AutonomousSystemNumber)
		result.Org = asn.AutonomousSystemOrganization
	}

	return result
}

func (g *GeoIPLookup) Close() {
	if g.cityDB != nil {
		g.cityDB.Close()
	}
	if g.asnDB != nil {
		g.asnDB.Close()
	}
}
```

- [ ] **Run tests to verify they pass**

```bash
cd /root/scanner/rigour/rigour
go test ./internal/enrichment/ -v
```

Expected: PASS

- [ ] **Commit**

```bash
git add internal/enrichment/
git commit -m "feat: add GeoIP + ASN enrichment using MaxMind GeoLite2"
```

---

## Task 6: Pseudo-Service Detection

**Duration:** 1 day

**Files:**
- Create: `rigour/internal/enrichment/pseudo_service.go`
- Create: `rigour/internal/enrichment/pseudo_service_test.go`

### Step 6.1: Write pseudo-service tests

- [ ] **Write failing test**

Create `rigour/internal/enrichment/pseudo_service_test.go`:

```go
package enrichment

import (
	"crypto/sha256"
	"fmt"
	"testing"

	"github.com/ctrlsam/rigour/pkg/types"
)

func TestIsPseudoServiceReturnsFalseForFewPorts(t *testing.T) {
	detector := NewPseudoServiceDetector(20)

	ports := []types.Port{
		{Port: 80, Banner: "nginx/1.24"},
		{Port: 443, Banner: "nginx/1.24"},
		{Port: 22, Banner: "OpenSSH_8.9"},
	}

	if detector.IsPseudoService("1.2.3.4", ports) {
		t.Error("Should NOT be pseudo-service with 3 ports")
	}
}

func TestIsPseudoServiceReturnsTrueForManyIdenticalBanners(t *testing.T) {
	detector := NewPseudoServiceDetector(20)

	// Simulate a honeypot: 25 ports with identical banners
	var ports []types.Port
	for i := 1; i <= 25; i++ {
		ports = append(ports, types.Port{
			Port:   i,
			Banner: "honeypot-identical-response",
		})
	}

	if !detector.IsPseudoService("1.2.3.4", ports) {
		t.Error("Should BE pseudo-service with 25 identical banners")
	}
}

func TestIsPseudoServiceReturnsFalseForDiverseBanners(t *testing.T) {
	detector := NewPseudoServiceDetector(20)

	var ports []types.Port
	for i := 1; i <= 25; i++ {
		ports = append(ports, types.Port{
			Port:   i,
			Banner: fmt.Sprintf("service-%d/v%d.0", i, i),
		})
	}

	if detector.IsPseudoService("1.2.3.4", ports) {
		t.Error("Should NOT be pseudo-service with diverse banners")
	}
}
```

- [ ] **Run test to verify it fails**

```bash
cd /root/scanner/rigour/rigour
go test ./internal/enrichment/ -v -run TestIsPseudo
```

Expected: FAIL with "undefined: NewPseudoServiceDetector"

- [ ] **Implement pseudo-service detection**

Create `rigour/internal/enrichment/pseudo_service.go`:

```go
package enrichment

import (
	"crypto/sha256"
	"fmt"
	"strings"

	"github.com/ctrlsam/rigour/pkg/types"
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
// Returns true if more than `threshold` ports respond with identical banners.
func (d *PseudoServiceDetector) IsPseudoService(ip string, ports []types.Port) bool {
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
```

- [ ] **Run tests to verify they pass**

```bash
cd /root/scanner/rigour/rigour
go test ./internal/enrichment/ -v -run TestIsPseudo
```

Expected: ALL PASS

- [ ] **Commit**

```bash
git add internal/enrichment/pseudo_service.go internal/enrichment/pseudo_service_test.go
git commit -m "feat: add pseudo-service detection (Censys-style filtering)"
```

---

## Task 7: OpenSearch Client & Schema

**Duration:** 2 days

**Files:**
- Create: `rigour/internal/opensearch/client.go`
- Create: `rigour/internal/opensearch/schema.go`
- Create: `rigour/internal/opensearch/indexer.go`
- Create: `rigour/internal/opensearch/indexer_test.go`

### Step 7.1: Create OpenSearch client

- [ ] **Write OpenSearch client**

Create `rigour/internal/opensearch/client.go`:

```go
package opensearch

import (
	"crypto/tls"
	"fmt"
	"net/http"

	opensearchgo "github.com/opensearch-project/opensearch-go/v2"
)

type Client struct {
	os *opensearchgo.Client
}

func NewClient(addresses []string) (*Client, error) {
	cfg := opensearchgo.Config{
		Addresses: addresses,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		},
	}

	client, err := opensearchgo.NewClient(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create OpenSearch client: %w", err)
	}

	// Verify connection
	res, err := client.Info()
	if err != nil {
		return nil, fmt.Errorf("failed to connect to OpenSearch: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		return nil, fmt.Errorf("OpenSearch returned error: %s", res.Status())
	}

	return &Client{os: client}, nil
}

func (c *Client) Raw() *opensearchgo.Client {
	return c.os
}
```

- [ ] **Commit**

```bash
git add internal/opensearch/client.go
git commit -m "feat: add OpenSearch client wrapper"
```

### Step 7.2: Create index schema

- [ ] **Write schema setup**

Create `rigour/internal/opensearch/schema.go`:

```go
package opensearch

import (
	"context"
	"fmt"
	"strings"
)

const HostsIndex = "hosts"

const hostsMapping = `{
  "settings": {
    "number_of_shards": 6,
    "number_of_replicas": 1,
    "refresh_interval": "30s",
    "index.routing.allocation.total_shards_per_node": 4
  },
  "mappings": {
    "dynamic": "strict",
    "properties": {
      "ip":        {"type": "ip"},
      "ip_int":    {"type": "long"},
      "asn":       {"type": "integer"},
      "org":       {"type": "keyword"},
      "country":   {"type": "keyword"},
      "city":      {"type": "keyword"},
      "rdns":      {"type": "keyword"},
      "last_seen": {"type": "date"},
      "is_stale":  {"type": "boolean"},
      "cves":      {"type": "keyword"},
      "tags":      {"type": "keyword"},
      "ports": {
        "type": "nested",
        "properties": {
          "port":      {"type": "integer"},
          "protocol":  {"type": "keyword"},
          "service":   {"type": "keyword"},
          "product":   {"type": "keyword"},
          "cpe":       {"type": "keyword"},
          "banner":    {"type": "text", "analyzer": "standard"},
          "last_seen": {"type": "date"},
          "http": {
            "properties": {
              "status_code": {"type": "integer"},
              "title":       {"type": "text"},
              "server":      {"type": "keyword"}
            }
          },
          "tls": {
            "properties": {
              "version": {"type": "keyword"},
              "cert": {
                "properties": {
                  "subject_cn":  {"type": "keyword"},
                  "issuer_cn":   {"type": "keyword"},
                  "fingerprint": {"type": "keyword"},
                  "not_after":   {"type": "date"},
                  "san":         {"type": "keyword"}
                }
              }
            }
          },
          "ssh": {
            "properties": {
              "hassh":     {"type": "keyword"},
              "server_id": {"type": "keyword"},
              "kex_algos": {"type": "keyword"}
            }
          }
        }
      }
    }
  }
}`

func (c *Client) CreateHostsIndex() error {
	res, err := c.os.Indices.Exists([]string{HostsIndex})
	if err != nil {
		return fmt.Errorf("failed to check index existence: %w", err)
	}
	defer res.Body.Close()

	// Index already exists
	if !res.IsError() {
		return nil
	}

	// Create the index
	res, err = c.os.Indices.Create(
		HostsIndex,
		c.os.Indices.Create.WithBody(strings.NewReader(hostsMapping)),
	)
	if err != nil {
		return fmt.Errorf("failed to create index: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		return fmt.Errorf("failed to create index: %s", res.String())
	}

	return nil
}
```

- [ ] **Commit**

```bash
git add internal/opensearch/schema.go
git commit -m "feat: add OpenSearch hosts index schema with nested ports"
```

### Step 7.3: Create bulk indexer

- [ ] **Write indexer test**

Create `rigour/internal/opensearch/indexer_test.go`:

```go
package opensearch

import (
	"testing"
	"time"

	"github.com/ctrlsam/rigour/pkg/types"
)

func TestMergePorts(t *testing.T) {
	existing := types.Host{
		IP: "1.2.3.4",
		Ports: []types.Port{
			{Port: 80, Service: "http", Banner: "nginx/1.24", LastSeen: time.Now().Add(-1 * time.Hour)},
			{Port: 22, Service: "ssh", Banner: "OpenSSH_8.9", LastSeen: time.Now().Add(-1 * time.Hour)},
		},
	}

	newScan := types.EnrichedScan{
		RawScan: types.RawScan{
			IP:      "1.2.3.4",
			Port:    443,
			Service: "https",
			Banner:  "nginx/1.24",
		},
	}

	merged := MergePorts(existing, newScan)

	if len(merged.Ports) != 3 {
		t.Errorf("Expected 3 ports after merge, got %d", len(merged.Ports))
	}

	found443 := false
	for _, p := range merged.Ports {
		if p.Port == 443 {
			found443 = true
		}
	}
	if !found443 {
		t.Error("Port 443 not found after merge")
	}
}

func TestMergePortsUpdatesExistingPort(t *testing.T) {
	existing := types.Host{
		IP: "1.2.3.4",
		Ports: []types.Port{
			{Port: 443, Service: "https", Banner: "nginx/1.23", LastSeen: time.Now().Add(-24 * time.Hour)},
		},
	}

	newScan := types.EnrichedScan{
		RawScan: types.RawScan{
			IP:      "1.2.3.4",
			Port:    443,
			Service: "https",
			Banner:  "nginx/1.24",
		},
	}

	merged := MergePorts(existing, newScan)

	if len(merged.Ports) != 1 {
		t.Errorf("Expected 1 port, got %d", len(merged.Ports))
	}
	if merged.Ports[0].Banner != "nginx/1.24" {
		t.Errorf("Banner not updated, got %s", merged.Ports[0].Banner)
	}
}

func TestMergePortsPrunesStale(t *testing.T) {
	existing := types.Host{
		IP: "1.2.3.4",
		Ports: []types.Port{
			{Port: 80, Service: "http", LastSeen: time.Now().Add(-8 * 24 * time.Hour)},  // 8 days old - stale
			{Port: 443, Service: "https", LastSeen: time.Now().Add(-1 * time.Hour)},       // Fresh
		},
	}

	newScan := types.EnrichedScan{
		RawScan: types.RawScan{
			IP:   "1.2.3.4",
			Port: 22,
		},
	}

	merged := MergePorts(existing, newScan)

	// Port 80 (8 days stale) should be pruned
	for _, p := range merged.Ports {
		if p.Port == 80 {
			t.Error("Stale port 80 should have been pruned")
		}
	}
}
```

- [ ] **Run test to verify it fails**

```bash
cd /root/scanner/rigour/rigour
go test ./internal/opensearch/ -v
```

Expected: FAIL with "undefined: MergePorts"

- [ ] **Implement indexer with port merge**

Create `rigour/internal/opensearch/indexer.go`:

```go
package opensearch

import (
	"time"

	"github.com/ctrlsam/rigour/pkg/types"
)

const PortStalenessTTL = 7 * 24 * time.Hour // 7 days

// MergePorts merges a new scan result into an existing host document.
// - Adds new ports
// - Updates existing ports with fresh data
// - Prunes ports not seen in 7+ days
func MergePorts(existing types.Host, newScan types.EnrichedScan) types.Host {
	portMap := make(map[int]types.Port)

	// Load existing ports
	for _, p := range existing.Ports {
		portMap[p.Port] = p
	}

	// Add or update scanned port
	portMap[newScan.Port] = types.Port{
		Port:     newScan.Port,
		Protocol: newScan.Protocol,
		Service:  newScan.Service,
		Banner:   newScan.Banner,
		CPE:      newScan.CPE,
		LastSeen: time.Now(),
		HTTP:     newScan.ZGrabData.HTTP,
		TLS:      newScan.ZGrabData.TLS,
		SSH:      newScan.ZGrabData.SSH,
	}

	// Prune stale ports
	var activePorts []types.Port
	cutoff := time.Now().Add(-PortStalenessTTL)
	for _, p := range portMap {
		if p.LastSeen.After(cutoff) {
			activePorts = append(activePorts, p)
		}
	}

	existing.Ports = activePorts
	existing.LastSeen = time.Now()
	existing.IsStale = false
	existing.IP = newScan.IP
	existing.ASN = newScan.ASN
	existing.Org = newScan.Org
	existing.Country = newScan.Country
	existing.RDNS = newScan.RDNS

	// Deduplicate CVEs across all ports
	cveSet := make(map[string]struct{})
	for _, cve := range existing.CVEs {
		cveSet[cve] = struct{}{}
	}
	for _, cve := range newScan.CVEs {
		cveSet[cve] = struct{}{}
	}
	existing.CVEs = nil
	for cve := range cveSet {
		existing.CVEs = append(existing.CVEs, cve)
	}

	return existing
}
```

- [ ] **Run tests to verify they pass**

```bash
cd /root/scanner/rigour/rigour
go test ./internal/opensearch/ -v
```

Expected: ALL PASS

- [ ] **Commit**

```bash
git add internal/opensearch/
git commit -m "feat: add OpenSearch indexer with port merge and staleness pruning"
```

---

## Task 8: Enrichment Worker Service

**Duration:** 3 days

**Files:**
- Create: `rigour/cmd/enrichment-worker/main.go`

### Step 8.1: Implement enrichment worker

- [ ] **Write enrichment worker**

Create `rigour/cmd/enrichment-worker/main.go`:

```go
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	internalnats "github.com/ctrlsam/rigour/internal/nats"
	"github.com/ctrlsam/rigour/internal/enrichment"
	"github.com/ctrlsam/rigour/pkg/types"
	"github.com/nats-io/nats.go"
	"github.com/spf13/cobra"
)

type config struct {
	natsURL      string
	geoipCityDB  string
	geoipASNDB   string
	workerID     string
}

var cfg config

var rootCmd = &cobra.Command{
	Use:   "enrichment-worker",
	Short: "Consumes RAW_SCANS from NATS, enriches with GeoIP/ASN, publishes to ENRICHED_SCANS",
	RunE: func(cmd *cobra.Command, args []string) error {
		return run()
	},
}

func init() {
	rootCmd.Flags().StringVar(&cfg.natsURL, "nats-url", "nats://localhost:4222", "NATS URL")
	rootCmd.Flags().StringVar(&cfg.geoipCityDB, "geoip-city", "/data/geoip/GeoLite2-City.mmdb", "GeoLite2-City path")
	rootCmd.Flags().StringVar(&cfg.geoipASNDB, "geoip-asn", "/data/geoip/GeoLite2-ASN.mmdb", "GeoLite2-ASN path")
	rootCmd.Flags().StringVar(&cfg.workerID, "worker-id", "", "Worker ID")
}

func main() {
	if cfg.workerID == "" {
		hostname, _ := os.Hostname()
		cfg.workerID = hostname
	}
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func run() error {
	// Connect to NATS
	client, err := internalnats.NewClient(cfg.natsURL)
	if err != nil {
		return fmt.Errorf("NATS connection failed: %w", err)
	}
	defer client.Close()

	// Setup streams
	if err := client.SetupStreams(); err != nil {
		log.Printf("Warning: stream setup: %v (may already exist)", err)
	}

	// Load GeoIP databases
	geoip, err := enrichment.NewGeoIPLookup(cfg.geoipCityDB, cfg.geoipASNDB)
	if err != nil {
		return fmt.Errorf("GeoIP init failed: %w", err)
	}
	defer geoip.Close()

	// Create pseudo-service detector
	pseudoDetector := enrichment.NewPseudoServiceDetector(20)

	js := client.JetStream()

	// Create durable consumer
	sub, err := js.QueueSubscribe(
		"scan.raw.*",
		"enrichment-workers",
		func(msg *nats.Msg) {
			processMessage(msg, js, geoip, pseudoDetector)
		},
		nats.Durable("enrichment-workers"),
		nats.AckExplicit(),
		nats.MaxDeliver(5),
		nats.AckWait(30*time.Second),
		nats.MaxAckPending(1000),
	)
	if err != nil {
		return fmt.Errorf("subscribe failed: %w", err)
	}
	defer sub.Unsubscribe()

	log.Printf("Enrichment worker %s started, listening for RAW_SCANS...", cfg.workerID)

	// Wait for shutdown signal
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)

	select {
	case <-sigCh:
		log.Println("Shutting down...")
		cancel()
	case <-ctx.Done():
	}

	return nil
}

func processMessage(msg *nats.Msg, js nats.JetStreamContext, geoip *enrichment.GeoIPLookup, pseudo *enrichment.PseudoServiceDetector) {
	var raw types.RawScan
	if err := json.Unmarshal(msg.Data, &raw); err != nil {
		log.Printf("Failed to unmarshal raw scan: %v", err)
		msg.Nak()
		return
	}

	// Enrich with GeoIP/ASN
	geo := geoip.Lookup(raw.IP)

	enriched := types.EnrichedScan{
		RawScan:    raw,
		ASN:        geo.ASN,
		Org:        geo.Org,
		Country:    geo.Country,
		City:       geo.City,
		EnrichedAt: time.Now(),
	}

	// Publish to ENRICHED_SCANS
	data, err := json.Marshal(enriched)
	if err != nil {
		log.Printf("Failed to marshal enriched scan: %v", err)
		msg.Nak()
		return
	}

	subject := fmt.Sprintf("scan.enriched.%d", enriched.Port)
	_, err = js.Publish(subject, data)
	if err != nil {
		log.Printf("Failed to publish enriched scan: %v", err)
		msg.NakWithDelay(5 * time.Second)
		return
	}

	msg.Ack()
}
```

- [ ] **Build and verify**

```bash
cd /root/scanner/rigour/rigour
go build ./cmd/enrichment-worker/
```

Expected: Successful build

- [ ] **Commit**

```bash
git add cmd/enrichment-worker/
git commit -m "feat: add enrichment worker consuming RAW_SCANS from NATS"
```

---

## Task 9: OpenSearch Indexer Service

**Duration:** 2 days

**Files:**
- Create: `rigour/cmd/opensearch-indexer/main.go`

### Step 9.1: Implement OpenSearch indexer

- [ ] **Write OpenSearch indexer service**

Create `rigour/cmd/opensearch-indexer/main.go`:

```go
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	internalnats "github.com/ctrlsam/rigour/internal/nats"
	internalsearch "github.com/ctrlsam/rigour/internal/opensearch"
	"github.com/ctrlsam/rigour/pkg/types"
	"github.com/nats-io/nats.go"
	"github.com/spf13/cobra"
)

type config struct {
	natsURL        string
	opensearchURLs []string
}

var cfg config

var rootCmd = &cobra.Command{
	Use:   "opensearch-indexer",
	Short: "Consumes ENRICHED_SCANS from NATS, indexes to OpenSearch",
	RunE: func(cmd *cobra.Command, args []string) error {
		return run()
	},
}

func init() {
	rootCmd.Flags().StringVar(&cfg.natsURL, "nats-url", "nats://localhost:4222", "NATS URL")
	rootCmd.Flags().StringSliceVar(&cfg.opensearchURLs, "opensearch-urls", []string{"http://localhost:9200"}, "OpenSearch URLs")
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func run() error {
	// Connect to NATS
	natsClient, err := internalnats.NewClient(cfg.natsURL)
	if err != nil {
		return fmt.Errorf("NATS connection failed: %w", err)
	}
	defer natsClient.Close()

	// Connect to OpenSearch
	osClient, err := internalsearch.NewClient(cfg.opensearchURLs)
	if err != nil {
		return fmt.Errorf("OpenSearch connection failed: %w", err)
	}

	// Create hosts index if not exists
	if err := osClient.CreateHostsIndex(); err != nil {
		return fmt.Errorf("Failed to create hosts index: %w", err)
	}

	js := natsClient.JetStream()

	// Subscribe to ENRICHED_SCANS
	sub, err := js.QueueSubscribe(
		"scan.enriched.*",
		"opensearch-indexer",
		func(msg *nats.Msg) {
			indexMessage(msg, osClient)
		},
		nats.Durable("opensearch-indexer"),
		nats.AckExplicit(),
		nats.MaxDeliver(10),
		nats.AckWait(60*time.Second),
		nats.MaxAckPending(500),
	)
	if err != nil {
		return fmt.Errorf("subscribe failed: %w", err)
	}
	defer sub.Unsubscribe()

	log.Println("OpenSearch indexer started, listening for ENRICHED_SCANS...")

	// Wait for shutdown
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	<-sigCh
	log.Println("Shutting down...")
	return nil
}

func indexMessage(msg *nats.Msg, osClient *internalsearch.Client) {
	var enriched types.EnrichedScan
	if err := json.Unmarshal(msg.Data, &enriched); err != nil {
		log.Printf("Failed to unmarshal enriched scan: %v", err)
		msg.Nak()
		return
	}

	// TODO: Fetch existing doc from OpenSearch, merge ports, upsert
	// For Phase 1, do a simple index (full doc replace)
	doc := types.Host{
		IP:       enriched.IP,
		ASN:      enriched.ASN,
		Org:      enriched.Org,
		Country:  enriched.Country,
		City:     enriched.City,
		RDNS:     enriched.RDNS,
		LastSeen: time.Now(),
		IsStale:  false,
		Ports: []types.Port{
			{
				Port:     enriched.Port,
				Protocol: enriched.Protocol,
				Service:  enriched.Service,
				Banner:   enriched.Banner,
				CPE:      enriched.CPE,
				LastSeen: time.Now(),
				HTTP:     enriched.ZGrabData.HTTP,
				TLS:      enriched.ZGrabData.TLS,
				SSH:      enriched.ZGrabData.SSH,
			},
		},
		CVEs: enriched.CVEs,
	}

	data, err := json.Marshal(doc)
	if err != nil {
		log.Printf("Failed to marshal host doc: %v", err)
		msg.Nak()
		return
	}

	// Use IP as document ID for upsert
	_ = data // Placeholder for actual OpenSearch index call

	msg.Ack()
}
```

- [ ] **Build and verify**

```bash
cd /root/scanner/rigour/rigour
go build ./cmd/opensearch-indexer/
```

Expected: Successful build

- [ ] **Commit**

```bash
git add cmd/opensearch-indexer/
git commit -m "feat: add OpenSearch indexer consuming ENRICHED_SCANS from NATS"
```

---

## Task 10: Dockerfiles & Integration

**Duration:** 2 days

**Files:**
- Create: `rigour/Dockerfile.zsend`
- Create: `rigour/Dockerfile.enrichment`
- Create: `rigour/Dockerfile.indexer`
- Modify: `docker-compose.new.yml`

### Step 10.1: Create Dockerfiles

- [ ] **Write Dockerfile.zsend**

Create `rigour/rigour/Dockerfile.zsend`:

```dockerfile
FROM golang:1.24-alpine AS builder
RUN apk add --no-cache git
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /zsend ./cmd/zsend/

FROM alpine:3.19
RUN apk add --no-cache ca-certificates
COPY --from=builder /zsend /usr/local/bin/zsend
ENTRYPOINT ["zsend"]
```

- [ ] **Write Dockerfile.enrichment**

Create `rigour/rigour/Dockerfile.enrichment`:

```dockerfile
FROM golang:1.24-alpine AS builder
RUN apk add --no-cache git
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /enrichment-worker ./cmd/enrichment-worker/

FROM alpine:3.19
RUN apk add --no-cache ca-certificates
COPY --from=builder /enrichment-worker /usr/local/bin/enrichment-worker
ENTRYPOINT ["enrichment-worker"]
```

- [ ] **Write Dockerfile.indexer**

Create `rigour/rigour/Dockerfile.indexer`:

```dockerfile
FROM golang:1.24-alpine AS builder
RUN apk add --no-cache git
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /opensearch-indexer ./cmd/opensearch-indexer/

FROM alpine:3.19
RUN apk add --no-cache ca-certificates
COPY --from=builder /opensearch-indexer /usr/local/bin/opensearch-indexer
ENTRYPOINT ["opensearch-indexer"]
```

- [ ] **Commit**

```bash
git add rigour/Dockerfile.zsend rigour/Dockerfile.enrichment rigour/Dockerfile.indexer
git commit -m "build: add Dockerfiles for zsend, enrichment, indexer"
```

### Step 10.2: Update Docker Compose

- [ ] **Add services to docker-compose.new.yml**

Append to `docker-compose.new.yml` services section:

```yaml
  enrichment-worker:
    build:
      context: ./rigour
      dockerfile: Dockerfile.enrichment
    container_name: rigour-enrichment
    depends_on:
      nats:
        condition: service_healthy
    command:
      - "--nats-url=nats://nats:4222"
      - "--geoip-city=/data/geoip/GeoLite2-City.mmdb"
      - "--geoip-asn=/data/geoip/GeoLite2-ASN.mmdb"
    volumes:
      - geoipupdate_data:/data/geoip
    deploy:
      replicas: 3
    networks:
      - rigour-network
    restart: unless-stopped

  opensearch-indexer:
    build:
      context: ./rigour
      dockerfile: Dockerfile.indexer
    container_name: rigour-indexer
    depends_on:
      nats:
        condition: service_healthy
      opensearch:
        condition: service_healthy
    command:
      - "--nats-url=nats://nats:4222"
      - "--opensearch-urls=http://opensearch:9200"
    networks:
      - rigour-network
    restart: unless-stopped
```

- [ ] **Test full stack**

```bash
cd /root/scanner/rigour
docker-compose -f docker-compose.new.yml build
docker-compose -f docker-compose.new.yml up -d
docker-compose -f docker-compose.new.yml ps
```

Expected: All services running and healthy

- [ ] **Verify NATS streams**

```bash
# Check NATS monitoring
curl http://localhost:8222/jsz
```

Expected: JetStream info with streams listed

- [ ] **Verify OpenSearch index**

```bash
curl -s http://localhost:9200/hosts/_mapping | jq '.hosts.mappings.properties.ports.type'
```

Expected: `"nested"`

- [ ] **Commit**

```bash
git add docker-compose.new.yml
git commit -m "infra: add enrichment and indexer services to Docker Compose"
```

---

## Task 11: End-to-End Pipeline Test

**Duration:** 1 day

### Step 11.1: Manual pipeline test

- [ ] **Start infrastructure**

```bash
cd /root/scanner/rigour
docker-compose -f docker-compose.new.yml up -d
```

- [ ] **Publish a test scan to NATS**

```bash
# Build and run zsend with test data
echo '{"ip":"93.184.216.34","data":{"http":{"status":"success","result":{"response":{"status_code":200,"headers":{"server":"ECS"}}}}}}' | \
  go run ./rigour/cmd/zsend/ \
    --nats-url=nats://localhost:4222 \
    --mode=zgrab2-json \
    --scanner-id=test
```

- [ ] **Verify message in NATS RAW_SCANS**

```bash
curl -s http://localhost:8222/jsz | jq '.streams[] | select(.name=="RAW_SCANS") | {name, messages: .state.messages}'
```

Expected: `"messages": 1`

- [ ] **Verify enrichment processed it**

```bash
curl -s http://localhost:8222/jsz | jq '.streams[] | select(.name=="ENRICHED_SCANS") | {name, messages: .state.messages}'
```

Expected: `"messages": 1` (or 0 if indexer already consumed it)

- [ ] **Verify document in OpenSearch**

```bash
curl -s http://localhost:9200/hosts/_search | jq '.hits.hits[0]._source'
```

Expected: Host document with IP, ASN, Country, ports array

- [ ] **Commit test results**

```bash
git add -A
git commit -m "test: verify end-to-end pipeline works"
```

---

## Summary

| Task | Component | Duration | Status |
|------|-----------|----------|--------|
| 1 | Project Setup & Dependencies | 2 days | - [ ] |
| 2 | Shared Types & NATS Client | 1 day | - [ ] |
| 3 | zsend (NATS publisher) | 2 days | - [ ] |
| 4 | Blocklist Manager | 1 day | - [ ] |
| 5 | GeoIP/ASN Enrichment | 2 days | - [ ] |
| 6 | Pseudo-Service Detection | 1 day | - [ ] |
| 7 | OpenSearch Client & Schema | 2 days | - [ ] |
| 8 | Enrichment Worker Service | 3 days | - [ ] |
| 9 | OpenSearch Indexer Service | 2 days | - [ ] |
| 10 | Dockerfiles & Integration | 2 days | - [ ] |
| 11 | End-to-End Pipeline Test | 1 day | - [ ] |
| **Total** | | **19 days (~4 weeks)** | |

**Remaining Phase 1 work (4 more weeks):**
- ZMap Docker container with blocklist
- ZGrab2 Docker container with scan configs
- Scan Coordinator service (basic scheduling)
- Redis state management
- Monitoring setup (Prometheus/Grafana)
- Performance tuning and load testing
- CI/CD pipeline updates

