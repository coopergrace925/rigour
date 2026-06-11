# AGENTS.md

High-signal guidance for working with the Rigour codebase.

## Project Structure

Rigour is migrating from an MVP stack (Kafka + MongoDB + Naabu/Fingerprintx) to a Censys-competitor architecture (NATS JetStream + OpenSearch + Redis + ZMap/ZGrab2).

### Go Services & Layout (`rigour/` Go monorepo, Go 1.25.0)

**New Pipeline Services (Phase 1):**
- `cmd/zsend/` - CLI tool to stream ZMap (CSV) / ZGrab2 (NDJSON) output to NATS `scan.raw.<port>`
- `cmd/enrichment-worker/` - Queue consumer (`enrichment-workers`) that enriches scans with GeoIP/ASN and publishes to `scan.enriched.<port>`
- `cmd/opensearch-indexer/` - Queue consumer (`opensearch-indexer`) that indexes/upserts hosts into OpenSearch

**Shared Logic (`internal/`):**
- `internal/blocklist/` - Thread-safe RFC1918, DoD, and IANA reserved IP blocklist and opt-out manager
- `internal/enrichment/` - GeoIP lookup (MaxMind GeoLite2) and Pseudo-service / Honeypot detection
- `internal/opensearch/` - OpenSearch client, Hosts index creator, and Censys-style 7-day port staleness pruner / merge utility
- `internal/nats/` - NATS JetStream client and idempotent stream configuration (`RAW_SCANS`, `ENRICHED_SCANS`, `SCAN_EVENTS`)

**MVP Legacy Services:**
- `cmd/crawler/` - Network scanner (Naabu + Fingerprintx)
- `cmd/persistence/` - Kafka consumer, enriches and stores to MongoDB
- `cmd/api/` - REST API server (Chi router querying MongoDB)

**Frontend:**
- `rigour-ui/` - Next.js UI for viewing scan results

---

## Infrastructure Stack

- **New Stack (NATS JetStream + OpenSearch + Redis)**: Managed via `docker-compose.new.yml`
- **Legacy Stack (Kafka + ZooKeeper + MongoDB + GeoIPUpdate)**: Managed via `docker-compose.yml` and `docker-compose.override.yml`

---

## Development & Test Commands

Run Go commands from the `rigour/` directory.

### Build New Pipeline Services
```bash
go build ./cmd/zsend/
go build ./cmd/enrichment-worker/
go build ./cmd/opensearch-indexer/
```

### Run Pipeline Unit Tests (Fast & Self-Contained)
Unit tests for the new architecture do not require NATS, OpenSearch, or Docker to be running:
```bash
go test -v ./cmd/zsend ./internal/blocklist ./internal/enrichment ./internal/opensearch
```

### Run Entire Test Suite (Requires Host libpcap-dev and Docker daemon)
Legacy crawler tests use `dockertest` and will fail if Docker is not running.
```bash
# Setup dependency
sudo apt-get update && sudo apt-get install -y libpcap-dev
# Run tests
go test -v -race ./...
```

---

## Crucial Architectural Constraints

1. **NO MongoDB in New Stack**: OpenSearch is the single source of truth for host and port state.
2. **Nested Mapping**: OpenSearch `ports` schema must be `"type": "nested"` to prevent cross-port query contamination.
3. **OpenSearch Upsert**: Index documents using the host IP address as the document ID.
4. **Idempotency**: NATS stream creation (`SetupStreams`) and OpenSearch index creation (`CreateHostsIndex`) check for existing resources first to prevent errors on restart.
5. **Staleness Pruning**: `MergePorts` implements a 7-day TTL cutoff to prune ports not scanned recently.
6. **Go 1.25.0**: Monorepo uses Go 1.25.0 due to NATS dependencies.

---

## Environment & Run Gotchas

- **MaxMind Databases**: The enrichment worker requires GeoLite2-City and GeoLite2-ASN databases to start up. If running locally without Docker volume `geoipupdate_data`, provide paths using `--geoip-city` and `--geoip-asn`.
- **IP Validation**: `zsend` validates that incoming IPs are well-formed IPv4 addresses to prevent stream pollution.
- **Port Extraction**: ZGrab2 inputs parsed by `zsend` must have their port extracted and populated, otherwise routing to `scan.raw.<port>` fails.
- **Nil IP Checks**: Ensure blocklist and GeoIP lookups handle `nil` IP inputs gracefully to avoid panics.
