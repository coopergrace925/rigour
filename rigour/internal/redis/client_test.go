package redis

import (
	"context"
	"testing"
	"time"
)

func TestRedisClientInterface(t *testing.T) {
	client := &Client{}
	ctx := context.Background()
	_, err := client.AcquireLock(ctx, "1.2.3.4/24", "agent", 1*time.Second)
	if err == nil {
		t.Error("Should fail when inner redis client is nil")
	}
}
