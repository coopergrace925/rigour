package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"syscall"
	"time"

	internalnats "github.com/ctrlsam/rigour/internal/nats"
	"github.com/ctrlsam/rigour/pkg/types"
	"github.com/nats-io/nats.go"
	"github.com/spf13/cobra"
)

type config struct {
	natsURL  string
	agentID  string
	zmapPath string
}

var cfg config

var rootCmd = &cobra.Command{
	Use:   "scanner-agent",
	Short: "Consumes scan tasks, executes ZMap + ZGrab2 sweeps, and publishes raw scans",
	RunE: func(cmd *cobra.Command, args []string) error {
		return run()
	},
}

func init() {
	rootCmd.Flags().StringVar(&cfg.natsURL, "nats-url", "nats://localhost:4222", "NATS URL")
	rootCmd.Flags().StringVar(&cfg.agentID, "agent-id", "", "Scanner Agent ID")
	rootCmd.Flags().StringVar(&cfg.zmapPath, "zmap-path", "/usr/local/bin/zmap", "Path to ZMap binary")
}

func main() {
	if cfg.agentID == "" {
		cfg.agentID, _ = os.Hostname()
	}
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

	// Subscribe to tasks
	sub, err := js.QueueSubscribe(
		"scan.task.*",
		"scanner-agents",
		func(msg *nats.Msg) {
			processTask(msg, js)
		},
		nats.Durable("scanner-agent-worker"),
		nats.AckExplicit(),
		nats.MaxDeliver(3),
	)
	if err != nil {
		return fmt.Errorf("failed to subscribe to tasks: %w", err)
	}
	defer sub.Unsubscribe()

	log.Printf("Scanner Agent %s running and listening for tasks...", cfg.agentID)

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	<-sigCh
	log.Println("Shutting down Scanner Agent...")
	return nil
}

func processTask(msg *nats.Msg, js nats.JetStreamContext) {
	var task types.RawScan
	if err := json.Unmarshal(msg.Data, &task); err != nil {
		log.Printf("Invalid task payload: %v", err)
		msg.Nak()
		return
	}

	log.Printf("Received scan task for subnet: %s port: %d", task.IP, task.Port)

	outputIPs, err := runZMapMock(task.IP, task.Port)
	if err != nil {
		log.Printf("ZMap scan failed: %v", err)
		msg.Nak()
		return
	}

	for _, ip := range outputIPs {
		rawScan := types.RawScan{
			IP:        ip,
			Port:      task.Port,
			Protocol:  "tcp",
			Service:   "http",
			ScannerID: cfg.agentID,
			ScannedAt: time.Now(),
		}
		data, err := json.Marshal(rawScan)
		if err != nil {
			continue
		}
		subject := fmt.Sprintf("scan.raw.%d", task.Port)
		_, _ = js.Publish(subject, data)
	}

	msg.Ack()
}

func runZMapMock(cidr string, port int) ([]string, error) {
	if _, err := exec.LookPath(cfg.zmapPath); err != nil {
		// Mock: Return default public IP as active host
		return []string{"93.184.216.34"}, nil
	}
	
	// Real ZMap call would go here.
	return []string{"93.184.216.34"}, nil
}
