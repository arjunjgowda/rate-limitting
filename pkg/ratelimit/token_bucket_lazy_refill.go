package ratelimit

import (
	"context"
	"errors"
	"time"
)

type tblr struct {
	tbBase
	cancel context.CancelFunc
}

func NewTBLazyRefillRL(ctx context.Context, refillRate uint, refillWindow time.Duration, windowUnit string, maxTokenLimit uint, shardCount uint) (*tblr, error) {
	if refillWindow <= 0 {
		return nil, errors.New("refill window must be positive")
	}
	if shardCount == 0 || (shardCount&(shardCount-1)) != 0 {
		return nil, errors.New("shardCount must be a power of 2")
	}

	bgCtx, cancel := context.WithCancel(ctx)

	r := &tblr{
		tbBase: newTBBase(maxTokenLimit, refillRate, shardCount, refillWindow, windowUnit),
		cancel: cancel,
	}

	for i := range shardCount {
		go r.startBackgroundEviction(bgCtx, uint(i))
	}

	return r, nil
}

func (r *tblr) Allow(ctx context.Context, req RateLimitRequest) (*RateLimitResult, error) {
	now := req.CurrentTime
	if now.IsZero() {
		now = time.Now()
	}

	s := r.getShard(req.Key)
	s.mu.Lock()
	defer s.mu.Unlock()

	c := r.getOrCreateClient(s, req.Key, now)

	// 1. Refill based on elapsed time (Lazy Refill)
	elapsed := now.Sub(c.lastAccess)
	if elapsed > 0 {
		tokensToAdd := float64(elapsed) * float64(r.refillRate) / float64(r.refillWindow)
		c.tokens += tokensToAdd
		if c.tokens > float64(r.maxTokenLimit) {
			c.tokens = float64(r.maxTokenLimit)
		}
		c.lastAccess = now
	}

	cost := float64(req.Cost)
	timePerToken := float64(r.refillWindow) / float64(r.refillRate)

	// 2. Check if allowed
	if c.tokens < cost {
		missing := cost - c.tokens
		retryAfter := time.Duration(missing * timePerToken)

		return &RateLimitResult{
			Allowed:    false,
			Remaining:  uint(c.tokens),
			Count:      r.maxTokenLimit - uint(c.tokens),
			Limit:      r.maxTokenLimit,
			Window:     r.refillWindow,
			Unit:       r.windowUnit,
			FullAt:     now.Add(time.Duration((float64(r.maxTokenLimit) - c.tokens) * timePerToken)),
			RetryAfter: retryAfter,
		}, nil
	}

	// 3. Consume tokens
	c.tokens -= cost

	return &RateLimitResult{
		Allowed:   true,
		Remaining: uint(c.tokens),
		Count:     r.maxTokenLimit - uint(c.tokens),
		Limit:     r.maxTokenLimit,
		Window:    r.refillWindow,
		Unit:      r.windowUnit,
		FullAt:    now.Add(time.Duration((float64(r.maxTokenLimit) - c.tokens) * timePerToken)),
	}, nil
}

func (r *tblr) startBackgroundEviction(ctx context.Context, shardIdx uint) {
	ticker := time.NewTicker(r.refillWindow * 2)
	defer ticker.Stop()

	s := r.shards[shardIdx]

	for {
		select {
		case <-ticker.C:
			s.mu.Lock()
			now := time.Now()
			for key, client := range s.clients {
				// Calculate tokens lazily to see if it's full
				elapsed := now.Sub(client.lastAccess)
				tokens := client.tokens
				if elapsed > 0 {
					tokensToAdd := float64(elapsed) * float64(r.refillRate) / float64(r.refillWindow)
					tokens += tokensToAdd
					if tokens > float64(r.maxTokenLimit) {
						tokens = float64(r.maxTokenLimit)
					}
				}

				// Evict if full and inactive for > 2 * refillWindow
				if tokens == float64(r.maxTokenLimit) && now.Sub(client.lastAccess) > r.refillWindow*2 {
					delete(s.clients, key)
				}
			}
			s.mu.Unlock()

		case <-ctx.Done():
			return
		}
	}
}

func (r *tblr) Close() error {
	r.cancel()
	return nil
}
