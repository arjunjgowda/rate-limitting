package ratelimit

import (
	"context"
	"time"
)

type RateLimitRequest struct {
	Key         string
	Cost        int
	CurrentTime time.Time
}

type RateLimitResult struct {
	Allowed    bool
	Count      int
	Limit      int
	Remaining  int
	ResetTime  time.Time
	RetryAfter time.Duration
}

type RateLimiter interface {
	// Allow now uses the universal context.Context
	Allow(ctx context.Context, req RateLimitRequest) (*RateLimitResult, error)
}
