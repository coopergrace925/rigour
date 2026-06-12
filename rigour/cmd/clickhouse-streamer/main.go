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

	"github.com/ctrlsam/rigour/internal/clickhouse"
	internalnats "github.com/ctrlsam/rigour/internal/nats"
	"github.com/ctrlsam/rigour/pkg/types"
	"github.com/nats-io/nats.go"
	"github.com/spf13/cobra"
)

type config struct {
	natsURL        string
	clickhouseHost string
	clickhousePort int
	clickhouseDB   string
	clickhouseUser string
	clickhousePass string
	batchSize      int
	flushInterval  time.Duration
}

var cfg config

var rootCmd = &cobra.Command{
	Use:   "clickhouse-streamer",
	Short: "Streams enriched scan data from NATS to ClickHouse for analytics",
	RunE: func(cmd *cobra.Command, args []string) error {
		return run()
	},
}

func init() {
	rootCmd.Flags().StringVar(&cfg.natsURL, "nats-url", "nats://localhost:4222", "NATS URL")
	rootCmd.Flags().StringVar(&cfg.clickhouseHost, "clickhouse-host", "localhost", "ClickHouse host")
	rootCmd.Flags().IntVar(&cfg.clickhousePort, "clickhouse-port", 9000, "ClickHouse port")
	rootCmd.Flags().StringVar(&cfg.clickhouseDB, "clickhouse-db", "rigour_analytics", "ClickHouse database")
	rootCmd.Flags().StringVar(&cfg.clickhouseUser, "clickhouse-user", "default", "ClickHouse username")
	rootCmd.Flags().StringVar(&cfg.clickhousePass, "clickhouse-pass", "", "ClickHouse password")
	rootCmd.Flags().IntVar(&cfg.batchSize, "batch-size", 1000, "Batch size for ClickHouse inserts")
	rootCmd.Flags().DurationVar(&cfg.flushInterval, "flush-interval", 10*time.Second, "Flush interval for batched inserts")
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func run() error {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	// Connect to NATS
	natsClient, err := internalnats.NewClient(cfg.natsURL)
	if err != nil {
		return fmt.Errorf("NATS connection failed: %w", err)
	}
	defer natsClient.Close()

	js := natsClient.JetStream()

	// Connect to ClickHouse
	chClient, err := clickhouse.NewClient(clickhouse.Config{
		Host:     cfg.clickhouseHost,
		Port:     cfg.clickhousePort,
		Database: cfg.clickhouseDB,
		Username: cfg.clickhouseUser,
		Password: cfg.clickhousePass,
	})
	if err != nil {
		return fmt.Errorf("ClickHouse connection failed: %w", err)
	}
	defer chClient.Close()

	log.Printf("ClickHouse Streamer started")
	log.Printf("NATS: %s", cfg.natsURL)
	log.Printf("ClickHouse: %s:%d/%s", cfg.clickhouseHost, cfg.clickhousePort, cfg.clickhouseDB)
	log.Printf("Batch size: %d, Flush interval: %s", cfg.batchSize, cfg.flushInterval)

	// Create batch buffer
	batch := make([]types.EnrichedScan, 0, cfg.batchSize)
	batchMutex := make(chan struct{}, 1)
	batchMutex <- struct{}{}

	// Flush function
	flushBatch := func() error {
		<-batchMutex
		defer func() { batchMutex <- struct{}{} }()

		if len(batch) == 0 {
			return nil
		}

		log.Printf("Flushing batch of %d events to ClickHouse", len(batch))
		if err := chClient.BatchInsertScanEvents(ctx, batch); err != nil {
			log.Printf("Failed to insert batch: %v", err)
			return err
		}

		batch = batch[:0] // Clear batch
		return nil
	}

	// Periodic flush ticker
	flushTicker := time.NewTicker(cfg.flushInterval)
	defer flushTicker.Stop()

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case <-flushTicker.C:
				if err := flushBatch(); err != nil {
					log.Printf("Periodic flush failed: %v", err)
				}
			}
		}
	}()

	// Subscribe to enriched scans
	sub, err := js.QueueSubscribe(
		"scan.enriched.*",
		"clickhouse-streamers",
		func(msg *nats.Msg) {
			var scan types.EnrichedScan
			if err := json.Unmarshal(msg.Data, &scan); err != nil {
				log.Printf("Failed to unmarshal enriched scan: %v", err)
				msg.Nak()
				return
			}

			// Add to batch
			<-batchMutex
			batch = append(batch, scan)
			shouldFlush := len(batch) >= cfg.batchSize
			batchMutex <- struct{}{}

			// Flush if batch is full
			if shouldFlush {
				if err := flushBatch(); err != nil {
					log.Printf("Batch flush failed: %v", err)
					msg.Nak()
					return
				}
			}

			msg.Ack()
		},
		nats.Durable("clickhouse-streamers"),
		nats.AckExplicit(),
		nats.MaxDeliver(3),
		nats.AckWait(30*time.Second),
		nats.MaxAckPending(5000),
	)
	if err != nil {
		return fmt.Errorf("subscribe failed: %w", err)
	}
	defer sub.Unsubscribe()

	log.Println("Listening for enriched scans...")

	// Wait for shutdown
	<-ctx.Done()

	// Final flush
	log.Println("Shutting down, flushing remaining events...")
	if err := flushBatch(); err != nil {
		log.Printf("Final flush failed: %v", err)
	}

	log.Println("ClickHouse Streamer stopped")
	return nil
}
