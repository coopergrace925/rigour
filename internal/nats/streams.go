package nats

import (
	"fmt"
	"time"

	"github.com/nats-io/nats.go"
)

const (
	StreamRawScans      = "RAW_SCANS"
	StreamEnrichedScans = "ENRICHED_SCANS"
	StreamScanEvents    = "SCAN_EVENTS"
)

func (c *Client) SetupStreams() error {
	// RAW_SCANS stream
	_, err := c.js.StreamInfo(StreamRawScans)
	if err != nil {
		// Stream doesn't exist, create it
		_, err = c.js.AddStream(&nats.StreamConfig{
			Name:      StreamRawScans,
			Subjects:  []string{"scan.raw.*"},
			Retention: nats.WorkQueuePolicy,
			MaxAge:    48 * time.Hour,
			Storage:   nats.FileStorage,
			Replicas:  1, // 3 in production
			Discard:   nats.DiscardOld,
		})
		if err != nil {
			return fmt.Errorf("failed to create RAW_SCANS stream: %w", err)
		}
	}

	// ENRICHED_SCANS stream
	_, err = c.js.StreamInfo(StreamEnrichedScans)
	if err != nil {
		// Stream doesn't exist, create it
		_, err = c.js.AddStream(&nats.StreamConfig{
			Name:      StreamEnrichedScans,
			Subjects:  []string{"scan.enriched.*"},
			Retention: nats.WorkQueuePolicy,
			MaxAge:    24 * time.Hour,
			Storage:   nats.FileStorage,
			Replicas:  1,
			Discard:   nats.DiscardOld,
		})
		if err != nil {
			return fmt.Errorf("failed to create ENRICHED_SCANS stream: %w", err)
		}
	}

	// SCAN_EVENTS stream (audit/monitoring)
	_, err = c.js.StreamInfo(StreamScanEvents)
	if err != nil {
		// Stream doesn't exist, create it
		_, err = c.js.AddStream(&nats.StreamConfig{
			Name:      StreamScanEvents,
			Subjects:  []string{"scan.events.*"},
			Retention: nats.LimitsPolicy,
			MaxAge:    7 * 24 * time.Hour,
			Storage:   nats.FileStorage,
			Replicas:  1,
		})
		if err != nil {
			return fmt.Errorf("failed to create SCAN_EVENTS stream: %w", err)
		}
	}

	return nil
}
