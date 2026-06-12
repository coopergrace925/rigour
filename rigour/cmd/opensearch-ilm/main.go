package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/ctrlsam/rigour/internal/opensearch"
	"github.com/spf13/cobra"
)

type config struct {
	opensearchURL string
	indexPattern  string
	policyName    string
}

var cfg config

var rootCmd = &cobra.Command{
	Use:   "opensearch-ilm",
	Short: "OpenSearch Index Lifecycle Management CLI",
}

var createPolicyCmd = &cobra.Command{
	Use:   "create-policy",
	Short: "Create ILM policy for hot/warm/delete lifecycle",
	RunE: func(cmd *cobra.Command, args []string) error {
		return createPolicy()
	},
}

var attachPolicyCmd = &cobra.Command{
	Use:   "attach-policy",
	Short: "Attach ILM policy to an index pattern",
	RunE: func(cmd *cobra.Command, args []string) error {
		return attachPolicy()
	},
}

var createRollingIndexCmd = &cobra.Command{
	Use:   "create-rolling",
	Short: "Create a new rolling index with write alias",
	RunE: func(cmd *cobra.Command, args []string) error {
		return createRollingIndex()
	},
}

var statsCmd = &cobra.Command{
	Use:   "stats",
	Short: "Show index statistics",
	RunE: func(cmd *cobra.Command, args []string) error {
		return showStats()
	},
}

var rolloverCmd = &cobra.Command{
	Use:   "rollover",
	Short: "Manually trigger index rollover",
	RunE: func(cmd *cobra.Command, args []string) error {
		return rollover()
	},
}

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show ILM policy status for an index",
	RunE: func(cmd *cobra.Command, args []string) error {
		return showStatus()
	},
}

func init() {
	rootCmd.PersistentFlags().StringVar(&cfg.opensearchURL, "url", "https://localhost:9200", "OpenSearch URL")
	rootCmd.PersistentFlags().StringVar(&cfg.policyName, "policy", "rigour-hosts-ilm", "ILM policy name")
	rootCmd.PersistentFlags().StringVar(&cfg.indexPattern, "index", "hosts", "Index pattern or name")

	rootCmd.AddCommand(createPolicyCmd)
	rootCmd.AddCommand(attachPolicyCmd)
	rootCmd.AddCommand(createRollingIndexCmd)
	rootCmd.AddCommand(statsCmd)
	rootCmd.AddCommand(rolloverCmd)
	rootCmd.AddCommand(statusCmd)
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func createPolicy() error {
	client, err := opensearch.NewClient([]string{cfg.opensearchURL})
	if err != nil {
		return fmt.Errorf("failed to connect to OpenSearch: %w", err)
	}

	ctx := context.Background()
	if err := client.CreateILMPolicy(ctx, cfg.policyName); err != nil {
		return fmt.Errorf("failed to create ILM policy: %w", err)
	}

	log.Printf("Successfully created ILM policy: %s", cfg.policyName)
	log.Println("Lifecycle: hot (7d) -> warm (90d) -> delete")
	log.Println("  - Hot: rollover at 1d or 30GB per shard")
	log.Println("  - Warm: reduce replicas, force merge")
	log.Println("  - Delete: after 90 days")

	return nil
}

func attachPolicy() error {
	client, err := opensearch.NewClient([]string{cfg.opensearchURL})
	if err != nil {
		return fmt.Errorf("failed to connect to OpenSearch: %w", err)
	}

	ctx := context.Background()
	if err := client.AttachILMPolicy(ctx, cfg.indexPattern, cfg.policyName); err != nil {
		return fmt.Errorf("failed to attach ILM policy: %w", err)
	}

	log.Printf("Successfully attached policy '%s' to index pattern '%s'", cfg.policyName, cfg.indexPattern)

	return nil
}

func createRollingIndex() error {
	client, err := opensearch.NewClient([]string{cfg.opensearchURL})
	if err != nil {
		return fmt.Errorf("failed to connect to OpenSearch: %w", err)
	}

	ctx := context.Background()
	if err := client.CreateRollingIndex(ctx, cfg.indexPattern); err != nil {
		return fmt.Errorf("failed to create rolling index: %w", err)
	}

	log.Printf("Successfully created rolling index with alias '%s'", cfg.indexPattern)
	log.Println("Write operations should use the alias, not the physical index name")

	return nil
}

func showStats() error {
	client, err := opensearch.NewClient([]string{cfg.opensearchURL})
	if err != nil {
		return fmt.Errorf("failed to connect to OpenSearch: %w", err)
	}

	ctx := context.Background()
	stats, err := client.GetIndexStats(ctx, cfg.indexPattern)
	if err != nil {
		return fmt.Errorf("failed to get index stats: %w", err)
	}

	log.Printf("Index Statistics for: %s", cfg.indexPattern)
	log.Printf("Stats: %+v", stats)

	return nil
}

func rollover() error {
	client, err := opensearch.NewClient([]string{cfg.opensearchURL})
	if err != nil {
		return fmt.Errorf("failed to connect to OpenSearch: %w", err)
	}

	ctx := context.Background()
	if err := client.ForceRollover(ctx, cfg.indexPattern); err != nil {
		return fmt.Errorf("failed to rollover index: %w", err)
	}

	log.Printf("Successfully triggered rollover for alias '%s'", cfg.indexPattern)

	return nil
}

func showStatus() error {
	client, err := opensearch.NewClient([]string{cfg.opensearchURL})
	if err != nil {
		return fmt.Errorf("failed to connect to OpenSearch: %w", err)
	}

	ctx := context.Background()
	status, err := client.GetILMPolicyStatus(ctx, cfg.indexPattern)
	if err != nil {
		return fmt.Errorf("failed to get ILM status: %w", err)
	}

	log.Printf("ILM Policy Status for: %s", cfg.indexPattern)
	log.Printf("Status: %+v", status)

	return nil
}
