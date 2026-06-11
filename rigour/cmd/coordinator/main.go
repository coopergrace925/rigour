package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
	"os"
	"time"

	"github.com/ctrlsam/rigour/internal/blocklist"
	"github.com/ctrlsam/rigour/internal/coordinator"
	internalnats "github.com/ctrlsam/rigour/internal/nats"
	"github.com/ctrlsam/rigour/pkg/types"
	"github.com/nats-io/nats.go"
	"github.com/spf13/cobra"
)

type config struct {
	natsURL     string
	crawlerCIDR string
	ports       []int
}

var cfg config

var rootCmd = &cobra.Command{
	Use:   "coordinator",
	Short: "Emits ScanTasks to NATS JetStream",
	RunE: func(cmd *cobra.Command, args []string) error {
		return run()
	},
}

func init() {
	rootCmd.Flags().StringVar(&cfg.natsURL, "nats-url", "nats://localhost:4222", "NATS URL")
	rootCmd.Flags().StringVar(&cfg.crawlerCIDR, "cidr", "192.168.0.0/22", "Target CIDR block to scan")
	rootCmd.Flags().IntSliceVar(&cfg.ports, "ports", []int{80, 443}, "Ports to scan")
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func run() error {
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
		log.Printf("SCAN_TASKS stream already exists or failed: %v", err)
	}

	bl := blocklist.NewBlocklist()
	subnets, err := coordinator.SplitInto24(cfg.crawlerCIDR)
	if err != nil {
		return err
	}

	log.Printf("Coordinator started. Split %s into %d tasks.", cfg.crawlerCIDR, len(subnets))

	// Publish tasks for each port and subnet
	for _, port := range cfg.ports {
		for _, subnet := range subnets {
			// Basic filtering
			_, ipnet, err := net.ParseCIDR(subnet)
			if err != nil || bl.IsBlocked(ipnet.IP) {
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
			}
		}
	}

	log.Println("All scan tasks published successfully. Exiting.")
	return nil
}
