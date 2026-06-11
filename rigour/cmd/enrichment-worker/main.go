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
	geo, err := geoip.Lookup(raw.IP)
	if err != nil {
		log.Printf("GeoIP lookup failed: %v", err)
		// Don't fail the message, process without geo info
	}

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
