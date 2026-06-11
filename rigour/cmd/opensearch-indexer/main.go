package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	internalnats "github.com/ctrlsam/rigour/internal/nats"
	internalsearch "github.com/ctrlsam/rigour/internal/opensearch"
	"github.com/ctrlsam/rigour/pkg/types"
	"github.com/nats-io/nats.go"
	opensearchapi "github.com/opensearch-project/opensearch-go/v2/opensearchapi"
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
		return fmt.Errorf("failed to create hosts index: %w", err)
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

	// Convert ZGrabData types to OpenSearch types
	var httpData *types.HTTPData
	if enriched.ZGrabData.HTTP != nil {
		httpData = &types.HTTPData{
			StatusCode: enriched.ZGrabData.HTTP.StatusCode,
			Title:      enriched.ZGrabData.HTTP.Title,
			Server:     enriched.ZGrabData.HTTP.Server,
		}
	}

	var tlsData *types.TLSData
	if enriched.ZGrabData.TLS != nil {
		var certData *types.CertData
		if enriched.ZGrabData.TLS.Cert != nil {
			certData = &types.CertData{
				SubjectCN:   enriched.ZGrabData.TLS.Cert.SubjectCN,
				IssuerCN:    enriched.ZGrabData.TLS.Cert.IssuerCN,
				Fingerprint: enriched.ZGrabData.TLS.Cert.Fingerprint,
				NotAfter:    enriched.ZGrabData.TLS.Cert.NotAfter,
				SAN:         enriched.ZGrabData.TLS.Cert.SAN,
			}
		}
		tlsData = &types.TLSData{
			Version: enriched.ZGrabData.TLS.Version,
			Cert:    certData,
		}
	}

	var sshData *types.SSHData
	if enriched.ZGrabData.SSH != nil {
		sshData = &types.SSHData{
			HASSH:    enriched.ZGrabData.SSH.HASSH,
			ServerID: enriched.ZGrabData.SSH.ServerID,
			KexAlgos: enriched.ZGrabData.SSH.KexAlgos,
		}
	}

	// For Phase 1, do a simple index (full doc replace)
	doc := types.HostDocument{
		IP:       enriched.IP,
		IPInt:    ipToInt(enriched.IP), // Use ipToInt helper
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
				HTTP:     httpData,
				TLS:      tlsData,
				SSH:      sshData,
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
	req := opensearchapi.IndexRequest{
		Index:      internalsearch.HostsIndex,
		DocumentID: doc.IP,
		Body:       bytes.NewReader(data),
		Refresh:    "true",
	}

	res, err := req.Do(context.Background(), osClient.Raw())
	if err != nil {
		log.Printf("Failed to index host document in OpenSearch: %v", err)
		msg.NakWithDelay(5 * time.Second)
		return
	}
	defer res.Body.Close()

	if res.IsError() {
		log.Printf("OpenSearch indexing returned error status: %s", res.Status())
		msg.NakWithDelay(5 * time.Second)
		return
	}

	msg.Ack()
}

func ipToInt(ip string) uint64 {
	parsedIP := net.ParseIP(ip)
	if parsedIP == nil {
		return 0
	}

	ipv4 := parsedIP.To4()
	if ipv4 == nil {
		return 0 // Only IPv4 supported
	}

	return uint64(ipv4[0])<<24 | uint64(ipv4[1])<<16 | uint64(ipv4[2])<<8 | uint64(ipv4[3])
}
