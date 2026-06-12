---

## NATS Streams & Subjects

**Streams (JetStream):**
- `RAW_SCANS` - Raw ZMap+ZGrab2 output, 48h retention, WorkQueue
- `ENRICHED_SCANS` - GeoIP/rDNS/CPE/CVE enriched, 24h retention, WorkQueue
- `SCAN_EVENTS` - Audit/alert events, 7d retention, Limits
- `SCAN_TASKS` - Coordinator → scanner-agent task queue, 24h retention

**Subject patterns:**
- `scan.raw.<port>` - Raw scans per port (e.g., `scan.raw.443`)
- `scan.enriched.<port>` - Enriched scans per port
- `scan.task.<port>` - Scan tasks per port
- `scan.event.*` - Audit/alert events

**Queue groups:**
- `enrichment-workers` - Enrichment worker consumers
- `opensearch-indexer` - Indexer consumers
- `scanner-agents` - Scanner agent consumers

---

## OpenSearch Schema

**Index:** `hosts`

**Key mappings:**
```json
{
  "ip": { "type": "ip" },
  "ports": { 
    "type": "nested",  // CRITICAL - prevents cross-port contamination
    "properties": {
      "port": { "type": "integer" },
      "protocol": { "type": "keyword" },
      "service": { "type": "keyword" },
      "banner": { "type": "text" },
      "last_seen": { "type": "date" }
    }
  },
  "asn": { "type": "integer" },
  "country": { "type": "keyword" },
  "cves": {
    "type": "nested",
    "properties": {
      "id": { "type": "keyword" },
      "verified": { "type": "boolean" }
    }
  }
}
```

**ILM Policy:** hot (1d/30GB) → warm (30d) → delete

---

## API Query Syntax (Shodan-Compatible)

**Shodan dork parser** in `internal/api/parser.go`:

```
port:22                    # Port filter
country:US                 # Country code
asn:AS15169               # ASN filter
org:"Google LLC"          # Organization
cve:CVE-2023-12345        # CVE filter
verified:true             # Verified CVEs only
banner:"OpenSSH"          # Banner text search
title:"Welcome"           # HTTP title
server:nginx              # HTTP server header
product:apache            # Product name
```

**API endpoint:** `GET /api/hosts/search?q=<query>&limit=<N>`

---

## Breaking Changes (2026-06-12)

Recent world-class implementation introduced breaking changes:

1. **Port coverage:** Changed from "all 65,535 ports" to "3,847 Shodan ports"
2. **CVE type:** `[]string` → `[]CVEInfo` (with verified flag)
3. **RawScan fields:** Added `BannerID`, `ParentID`, `ScannerRegion`
4. **CertInfo fields:** Added `Hostnames` array for easy querying

**If working with old code:** Check git history around commit `2949dbc` (2026-06-12).

---

## Common Mistakes to Avoid

1. **Don't add MongoDB** - OpenSearch is the single database
2. **Don't use `[]string` for CVEs** - Use `[]CVEInfo` with verified flag
3. **Don't hardcode port lists** - Use port registry from `internal/coordinator/ports.go`
4. **Don't scan all 65,535 ports** - Use Shodan's 3,847 port list
5. **Don't forget nested type** - OpenSearch ports MUST be `"type": "nested"`
6. **Don't skip banner IDs** - All scans need `BannerID` for deduplication
7. **Don't disable SNI/Host headers** - Required for cloud/CDN detection

---

## Reference Documents

- **Design Spec:** `docs/superpowers/specs/2026-06-11-rigour-censys-competitor-design.md`
- **Implementation Summary:** `docs/superpowers/analysis/2026-06-12-world-class-implementation-summary.md`
- **Phase Plans:** `docs/superpowers/plans/2026-06-11-phase*.md`

**For architecture questions:** Read the design spec first. For implementation status: Read the world-class summary.