package redis

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	rateLimitPrefix = "ratelimit:"
)

// RateLimiter handles rate limiting using Redis
type RateLimiter struct {
	client            *Client
	requestsPerMinute int
	burst             int
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter(client *Client, requestsPerMinute, burst int) *RateLimiter {
	return &RateLimiter{
		client:            client,
		requestsPerMinute: requestsPerMinute,
		burst:             burst,
	}
}

// Allow checks if a request should be allowed based on rate limits
// Returns (allowed, remaining, resetTime, error)
func (r *RateLimiter) Allow(ctx context.Context, key string) (bool, int, time.Time, error) {
	fullKey := fmt.Sprintf("%s%s", rateLimitPrefix, key)
	now := time.Now()
	windowStart := now.Truncate(time.Minute)
	windowEnd := windowStart.Add(time.Minute)

	pipe := r.client.rdb.Pipeline()

	// Increment counter
	incrCmd := pipe.Incr(ctx, fullKey)

	// Set expiry if key is new
	pipe.ExpireNX(ctx, fullKey, time.Minute)

	_, err := pipe.Exec(ctx)
	if err != nil && err != redis.Nil {
		return false, 0, time.Time{}, fmt.Errorf("failed to execute rate limit check: %w", err)
	}

	count := incrCmd.Val()
	limit := int64(r.requestsPerMinute + r.burst)
	remaining := int(limit - count)
	if remaining < 0 {
		remaining = 0
	}

	allowed := count <= limit

	return allowed, remaining, windowEnd, nil
}

// Reset resets the rate limit counter for a key
func (r *RateLimiter) Reset(ctx context.Context, key string) error {
	fullKey := fmt.Sprintf("%s%s", rateLimitPrefix, key)
	return r.client.rdb.Del(ctx, fullKey).Err()
}
