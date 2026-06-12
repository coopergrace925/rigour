package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ctrlsam/rigour/internal/api"
	"github.com/ctrlsam/rigour/internal/blocklist"
	"github.com/ctrlsam/rigour/internal/opensearch"
	internalredis "github.com/ctrlsam/rigour/internal/redis"
	storageopensearch "github.com/ctrlsam/rigour/internal/storage/opensearch"
	"github.com/spf13/cobra"
)

type cliConfig struct {
	opensearchURL string
	redisAddr     string
	addr          string
}

var config cliConfig

var rootCmd = &cobra.Command{
	Use:   "rigour-api",
	Short: "REST API server for Rigour",
	Long:  "A REST API server for querying scanned hosts and services from OpenSearch",
	RunE: func(cmd *cobra.Command, args []string) error {
		return runServer(cmd.Context())
	},
}

func init() {
	rootCmd.Flags().StringVar(&config.opensearchURL, "opensearch-url", "https://localhost:9200", "OpenSearch URL")
	rootCmd.Flags().StringVar(&config.redisAddr, "redis-addr", "localhost:6379", "Redis address")
	rootCmd.Flags().StringVar(&config.addr, "addr", ":8080", "Server address (host:port)")
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func runServer(ctx context.Context) error {
	// Create OpenSearch client
	osClient, err := opensearch.NewClient([]string{config.opensearchURL})
	if err != nil {
		return fmt.Errorf("failed to connect to OpenSearch: %w", err)
	}

	// Create Redis client
	redisClient, err := internalredis.NewClient(config.redisAddr)
	if err != nil {
		return fmt.Errorf("failed to connect to Redis: %w", err)
	}
	defer redisClient.Close()

	// Create OpenSearch repository
	repository := storageopensearch.NewHostRepository(osClient)

	// Create blocklist for opt-out support
	bl := blocklist.NewBlocklist()

	// Create rate limiter
	rateLimiter := api.NewRateLimiter(redisClient)

	// Create analytics handler (ClickHouse client would be passed here)
	var analyticsHandler *api.AnalyticsHandler
	// analyticsHandler = api.NewAnalyticsHandler(clickhouseClient)

	// Create router and handler with middleware (pass blocklist for opt-out support)
	router := api.NewRouterWithAnalytics(repository, redisClient, analyticsHandler, bl)
	
	// Apply middleware to the handler
	var handler http.Handler = router.Handler()
	handler = rateLimiter.RateLimitMiddleware(60)(handler) // 60 requests per minute
	handler = rateLimiter.APIKeyAuthMiddleware(handler)

	// Setup HTTP server
	server := &http.Server{
		Addr:         config.addr,
		Handler:      handler,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Setup graceful shutdown
	ctx, cancel := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	defer cancel()

	// Start server in a goroutine
	go func() {
		fmt.Printf("Starting Rigour API server on %s\n", config.addr)
		fmt.Printf("OpenSearch: %s\n", config.opensearchURL)
		fmt.Printf("Redis: %s\n", config.redisAddr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			fmt.Fprintf(os.Stderr, "error: server failed: %v\n", err)
		}
	}()

	// Wait for interrupt signal
	<-ctx.Done()

	// Graceful shutdown
	fmt.Println("\nShutting down server...")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("server shutdown failed: %w", err)
	}

	fmt.Println("Server stopped")
	return nil
}
