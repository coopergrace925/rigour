package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	internalnats "github.com/ctrlsam/rigour/internal/nats"
	"github.com/ctrlsam/rigour/pkg/types"
	"github.com/spf13/cobra"
)

type config struct {
	natsURL   string
	subject   string
	mode      string // "zmap-csv" or "zgrab2-json"
	scannerID string
}

var cfg config

var rootCmd = &cobra.Command{
	Use:   "zsend",
	Short: "Publish ZMap/ZGrab2 output to NATS JetStream",
	RunE: func(cmd *cobra.Command, args []string) error {
		return run()
	},
}

func init() {
	rootCmd.Flags().StringVar(&cfg.natsURL, "nats-url", "nats://localhost:4222", "NATS server URL")
	rootCmd.Flags().StringVar(&cfg.subject, "nats-subject", "scan.raw.0", "NATS subject to publish to")
	rootCmd.Flags().StringVar(&cfg.mode, "mode", "zgrab2-json", "Input mode: zmap-csv or zgrab2-json")
	rootCmd.Flags().StringVar(&cfg.scannerID, "scanner-id", "", "Scanner node ID")
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func run() error {
	// Set scanner-id default if empty
	if cfg.scannerID == "" {
		hostname, err := os.Hostname()
		if err != nil {
			fmt.Fprintf(os.Stderr, "warning: failed to get hostname: %v\n", err)
			cfg.scannerID = "unknown"
		} else {
			cfg.scannerID = hostname
		}
	}

	client, err := internalnats.NewClient(cfg.natsURL)
	if err != nil {
		return fmt.Errorf("failed to connect to NATS: %w", err)
	}
	defer client.Close()

	js := client.JetStream()
	scanner := bufio.NewScanner(os.Stdin)
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024) // 1MB buffer

	var published, errors int64

	for scanner.Scan() {
		line := scanner.Text()
		if strings.TrimSpace(line) == "" {
			continue
		}

		var scan *types.RawScan
		var parseErr error

		switch cfg.mode {
		case "zmap-csv":
			scan, parseErr = parseZMapCSVLine(line)
		case "zgrab2-json":
			scan, parseErr = parseZGrab2JSONLine([]byte(line))
		default:
			return fmt.Errorf("unknown mode: %s", cfg.mode)
		}

		if parseErr != nil {
			errors++
			fmt.Fprintf(os.Stderr, "parse error: %v\n", parseErr)
			continue
		}

		scan.ScannerID = cfg.scannerID
		scan.ScannedAt = time.Now()

		data, err := json.Marshal(scan)
		if err != nil {
			errors++
			fmt.Fprintf(os.Stderr, "marshal error: %v\n", err)
			continue
		}

		subject := fmt.Sprintf("scan.raw.%d", scan.Port)
		_, err = js.Publish(subject, data)
		if err != nil {
			errors++
			fmt.Fprintf(os.Stderr, "publish error: %v\n", err)
			continue
		}

		published++
		if published%10000 == 0 {
			fmt.Fprintf(os.Stderr, "published: %d, errors: %d\n", published, errors)
		}
	}

	fmt.Fprintf(os.Stderr, "done: published=%d errors=%d\n", published, errors)
	return scanner.Err()
}

func parseZMapCSVLine(line string) (*types.RawScan, error) {
	parts := strings.Split(line, ",")
	if len(parts) < 4 {
		return nil, fmt.Errorf("invalid CSV line: expected 4+ fields, got %d", len(parts))
	}

	ip := strings.TrimSpace(parts[0])
	if !isValidIPv4(ip) {
		return nil, fmt.Errorf("invalid IP address: %s", ip)
	}

	port, err := strconv.Atoi(strings.TrimSpace(parts[1]))
	if err != nil {
		return nil, fmt.Errorf("invalid port: %w", err)
	}

	return &types.RawScan{
		IP:       ip,
		Port:     port,
		Protocol: "tcp",
	}, nil
}

func parseZGrab2JSONLine(data []byte) (*types.RawScan, error) {
	var raw struct {
		IP     string                 `json:"ip"`
		Domain string                 `json:"domain,omitempty"`
		Port   int                    `json:"port,omitempty"`
		Data   map[string]interface{} `json:"data"`
	}

	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("invalid JSON: %w", err)
	}

	if !isValidIPv4(raw.IP) {
		return nil, fmt.Errorf("invalid IP address: %s", raw.IP)
	}

	scan := &types.RawScan{
		IP:   raw.IP,
		Port: raw.Port,
	}

	// Extract protocol data from ZGrab2 output
	for proto, protoData := range raw.Data {
		scan.Service = proto
		if pd, ok := protoData.(map[string]interface{}); ok {
			if status, ok := pd["status"].(string); ok {
				scan.ZGrabData.Status = status
			}
			scan.ZGrabData.Protocol = proto
		}
	}

	return scan, nil
}

func isValidIPv4(ip string) bool {
	parts := strings.Split(ip, ".")
	if len(parts) != 4 {
		return false
	}
	for _, part := range parts {
		n, err := strconv.Atoi(part)
		if err != nil || n < 0 || n > 255 {
			return false
		}
	}
	return true
}

func buildRawScan(ip string, port int, protocol string, scannerID string) *types.RawScan {
	return &types.RawScan{
		IP:        ip,
		Port:      port,
		Protocol:  protocol,
		ScannerID: scannerID,
		ScannedAt: time.Now(),
	}
}
