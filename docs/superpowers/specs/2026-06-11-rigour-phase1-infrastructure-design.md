# Design Specification: Phase 1 Infrastructure & Coordination

**Date:** 2026-06-11  
**Version:** 1.0  
**Status:** Approved for Implementation

---

## 1. System Architecture Details

The scanning layer is fully decoupled from scheduling. The **Scan Coordinator** is responsible only for orchestrating the sweep schedule, keeping track of blocklists/opt-outs, and emitting tasks to NATS. The **Scanner Agents** are stateless workers that consume tasks, run local ZMap and ZGrab2 sweeps inside Docker containers, and publish results to the NATS `RAW_SCANS` stream.

```
┌────────────────────────────────────────┐
│           SCAN COORDINATOR             │
│ - Reads Scheduler Config               │
│ - Evaluates CIDRs against Blocklist    │
│ - Splits target range into /24 blocks  │
│ - Enforces ASN rate limits via Redis   │
└───────────────────┬────────────────────┘
                    │
                    ▼  (scan.task.<port>)
┌────────────────────────────────────────┐
│             NATS JetStream             │
│  - Stream: SCAN_TASKS                  │
│  - Policy: WorkQueuePolicy (durable)   │
└───────────────────┬────────────────────┘
                    │
                    ▼  (Load-balanced pull)
┌────────────────────────────────────────┐
│            SCANNER AGENT               │
│  - Runs inside Docker (NET_RAW cap)    │
│  - Subprocesses: ZMap -> ZGrab2        │
│  - Reads output and publishes to raw   │
└───────────────────┬────────────────────┘
                    │
                    ▼  (scan.raw.<port>)
┌────────────────────────────────────────┐
│        NATS Stream: RAW_SCANS          │
└────────────────────────────────────────┘
```

---

## 2. Redis State Store & Schema

Redis provides fast, shared tracking of state across the coordinator and scanner nodes.

### 2.1 ASN Rate Limiting
To prevent abuse, we implement a distributed token bucket per autonomous system:
* **Key:** `rate:asn:<asn_id>` (String)
* **Value:** Packet count consumed.
* **TTL:** 1s (resets window).
* **Threshold:** Configurable max packets/sec per ASN.

### 2.2 Target Subnet Locking
To prevent multiple scanner agents from sweep-scanning the same `/24` concurrently:
* **Key:** `scan:lock:<cidr_hash>` (String)
* **Value:** `<agent_id>`
* **TTL:** 1h (auto-released on agent failure).

### 2.3 Opt-Out Registry
* **Key:** `blocklist:optout` (Set)
* **Value:** Set of individual IP addresses or CIDR blocks.
* **Lookup Complexity:** O(1) for IPs.

---

## 3. Scan Coordinator (Go Service)

The coordinator is a long-running Go service managing the scanning cycle.

```go
type ScanCoordinator struct {
    redisClient *redis.Client
    natsConn    *nats.Conn
    blocklist   *blocklist.Blocklist
    scheduler   *PortScheduler
}
```

### 3.1 Port Scheduler Config
```go
type PortSchedule struct {
    Interval time.Duration
    Ports    []int
}
```
Default schedules:
- **Critical Ports** (22, 23, 3389): Every 6h.
- **High-Priority Web** (80, 443): Every 24h.
- **SCADA/ICS Ports** (102, 502, 47808): Daily.
- **Top 1000 Ports**: Weekly.
- **Full 65K Range**: Monthly.

### 3.2 Coordination Flow
1. Periodic tick triggers scan for a schedule category.
2. Select target subnets based on `CRAWLER_CIDR` setting.
3. Split the target network into `/24` subnets (256 hosts).
4. Filter out subnets containing private, DoD, or opt-out IPs using the `Blocklist` manager.
5. Publish a `ScanTask` message for each target subnet to NATS JetStream:
   - **Stream:** `SCAN_TASKS`
   - **Subject:** `scan.task.<port>`
   - **Retention Policy:** `WorkQueuePolicy`

```go
type ScanTask struct {
    TaskID    string    `json:"task_id"`
    CIDR      string    `json:"cidr"`
    Port      int       `json:"port"`
    Protocol  string    `json:"protocol"`
    Bandwidth string    `json:"bandwidth"`
    CreatedAt time.Time `json:"created_at"`
}
```

---

## 4. Scanner Agent & Docker Integration

The scanner agent is a lightweight Go wrapper packaged inside a Docker container.

### 4.1 Execution Flow
1. Subscribes to NATS `scan.task.*` using durable worker queue group `scanner-agents`.
2. Receives a task and executes **ZMap** as a subprocess:
   ```bash
   zmap --probe-module=tcp_synscan --target-port=<port> --bandwidth=<bandwidth> --target-ips=<cidr> --output-fields=saddr
   ```
3. Parses ZMap stdout in real time, validating active IPs.
4. Active IPs are directly piped to **ZGrab2** stdin:
   ```bash
   zgrab2 multiple --config=/etc/zgrab2/protocols.ini
   ```
5. Reads ZGrab2 JSON output line-by-line.
6. Publishes parsed `RawScan` messages to NATS `scan.raw.<port>`.
7. Acks the task back to NATS on successful completion of the sweep.

### 4.2 Docker Configuration (`Dockerfile.scanner`)
Requires root capability `cap_add: [NET_RAW]` to perform SYN sweeps.
Uses compiled `zmap` and `zgrab2` binaries from source.
Loads a local blocklist configuration file.

---

## 5. Test Plan

1. **Unit Tests**:
   - Verify Port Scheduler schedules categories at correct intervals.
   - Verify Redis target locking prevents overlapping subnets.
   - Verify Go wrapper correctly parses ZMap outputs and passes them to ZGrab2.
2. **Integration Tests**:
   - Spin up local Redis + NATS in Compose.
   - Send mock scan task, confirm Scanner Agent executes mock ZMap/ZGrab2 binaries, and publishes raw scan records back to NATS.
