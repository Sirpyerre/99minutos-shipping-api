package redis

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

const defaultTimeout = 5 * time.Second

// Config captures the settings for establishing a Redis connection.
type Config struct {
	Addr    string
	DB      int
	Timeout time.Duration
}

// Connect initialises a Redis client and validates connectivity with a ping.
// A default timeout is applied when none is provided.
func Connect(ctx context.Context, cfg Config) (*redis.Client, error) {
	timeout := cfg.Timeout
	if timeout <= 0 {
		timeout = defaultTimeout
	}

	client := redis.NewClient(&redis.Options{
		Addr: cfg.Addr,
		DB:   cfg.DB,
	})

	pingCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	if err := client.Ping(pingCtx).Err(); err != nil {
		_ = client.Close()
		return nil, fmt.Errorf("redis ping: %w", err)
	}

	return client, nil
}
