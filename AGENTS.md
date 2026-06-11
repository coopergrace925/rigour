# AGENTS.md

High-signal guidance for working with the Rigour codebase.

## Project Structure

Rigour is a microservices-based IoT scanning tool with 3 Go services and 1 Next.js UI:

- **rigour/** (Go monorepo, Go 1.24.0)
  - `cmd/crawler/` - Network scanner (Naabu + Fingerprintx)
  - `cmd/persistence/` - Kafka consumer, enriches and stores to MongoDB
  - `cmd/api/` - REST API server (Chi router)
  - `pkg/` - Shared libraries
  - `internal/` - Internal shared code
  - `third_party/` - Vendored code (excluded from `go vet` in CI)

- **rigour-ui/** (Next.js 16.1.1, React 19, TypeScript)
  - Web interface for viewing scan results

## Running the System

**Required setup:**
1. Create `.env` from `.env.example` and set MaxMind credentials (required for GeoIP)
2. Set `CRAWLER_CIDR` (defaults to `0.0.0.0/0` - the ENTIRE internet, adjust carefully!)

**Start all services:**
```bash
docker compose up -d
```

**Access:**
- UI: http://localhost:3000
- API: http://localhost:8080
- Kafka: localhost:29092 (external), kafka:9092 (internal)
- MongoDB: localhost:27017

**Stop:**
```bash
docker compose down
```

## Development Commands

### Go services (run from `rigour/` directory)

**Install libpcap-dev first (required dependency):**
```bash
sudo apt-get update && sudo apt-get install -y libpcap-dev
```

**Test everything:**
```bash
go test -v -race -coverprofile=coverage.out -covermode=atomic ./...
```

**Lint (CI order matters):**
```bash
go mod verify
go mod download
go vet ./cmd/... ./internal/... ./pkg/...  # Excludes third_party/
gofmt -l .  # Should return empty
```

**Format:**
```bash
gofmt -w .
```

### Next.js UI (run from `rigour-ui/` directory)

**Dev server:**
```bash
npm run dev  # or yarn/pnpm/bun dev
```

**Build:**
```bash
npm run build
npm start
```

**Lint:**
```bash
npm run lint
```

## Architecture Notes

**Message flow:** Crawler → Kafka (`scanned_services` topic) → Persistence → MongoDB → API → UI

**Crawler:** Streams results to stdout (NDJSON) AND Kafka. Can run standalone without Kafka by omitting `--kafka-brokers`.

**Persistence:** Enriches scan data with GeoIP (MaxMind) before storing. Requires `/data/geoip` volume mount.

**API:** MongoDB query endpoint at `/api/hosts/search` with pagination. Filter param accepts MongoDB query JSON.

**Services are independent:** Each service has its own `main.go` and Dockerfile. No shared runtime state beyond Kafka/MongoDB.

## Testing Quirks

- `third_party/` directory contains code with unsafe `testing.T/B` calls from goroutines - excluded from `go vet` in CI
- Tests use `dockertest` for integration tests with real MongoDB/Kafka instances
- Race detector enabled in CI (`-race` flag)
- Coverage reports uploaded to Codecov but `fail_ci_if_error: false`

## CI Workflow

From `.github/workflows/ci.yml`:

1. Go 1.25.5 (note: go.mod says 1.24.0, CI uses 1.25.5)
2. Install libpcap-dev
3. `go mod verify && go mod download`
4. `go vet` (cmd, internal, pkg only)
5. `gofmt -l` check (must be empty)
6. `go test -v -race -coverprofile=coverage.out -covermode=atomic ./...`
7. Build Docker images (if tests pass)

Triggers on push/PR to `main` or `develop` branches, only when `rigour/**` files change.

## Important Flags & Defaults

**Crawler:**
- `--top-ports 1000` - Scans top 1000 ports (can use `full` for all ports)
- `--fast` - Only scan default ports per service
- `--scan-type c` - Connect scan (default), use `s` for SYN scan
- `--rate 50000` - Packets per second
- UDP scanning (`-U`) may require root on Linux/macOS

**Persistence:**
- `--geoip-path` - Must point to GeoLite2 database directory

**API:**
- `--addr :8080` - Server listen address
- `--mongo-collection hosts` - Default collection name

## Common Gotchas

- **CIDR default is 0.0.0.0/0** - Change this before running!
- **MaxMind account required** - GeoIP won't work without credentials in `.env`
- **Port discovery performance** - Linux performs better than macOS (especially Apple Silicon)
- **UDP scanning** - Requires elevated privileges on most systems
- **Third-party code** - Don't try to fix linting in `third_party/`, it's vendored
- **Go version mismatch** - go.mod says 1.24.0, CI uses 1.25.5

## API Reference

**GET /api/hosts/search**
- `filter` (optional): MongoDB query JSON
- `limit` (optional): Max results (50-500, default 50)
- `page_token` (optional): Pagination token from `next_page_token`

**GET /api/facets**
- `filter` (optional): MongoDB query JSON to restrict aggregation

## Key Dependencies

- **Naabu** (projectdiscovery) - Port discovery
- **Fingerprintx** (praetorian) - Service fingerprinting
- **Kafka** (Confluent) - Message bus
- **MongoDB** - Data storage
- **Chi** - HTTP router
- **Cobra** - CLI framework
- **MaxMind GeoIP** - Location enrichment
