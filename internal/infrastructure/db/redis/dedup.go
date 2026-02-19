package redis

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

const dedupTTL = time.Hour

// DedupChecker provides idempotency checks backed by Redis.
// Key format: dedup:<tracking_number>:<status>:<unix_timestamp>
type DedupChecker struct {
	client *redis.Client
}

// NewDedupChecker creates a DedupChecker wrapping the given Redis client.
func NewDedupChecker(client *redis.Client) *DedupChecker {
	return &DedupChecker{client: client}
}

// IsDuplicate reports whether this exact event has already been processed.
func (d *DedupChecker) IsDuplicate(ctx context.Context, trackingNumber, status string, ts time.Time) (bool, error) {
	n, err := d.client.Exists(ctx, d.key(trackingNumber, status, ts)).Result()
	if err != nil {
		return false, fmt.Errorf("dedup check: %w", err)
	}
	return n > 0, nil
}

// Mark records that this event has been processed (expires after dedupTTL).
func (d *DedupChecker) Mark(ctx context.Context, trackingNumber, status string, ts time.Time) error {
	return d.client.Set(ctx, d.key(trackingNumber, status, ts), "1", dedupTTL).Err()
}

func (d *DedupChecker) key(trackingNumber, status string, ts time.Time) string {
	return fmt.Sprintf("dedup:%s:%s:%d", trackingNumber, status, ts.Unix())
}
