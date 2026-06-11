package redis

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

type Client struct {
	rdb *redis.Client
}

func NewClient(addr string) (*Client, error) {
	rdb := redis.NewClient(&redis.Options{
		Addr: addr,
	})
	
	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	
	if err := rdb.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to ping Redis: %w", err)
	}
	
	return &Client{rdb: rdb}, nil
}

func (c *Client) Close() error {
	if c.rdb != nil {
		return c.rdb.Close()
	}
	return nil
}

func (c *Client) AcquireLock(ctx context.Context, cidr string, agentID string, ttl time.Duration) (bool, error) {
	if c.rdb == nil {
		return false, fmt.Errorf("redis client not initialized")
	}
	key := fmt.Sprintf("scan:lock:%s", cidr)
	success, err := c.rdb.SetNX(ctx, key, agentID, ttl).Result()
	if err != nil {
		return false, fmt.Errorf("failed to acquire lock: %w", err)
	}
	return success, nil
}

func (c *Client) IncrementASNRate(ctx context.Context, asn int, ttl time.Duration) (int64, error) {
	if c.rdb == nil {
		return 0, fmt.Errorf("redis client not initialized")
	}
	key := fmt.Sprintf("rate:asn:%d", asn)
	pipe := c.rdb.Pipeline()
	incr := pipe.Incr(ctx, key)
	pipe.Expire(ctx, key, ttl)
	_, err := pipe.Exec(ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to increment ASN rate: %w", err)
	}
	return incr.Val(), nil
}
