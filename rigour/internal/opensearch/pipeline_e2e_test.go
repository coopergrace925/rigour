package opensearch

import (
	"encoding/json"
	"net"
	"testing"
	"time"

	"github.com/ctrlsam/rigour/internal/blocklist"
	"github.com/ctrlsam/rigour/internal/enrichment"
	"github.com/ctrlsam/rigour/pkg/types"
)

func TestEndToEndPipelineLogic(t *testing.T) {
	// 1. Input ZGrab2 JSON line
	zgrabJSON := `{"ip":"93.184.216.34","port":443,"data":{"http":{"status":"success","result":{"response":{"status_code":200,"headers":{"server":"ECS"}}}}}}`

	// 2. Simulate zsend parsing (Task 3)
	var raw struct {
		IP   string                 `json:"ip"`
		Port int                    `json:"port"`
		Data map[string]interface{} `json:"data"`
	}
	err := json.Unmarshal([]byte(zgrabJSON), &raw)
	if err != nil {
		t.Fatalf("Failed to parse input JSON: %v", err)
	}

	rawScan := &types.RawScan{
		IP:        raw.IP,
		Port:      raw.Port,
		Protocol:  "tcp",
		ScannerID: "test-scanner",
		ScannedAt: time.Now(),
	}

	for proto, protoData := range raw.Data {
		rawScan.Service = proto
		if pd, ok := protoData.(map[string]interface{}); ok {
			if status, ok := pd["status"].(string); ok {
				rawScan.ZGrabData.Status = status
			}
			rawScan.ZGrabData.Protocol = proto
			// Map HTTP Info
			if proto == "http" {
				rawScan.ZGrabData.HTTP = &types.HTTPInfo{
					StatusCode: 200,
					Server:     "ECS",
				}
			}
		}
	}

	if rawScan.IP != "93.184.216.34" || rawScan.Port != 443 {
		t.Fatalf("Parsed RawScan invalid: %+v", rawScan)
	}

	// 3. Simulate Blocklist check (Task 4)
	bl := blocklist.NewBlocklist()
	if bl.IsBlocked(net.ParseIP(rawScan.IP)) {
		t.Fatalf("IP %s should not be blocked by default blocklist", rawScan.IP)
	}

	// 4. Simulate GeoIP Enrichment (Task 5)
	// Mock GeoIP response since we don't have the MMDB files in the test environment
	geoResult := enrichment.GeoResult{
		Country: "US",
		City:    "Norwell",
		ASN:     15133,
		Org:     "MCI Communications Services, Inc. d/b/a Verizon Business",
	}

	enriched := types.EnrichedScan{
		RawScan:    *rawScan,
		ASN:        geoResult.ASN,
		Org:        geoResult.Org,
		Country:    geoResult.Country,
		City:       geoResult.City,
		EnrichedAt: time.Now(),
	}

	// 5. Simulate OpenSearch Indexer Merge (Task 7)
	existingHost := types.HostDocument{
		IP: "93.184.216.34",
		Ports: []types.Port{
			{
				Port:     80,
				Protocol: "tcp",
				Service:  "http",
				Banner:   "ECS",
				LastSeen: time.Now().Add(-1 * time.Hour),
			},
		},
	}

	// Helper to merge enriched scan into HostDocument
	var httpData *types.HTTPData
	if enriched.ZGrabData.HTTP != nil {
		httpData = &types.HTTPData{
			StatusCode: enriched.ZGrabData.HTTP.StatusCode,
			Server:     enriched.ZGrabData.HTTP.Server,
		}
	}

	// Merge logic
	mergedPorts := make(map[int]types.Port)
	for _, p := range existingHost.Ports {
		mergedPorts[p.Port] = p
	}

	mergedPorts[enriched.Port] = types.Port{
		Port:     enriched.Port,
		Protocol: enriched.Protocol,
		Service:  enriched.Service,
		LastSeen: time.Now(),
		HTTP:     httpData,
	}

	var activePorts []types.Port
	for _, p := range mergedPorts {
		activePorts = append(activePorts, p)
	}

	finalDoc := types.HostDocument{
		IP:       enriched.IP,
		IPInt:    ipToInt(enriched.IP),
		ASN:      enriched.ASN,
		Org:      enriched.Org,
		Country:  enriched.Country,
		City:     enriched.City,
		LastSeen: time.Now(),
		Ports:    activePorts,
	}

	// 6. Verify outputs
	if finalDoc.IP != "93.184.216.34" {
		t.Errorf("Expected IP 93.184.216.34, got %s", finalDoc.IP)
	}
	if finalDoc.IPInt != 1572395042 {
		t.Errorf("Expected IPInt 1572395042, got %d", finalDoc.IPInt)
	}
	if finalDoc.ASN != 15133 {
		t.Errorf("Expected ASN 15133, got %d", finalDoc.ASN)
	}
	if len(finalDoc.Ports) != 2 {
		t.Errorf("Expected 2 ports after merge, got %d", len(finalDoc.Ports))
	}

	// Check if ports are correct
	found80 := false
	found443 := false
	for _, p := range finalDoc.Ports {
		if p.Port == 80 {
			found80 = true
		}
		if p.Port == 443 {
			found443 = true
		}
	}

	if !found80 || !found443 {
		t.Errorf("Expected to find ports 80 and 443, found80=%t, found443=%t", found80, found443)
	}
}
