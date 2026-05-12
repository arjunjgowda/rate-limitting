package ratelimit

import (
	"context"
	"errors"
	"time"
)

type tbfc struct {
	tbBase
	cancel    context.CancelFunc
	startTime time.Time // Anchor time for tick alignment
}

// NewTBFixedCounter initializes the rate limiter and starts background refill jobs.
func NewTBFixedCounter(ctx context.Context, refillRate uint, refillWindow time.Duration, windowUnit string, maxTokenLimit uint, shardCount uint) (*tbfc, error) {
	if refillWindow <= 0 {
		return nil, errors.New("refill window must be positive")
	}
	if shardCount == 0 || (shardCount&(shardCount-1)) != 0 {
		return nil, errors.New("shardCount must be a power of 2")
	}

	bgCtx, cancel := context.WithCancel(ctx)

	t := &tbfc{
		tbBase:    newTBBase(maxTokenLimit, refillRate, shardCount, refillWindow, windowUnit),
		cancel:    cancel,
		startTime: time.Now(),
	}

	for i := range shardCount {
		go t.startBackgroundRefill(bgCtx, uint(i))
	}

	return t, nil
}

func (t *tbfc) Allow(ctx context.Context, req RateLimitRequest) (*RateLimitResult, error) {
	now := req.CurrentTime
	if now.IsZero() {
		now = time.Now()
	}

	s := t.getShard(req.Key)
	s.mu.Lock()
	defer s.mu.Unlock()

	client := t.getOrCreateClient(s, req.Key, now)
	client.lastAccess = now

	// Calculate when the next background tick will happen
	elapsedSinceStart := now.Sub(t.startTime)
	nextTickIn := t.refillWindow - (elapsedSinceStart % t.refillWindow)

	// Calculate when the bucket will be completely full
	missing := float64(t.maxTokenLimit) - client.tokens
	ticksNeeded := uint(0)
	if missing > 0 {
		ticksNeeded = (uint(missing) + t.refillRate - 1) / t.refillRate // Ceiling division
	}

	var fullAt time.Time
	if ticksNeeded > 0 {
		fullAt = now.Add(nextTickIn).Add(time.Duration(ticksNeeded-1) * t.refillWindow)
	} else {
		fullAt = now
	}

	if client.tokens < float64(req.Cost) {
		return &RateLimitResult{
			Allowed:    false,
			Remaining:  uint(client.tokens),
			Count:      t.maxTokenLimit - uint(client.tokens),
			Limit:      t.maxTokenLimit,
			Window:     t.refillWindow,
			Unit:       t.windowUnit,
			RetryAfter: nextTickIn,
			FullAt:     fullAt,
		}, nil
	}

	// Consume tokens
	client.tokens -= float64(req.Cost)

	// Recalculate fullAt after consumption
	missing = float64(t.maxTokenLimit) - client.tokens
	ticksNeeded = (uint(missing) + t.refillRate - 1) / t.refillRate
	if ticksNeeded > 0 {
		fullAt = now.Add(nextTickIn).Add(time.Duration(ticksNeeded-1) * t.refillWindow)
	} else {
		fullAt = now
	}

	return &RateLimitResult{
		Allowed:   true,
		Remaining: uint(client.tokens),
		Count:     t.maxTokenLimit - uint(client.tokens),
		Limit:     t.maxTokenLimit,
		Window:    t.refillWindow,
		Unit:      t.windowUnit,
		FullAt:    fullAt,
	}, nil
}

// startBackgroundRefill runs in a goroutine and periodically refills tokens for a specific shard
func (t *tbfc) startBackgroundRefill(ctx context.Context, shardIdx uint) {
	ticker := time.NewTicker(t.refillWindow)
	defer ticker.Stop()

	s := t.shards[shardIdx]

	for {
		select {
		case <-ticker.C:
			s.mu.Lock()
			for key, client := range s.clients {
				newTokens := client.tokens + float64(t.refillRate)
				if newTokens > float64(t.maxTokenLimit) {
					newTokens = float64(t.maxTokenLimit)
				}
				client.tokens = newTokens

				// Optional: Eviction logic
				if client.tokens == float64(t.maxTokenLimit) && time.Since(client.lastAccess) > t.refillWindow*2 {
					delete(s.clients, key)
				}
			}
			s.mu.Unlock()

		case <-ctx.Done():
			return
		}
	}
}

func (t *tbfc) Close() error {
	t.cancel()
	return nil
}
