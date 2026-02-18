package mongo

import (
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const defaultTimeout = 10 * time.Second

// Config captures the minimal settings required to establish a MongoDB connection.
type Config struct {
	URI      string
	Database string
	Timeout  time.Duration
}

// Connect establishes a MongoDB client, verifies connectivity with a ping, and
// returns both the client and the selected database. A default timeout is
// applied when none is provided.
func Connect(ctx context.Context, cfg Config) (*mongo.Client, *mongo.Database, error) {
	timeout := cfg.Timeout
	if timeout <= 0 {
		timeout = defaultTimeout
	}

	connectCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	client, err := mongo.Connect(connectCtx, options.Client().ApplyURI(cfg.URI))
	if err != nil {
		return nil, nil, fmt.Errorf("mongo connect: %w", err)
	}

	if err := client.Ping(connectCtx, nil); err != nil {
		_ = client.Disconnect(connectCtx)
		return nil, nil, fmt.Errorf("mongo ping: %w", err)
	}

	db := client.Database(cfg.Database)
	return client, db, nil
}
