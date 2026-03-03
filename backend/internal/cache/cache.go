package cache

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

// Client wraps a Redis client with methods for response caching.
type Client struct {
	rdb *redis.Client
}

// New creates a Client from a Redis URL (e.g. "redis://localhost:6379").
func New(redisURL string) (*Client, error) {
	opts, err := redis.ParseURL(redisURL)
	if err != nil {
		return nil, err
	}
	return &Client{rdb: redis.NewClient(opts)}, nil
}

// Ping tests the Redis connection.
func (c *Client) Ping(ctx context.Context) error {
	return c.rdb.Ping(ctx).Err()
}

// Get retrieves cached bytes by key. Returns nil, nil on cache miss.
func (c *Client) Get(ctx context.Context, key string) ([]byte, error) {
	val, err := c.rdb.Get(ctx, key).Bytes()
	if err == redis.Nil {
		return nil, nil
	}
	return val, err
}

// Set stores bytes under key with the given TTL.
func (c *Client) Set(ctx context.Context, key string, data []byte, ttl time.Duration) error {
	return c.rdb.Set(ctx, key, data, ttl).Err()
}

// Del deletes a key from Redis.
func (c *Client) Del(ctx context.Context, key string) error {
	return c.rdb.Del(ctx, key).Err()
}

// Close closes the underlying Redis connection.
func (c *Client) Close() error {
	return c.rdb.Close()
}
