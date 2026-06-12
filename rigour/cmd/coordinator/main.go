package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ctrlsam/rigour/internal/blocklist"
	"github.com/ctrlsam/rigour/internal/coordinator"
	internalnats "github.com/ctrlsam/rigour/internal/nats"
	"github.com/ctrlsam/rigour/internal/redis"
	"github.com/ctrlsam/rigour/pkg/types"
	"github.com/nats-io/nats.go"
	"github.com/spf13/cobra"
)

type config struct {
	natsURL      string
	redisAddr    string
	crawlerCIDR  string
	asnRateLimit int
	continuous   bool
}

var cfg config

var rootCmd = &cobra.Command{
	Use:   "coordinator",
	Short: "Automated scan coordinator with port-priority scheduling and ASN rate limiting",
	RunE: func(cmd *cobra.Command, args []string) error {
		return run()
	},
}

func init() {
	rootCmd.Flags().StringVar(&cfg.natsURL, "nats-url", "nats://localhost:4222", "NATS URL")
	rootCmd.Flags().StringVar(&cfg.redisAddr, "redis-addr", "localhost:6379", "Redis address")
	rootCmd.Flags().StringVar(&cfg.crawlerCIDR, "cidr", "0.0.0.0/0", "Target CIDR block to scan")
	rootCmd.Flags().IntVar(&cfg.asnRateLimit, "asn-rate-limit", 100, "Max scans per minute per ASN")
	rootCmd.Flags().BoolVar(&cfg.continuous, "continuous", true, "Run continuously with scheduling")
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
		return fmt.Errorf("NATS failed: %w", err)
	}
	defer natsClient.Close()

	js := natsClient.JetStream()

	// Create tasks stream if not exists
	_, err = js.AddStream(&nats.StreamConfig{
		Name:      "SCAN_TASKS",
		Subjects:  []string{"scan.task.*"},
		Retention: nats.WorkQueuePolicy,
		MaxAge:    24 * time.Hour,
		Storage:   nats.FileStorage,
		Replicas:  1,
	})
	if err != nil {
		log.Printf("SCAN_TASKS stream setup: %v (may already exist)", err)
	}

	// Connect to Redis
	redisClient, err := redis.NewClient(cfg.redisAddr)
	if err != nil {
		return fmt.Errorf("Redis connection failed: %w", err)
	}
	defer redisClient.Close()

	// Initialize components
	bl := blocklist.NewBlocklist()
	scheduler := coordinator.NewScheduler()
	rateLimiter := coordinator.NewASNRateLimiter(redisClient, cfg.asnRateLimit)

	log.Printf("Coordinator started")
	log.Printf("Target CIDR: %s", cfg.crawlerCIDR)
	log.Printf("ASN rate limit: %d scans/min", cfg.asnRateLimit)
	log.Printf("Continuous mode: %v", cfg.continuous)
	log.Printf("Schedule summary: %+v", scheduler.GetScheduleSummary())

	if cfg.continuous {
		return runContinuous(ctx, js, bl, scheduler, rateLimiter)
	}

	return runOnce(ctx, js, bl, scheduler, rateLimiter)
}

func runContinuous(ctx context.Context, js nats.JetStreamContext, bl *blocklist.Blocklist, scheduler *coordinator.Scheduler, rateLimiter *coordinator.ASNRateLimiter) error {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	// Run immediately on start
	if err := scanDuePorts(ctx, js, bl, scheduler, rateLimiter); err != nil {
		log.Printf("Initial scan failed: %v", err)
	}

	for {
		select {
		case <-ctx.Done():
			log.Println("Coordinator shutting down...")
			return nil
		case <-ticker.C:
			if err := scanDuePorts(ctx, js, bl, scheduler, rateLimiter); err != nil {
				log.Printf("Scan cycle failed: %v", err)
			}
		}
	}
}

func runOnce(ctx context.Context, js nats.JetStreamContext, bl *blocklist.Blocklist, scheduler *coordinator.Scheduler, rateLimiter *coordinator.ASNRateLimiter) error {
	return scanDuePorts(ctx, js, bl, scheduler, rateLimiter)
}

func scanDuePorts(ctx context.Context, js nats.JetStreamContext, bl *blocklist.Blocklist, scheduler *coordinator.Scheduler, rateLimiter *coordinator.ASNRateLimiter) error {
	duePorts := scheduler.GetDuePorts(ctx)
	if len(duePorts) == 0 {
		nextScan := scheduler.GetNextScanTime()
		log.Printf("No ports due for scanning. Next scan: %s", nextScan.Format(time.RFC3339))
		return nil
	}

	log.Printf("Scanning %d due ports: %v", len(duePorts), duePorts)

	subnets, err := coordinator.SplitInto24(cfg.crawlerCIDR)
	if err != nil {
		return fmt.Errorf("CIDR split failed: %w", err)
	}

	tasksPublished := 0
	tasksRateLimited := 0

	for _, port := range duePorts {
		for _, subnet := range subnets {
			_, ipnet, err := net.ParseCIDR(subnet)
			if err != nil || bl.IsBlocked(ipnet.IP) {
				continue
			}

			// ASN rate limiting
			asn, err := rateLimiter.GetASNForIP(ipnet.IP.String())
			if err != nil {
				log.Printf("ASN lookup failed for %s: %v", ipnet.IP, err)
				continue
			}

			allowed, err := rateLimiter.CheckAndIncrement(ctx, asn)
			if err != nil {
				log.Printf("Rate limit check failed for ASN %d: %v", asn, err)
				continue
			}
			if !allowed {
				tasksRateLimited++
				continue
			}

			task := types.RawScan{
				IP:        subnet,
				Port:      port,
				Protocol:  "tcp",
				ScannerID: "coordinator",
				ScannedAt: time.Now(),
			}

			data, err := json.Marshal(task)
			if err != nil {
				continue
			}

			subject := fmt.Sprintf("scan.task.%d", port)
			_, err = js.Publish(subject, data)
			if err != nil {
				log.Printf("Failed to publish task for %s:%d: %v", subnet, port, err)
				continue
			}

			tasksPublished++
		}

		// Mark port as scanned
		scheduler.MarkScanned(port)
	}

	log.Printf("Scan cycle complete: %d tasks published, %d rate-limited", tasksPublished, tasksRateLimited)
	return nil
}
