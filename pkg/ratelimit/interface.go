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
	Count      uint
	Limit      uint
	Remaining  uint
	Window     time.Duration
	Unit       string
	FullAt     time.Time
	RetryAfter time.Duration
}

type RateLimiter interface {
	// Allow now uses the universal context.Context
	Allow(ctx context.Context, req RateLimitRequest) (*RateLimitResult, error)
	// Close cleans up any background resources (goroutines, tickers, etc.)
	Close() error
}
