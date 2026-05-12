package ratelimit

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/cespare/xxhash/v2"
	"github.com/stretchr/testify/assert"
)

const (
	testRefillRate     = 10
	testMaxTokenLimit  = 100
	testShardCount     = 64
)

func TestNewRateLimiter(t *testing.T) {
	t.Run("should_create_rate_limiter_with_valid_params", func(t *testing.T) {
		rl, err := NewTBLazyRefillRL(context.Background(), testRefillRate, time.Second, "seconds", testMaxTokenLimit, testShardCount)
		assert.NoError(t, err)
		assert.NotNil(t, rl)
	})

	t.Run("should_create_rate_limiter_with_single_shard", func(t *testing.T) {
		rl, err := NewTBLazyRefillRL(context.Background(), testRefillRate, time.Second, "seconds", testMaxTokenLimit, 1)
		assert.NoError(t, err)
		assert.Equal(t, uint(1), rl.shardCount)
	})

	t.Run("should_create_rate_limiter_with_multiple_shards", func(t *testing.T) {
		rl, err := NewTBLazyRefillRL(context.Background(), testRefillRate, time.Second, "seconds", testMaxTokenLimit, 8)
		assert.NoError(t, err)
		assert.Equal(t, uint(8), rl.shardCount)
	})

	t.Run("should_create_when_refill_is_faster_than_total_capacity", func(t *testing.T) {
		rl, err := NewTBLazyRefillRL(context.Background(), 1000, time.Second, "seconds", 10, testShardCount)
		assert.NoError(t, err)
		assert.NotNil(t, rl)
	})

	t.Run("should_fail_when_shard_count_is_zero", func(t *testing.T) {
		rl, err := NewTBLazyRefillRL(context.Background(), testRefillRate, time.Second, "seconds", testMaxTokenLimit, 0)
		assert.Error(t, err)
		assert.Nil(t, rl)
	})

	t.Run("should_fail_when_shard_count_is_not_power_of_two", func(t *testing.T) {
		rl, err := NewTBLazyRefillRL(context.Background(), testRefillRate, time.Second, "seconds", testMaxTokenLimit, 7)
		assert.Error(t, err)
		assert.Nil(t, rl)
	})
}

func TestAllow(t *testing.T) {
	t.Run("well_within_limit", func(t *testing.T) {
		rl, _ := NewTBLazyRefillRL(context.Background(), testRefillRate, time.Second, "seconds", testMaxTokenLimit, testShardCount)

		ctx := context.Background()
		req := RateLimitRequest{
			Key:         "user-1",
			Cost:        1,
			CurrentTime: time.Now(),
		}

		res, err := rl.Allow(ctx, req)
		assert.NoError(t, err)
		assert.True(t, res.Allowed)
		assert.Equal(t, uint(99), res.Remaining)
		assert.Equal(t, uint(1), res.Count)
		assert.Equal(t, uint(testMaxTokenLimit), res.Limit)
	})

	t.Run("multi_request_well_within_limit", func(t *testing.T) {
		rl, _ := NewTBLazyRefillRL(context.Background(), testRefillRate, time.Second, "seconds", testMaxTokenLimit, testShardCount)

		ctx := context.Background()
		now := time.Now()
		req1 := RateLimitRequest{
			Key:         "user-1",
			Cost:        1,
			CurrentTime: now,
		}

		req2 := RateLimitRequest{
			Key:         "user-1",
			Cost:        1,
			CurrentTime: now,
		}

		rl.Allow(ctx, req1)
		res, err := rl.Allow(ctx, req2)

		assert.NoError(t, err)
		assert.True(t, res.Allowed)
		assert.Equal(t, uint(98), res.Remaining)
	})

	t.Run("should_handle_a_burst_of_individual_requests_until_capacity_is_reached", func(t *testing.T) {
		rl, _ := NewTBLazyRefillRL(context.Background(), testRefillRate, time.Second, "seconds", testMaxTokenLimit, testShardCount)
		ctx := context.Background()
		now := time.Now()

		// Simulate individual requests arriving at the same instant up to capacity
		for i := uint(1); i <= testMaxTokenLimit; i++ {
			res, err := rl.Allow(ctx, RateLimitRequest{
				Key:         "user-1",
				Cost:        1,
				CurrentTime: now,
			})
			assert.NoError(t, err)
			assert.True(t, res.Allowed, "Request %d should be allowed", i)
			assert.Equal(t, uint(testMaxTokenLimit-i), res.Remaining)
		}

		// The very next request should fail immediately
		res, _ := rl.Allow(ctx, RateLimitRequest{
			Key:         "user-1",
			Cost:        1,
			CurrentTime: now,
		})
		assert.False(t, res.Allowed, "Should be denied after capacity is exhausted")
		assert.Equal(t, uint(0), res.Remaining)
		// retryAfter should be cost (1) * timePerToken (1s / 10 = 100ms)
		assert.Equal(t, 100*time.Millisecond, res.RetryAfter)
	})

	t.Run("should_allow_multiple_user_burst_without_interference", func(t *testing.T) {
		rl, _ := NewTBLazyRefillRL(context.Background(), testRefillRate, time.Second, "seconds", testMaxTokenLimit, testShardCount)
		ctx := context.Background()
		now := time.Now()

		// User 1 consumes their entire bucket
		for i := 0; i < int(testMaxTokenLimit); i++ {
			rl.Allow(ctx, RateLimitRequest{Key: "user-1", Cost: 1, CurrentTime: now})
		}

		// User 1 is now blocked
		res1, _ := rl.Allow(ctx, RateLimitRequest{Key: "user-1", Cost: 1, CurrentTime: now})
		assert.False(t, res1.Allowed, "User-1 should be blocked")

		// User 2 should be completely unaffected and have a full bucket
		res2, _ := rl.Allow(ctx, RateLimitRequest{Key: "user-2", Cost: 1, CurrentTime: now})
		assert.True(t, res2.Allowed, "User-2 should be allowed")
		assert.Equal(t, uint(testMaxTokenLimit-1), res2.Remaining)
	})

	t.Run("should_initialize_new_users_with_a_full_bucket", func(t *testing.T) {
		rl, _ := NewTBLazyRefillRL(context.Background(), testRefillRate, time.Second, "seconds", testMaxTokenLimit, testShardCount)
		ctx := context.Background()
		now := time.Now()

		res, _ := rl.Allow(ctx, RateLimitRequest{Key: "new-user", Cost: 1, CurrentTime: now})
		assert.True(t, res.Allowed)
		assert.Equal(t, uint(testMaxTokenLimit-1), res.Remaining)
	})

	t.Run("should_refill_tokens_after_a_time_gap", func(t *testing.T) {
		// Rate: 10 tokens per second
		rl, _ := NewTBLazyRefillRL(context.Background(), 10, time.Second, "seconds", 100, testShardCount)
		ctx := context.Background()
		t0 := time.Now()

		// Consume all 100 tokens at T0
		rl.Allow(ctx, RateLimitRequest{Key: "user-1", Cost: 100, CurrentTime: t0})

		// At T0 + 5 seconds, we should have 50 tokens refilled (10 tokens/sec * 5 sec)
		t1 := t0.Add(5 * time.Second)
		res, _ := rl.Allow(ctx, RateLimitRequest{Key: "user-1", Cost: 10, CurrentTime: t1})

		assert.True(t, res.Allowed)
		assert.Equal(t, uint(40), res.Remaining) // 50 refilled - 10 consumed
	})

	t.Run("one_user_above_limit_and_other_user_within_limit", func(t *testing.T) {
		rl, _ := NewTBLazyRefillRL(context.Background(), testRefillRate, time.Second, "seconds", testMaxTokenLimit, testShardCount)
		ctx := context.Background()
		now := time.Now()

		// User 1 consumes their entire bucket
		for i := 0; i < int(testMaxTokenLimit); i++ {
			rl.Allow(ctx, RateLimitRequest{Key: "user-1", Cost: 1, CurrentTime: now})
		}
		res1, _ := rl.Allow(ctx, RateLimitRequest{Key: "user-1", Cost: 1, CurrentTime: now})
		assert.False(t, res1.Allowed, "User-1 should be blocked")

		// User 2 makes a single request and should be allowed
		res2, _ := rl.Allow(ctx, RateLimitRequest{Key: "user-2", Cost: 1, CurrentTime: now})
		assert.True(t, res2.Allowed, "User-2 should be allowed")
	})

	t.Run("should_allow_request_with_cost_>_1", func(t *testing.T) {
		rl, _ := NewTBLazyRefillRL(context.Background(), testRefillRate, time.Second, "seconds", 100, testShardCount)
		ctx := context.Background()
		now := time.Now()

		// Requesting 50 tokens at once
		res, _ := rl.Allow(ctx, RateLimitRequest{Key: "user-1", Cost: 50, CurrentTime: now})
		assert.True(t, res.Allowed)
		assert.Equal(t, uint(50), res.Remaining)
	})

	t.Run("should_refill_correctly_for_sub-second_gaps_(partial_refill)", func(t *testing.T) {
		// Rate: 10 tokens per 1s (1 token per 100ms)
		rl, _ := NewTBLazyRefillRL(context.Background(), 10, time.Second, "seconds", 100, testShardCount)
		ctx := context.Background()
		t0 := time.Now()

		// Consume all tokens
		rl.Allow(ctx, RateLimitRequest{Key: "user-1", Cost: 100, CurrentTime: t0})

		// After 500ms, should have exactly 5 tokens refilled (10 * 0.5 sec)
		t1 := t0.Add(500 * time.Millisecond)
		res, _ := rl.Allow(ctx, RateLimitRequest{Key: "user-1", Cost: 2, CurrentTime: t1})

		assert.True(t, res.Allowed)
		assert.Equal(t, uint(3), res.Remaining) // 5 refilled - 2 consumed
	})

	t.Run("should_handle_zero_value_CurrentTime_gracefully", func(t *testing.T) {
		rl, _ := NewTBLazyRefillRL(context.Background(), testRefillRate, time.Second, "seconds", testMaxTokenLimit, testShardCount)
		ctx := context.Background()

		// Passing zero time (time.Time{}) should still allow the request
		// but no tokens will refill because no "time" has passed.
		res, _ := rl.Allow(ctx, RateLimitRequest{Key: "user-1", Cost: 1, CurrentTime: time.Time{}})
		assert.True(t, res.Allowed)
		assert.Equal(t, uint(testMaxTokenLimit-1), res.Remaining)
	})

	t.Run("should_not_allow_over_limit", func(t *testing.T) {
		// Capacity is only 5 tokens
		rl, _ := NewTBLazyRefillRL(context.Background(), 1, time.Second, "seconds", 5, testShardCount)
		ctx := context.Background()
		now := time.Now()

		// Requesting 6 tokens when capacity is only 5
		res, _ := rl.Allow(ctx, RateLimitRequest{Key: "user-1", Cost: 6, CurrentTime: now})

		assert.False(t, res.Allowed)
		assert.Equal(t, uint(5), res.Remaining) // Tokens remain unchanged
	})

	t.Run("two_users_can_both_be_independently_above_limit", func(t *testing.T) {
		rl, _ := NewTBLazyRefillRL(context.Background(), 1, time.Second, "seconds", 5, testShardCount)
		ctx := context.Background()
		now := time.Now()

		// Both users consume their entire buckets
		rl.Allow(ctx, RateLimitRequest{Key: "user-1", Cost: 5, CurrentTime: now})
		rl.Allow(ctx, RateLimitRequest{Key: "user-2", Cost: 5, CurrentTime: now})

		// Both users should now be blocked
		res1, _ := rl.Allow(ctx, RateLimitRequest{Key: "user-1", Cost: 1, CurrentTime: now})
		res2, _ := rl.Allow(ctx, RateLimitRequest{Key: "user-2", Cost: 1, CurrentTime: now})

		assert.False(t, res1.Allowed, "User-1 should be blocked")
		assert.False(t, res2.Allowed, "User-2 should be blocked")
	})

	t.Run("should_fail_till_refill", func(t *testing.T) {
		// Rate: 1 token per 1s
		rl, _ := NewTBLazyRefillRL(context.Background(), 1, time.Second, "seconds", 1, testShardCount)
		ctx := context.Background()
		t0 := time.Now()

		// Consume the only token at T0
		rl.Allow(ctx, RateLimitRequest{Key: "user-1", Cost: 1, CurrentTime: t0})

		// At T0 + 500ms, it should still fail (only 0.5 tokens produced, need 1.0)
		t1 := t0.Add(500 * time.Millisecond)
		res1, _ := rl.Allow(ctx, RateLimitRequest{Key: "user-1", Cost: 1, CurrentTime: t1})
		assert.False(t, res1.Allowed, "Should be denied at 500ms")

		// At T0 + 1s, it should finally succeed
		t2 := t0.Add(1 * time.Second)
		res2, _ := rl.Allow(ctx, RateLimitRequest{Key: "user-1", Cost: 1, CurrentTime: t2})
		assert.True(t, res2.Allowed, "Should be allowed at 1s")
	})
	t.Run("should_fail_if_single_request_cost_exceeds_maximum_bucket_capacity", func(t *testing.T) {
		rl, _ := NewTBLazyRefillRL(context.Background(), testRefillRate, time.Second, "seconds", 10, testShardCount)
		ctx := context.Background()
		now := time.Now()

		// Bucket only holds 10 tokens, so a cost of 11 is impossible
		res, _ := rl.Allow(ctx, RateLimitRequest{Key: "user-1", Cost: 11, CurrentTime: now})

		assert.False(t, res.Allowed)
		assert.Equal(t, uint(10), res.Remaining)
	})

	t.Run("should_handle_clock_regression_gracefully", func(t *testing.T) {
		rl, _ := NewTBLazyRefillRL(context.Background(), 10, time.Second, "seconds", 10, testShardCount)
		ctx := context.Background()
		now := time.Now()

		// First request at T=now. Bucket: 10 -> 5 tokens.
		rl.Allow(ctx, RateLimitRequest{Key: "user-1", Cost: 5, CurrentTime: now})

		// Second request at T=now-1s (clock regressed).
		// Refill should be 0, and it should just subtract from the current 5 tokens.
		res, _ := rl.Allow(ctx, RateLimitRequest{Key: "user-1", Cost: 1, CurrentTime: now.Add(-time.Second)})

		assert.True(t, res.Allowed)
		assert.Equal(t, uint(4), res.Remaining) // 5 - 1 = 4
	})

	t.Run("should_handle_concurrent_requests_for_the_same_key_safely", func(t *testing.T) {
		capacity := 50
		rl, _ := NewTBLazyRefillRL(context.Background(), 10, time.Second, "seconds", uint(capacity), testShardCount)
		ctx := context.Background()
		now := time.Now()
		key := "concurrent-user"

		concurrentReqs := 100
		var allowedCount int32
		var deniedCount int32
		var wg sync.WaitGroup

		wg.Add(concurrentReqs)
		for i := 0; i < concurrentReqs; i++ {
			go func() {
				defer wg.Done()
				res, err := rl.Allow(ctx, RateLimitRequest{
					Key:         key,
					Cost:        1,
					CurrentTime: now,
				})
				if err == nil {
					if res.Allowed {
						atomic.AddInt32(&allowedCount, 1)
					} else {
						atomic.AddInt32(&deniedCount, 1)
					}
				}
			}()
		}
		wg.Wait()

		assert.Equal(t, int32(capacity), allowedCount, "Should allow exactly up to capacity")
		assert.Equal(t, int32(concurrentReqs-capacity), deniedCount, "Should deny exactly the remainder")

		// Final check: remaining tokens should be zero
		res, _ := rl.Allow(ctx, RateLimitRequest{Key: key, Cost: 0, CurrentTime: now})
		assert.Equal(t, uint(0), res.Remaining)
	})
	t.Run("should_handle_concurrent_requests_across_different_shards", func(t *testing.T) {
		rl, _ := NewTBLazyRefillRL(context.Background(), testRefillRate, time.Second, "seconds", testMaxTokenLimit, testShardCount)
		ctx := context.Background()
		now := time.Now()

		userCount := 100
		var wg sync.WaitGroup
		wg.Add(userCount)

		for i := 0; i < userCount; i++ {
			userID := fmt.Sprintf("user-%d", i)
			go func(id string) {
				defer wg.Done()
				res, err := rl.Allow(ctx, RateLimitRequest{
					Key:         id,
					Cost:        1,
					CurrentTime: now,
				})
				assert.NoError(t, err)
				assert.True(t, res.Allowed)
			}(userID)
		}
		wg.Wait()
	})

	t.Run("should_refill_correctly_for_per-minute_window", func(t *testing.T) {
		// Rate: 60 tokens per 1 minute (1 token per second)
		rl, _ := NewTBLazyRefillRL(context.Background(), 60, time.Minute, "minutes", 60, testShardCount)
		ctx := context.Background()
		t0 := time.Now()

		// Consume all
		_, _ = rl.Allow(ctx, RateLimitRequest{Key: "u1", Cost: 60, CurrentTime: t0})

		// Wait 30 seconds -> should have 30 tokens
		res, err := rl.Allow(ctx, RateLimitRequest{Key: "u1", Cost: 10, CurrentTime: t0.Add(30 * time.Second)})
		assert.NoError(t, err)
		assert.True(t, res.Allowed)
		assert.Equal(t, uint(20), res.Remaining) // 30 (refill) - 10 (consumed)
	})
}

func TestGetShardKey(t *testing.T) {
	_, _ = NewTBLazyRefillRL(context.Background(), testRefillRate, time.Second, "seconds", testMaxTokenLimit, testShardCount)

	t.Run("should_produce_consistent_hash_for_same_key", func(t *testing.T) {
		key := "test-key"
		hash1 := xxhash.Sum64String(key)
		hash2 := xxhash.Sum64String(key)
		assert.Equal(t, hash1, hash2)
	})

	t.Run("should_produce_different_hashes_for_different_keys", func(t *testing.T) {
		hash1 := xxhash.Sum64String("key-1")
		hash2 := xxhash.Sum64String("key-2")
		assert.NotEqual(t, hash1, hash2)
	})

	t.Run("should_match_golden_values_(contract_test)", func(t *testing.T) {
		// These are specific values produced by xxhash for these strings.
		// Testing them ensures we don't accidentally switch hashing algorithms.
		assert.Equal(t, uint64(0xa173746b114c6be8), xxhash.Sum64String("user-1"))
		assert.Equal(t, uint64(0x7395dd9943ab55e9), xxhash.Sum64String("user-2"))
	})
}

func TestGetShard(t *testing.T) {
	rl, _ := NewTBLazyRefillRL(context.Background(), testRefillRate, time.Second, "seconds", testMaxTokenLimit, 4)

	t.Run("should_be_deterministic", func(t *testing.T) {
		key := "user-123"
		s1 := rl.getShard(key)
		s2 := rl.getShard(key)
		assert.Same(t, s1, s2)
	})

	t.Run("should_always_return_a_valid_shard_from_the_pool", func(t *testing.T) {
		keys := []string{"a", "b", "c", "d", "e", "f", "g", "h"}
		for _, k := range keys {
			s := rl.getShard(k)
			assert.NotNil(t, s, "Shard should not be nil for key %s", k)

			// Verify it's one of the shards in our list
			found := false
			for _, existingShard := range rl.shards {
				if existingShard == s {
					found = true
					break
				}
			}
			assert.True(t, found, "Shard should be part of the rate limiter's shard pool")
		}
	})
}

func TestGetOrCreate(t *testing.T) {
	rl, _ := NewTBLazyRefillRL(context.Background(), testRefillRate, time.Second, "seconds", 100, 1)
	s := rl.shards[0]
	now := time.Now()

	t.Run("should_create_new_tbClient_if_not_exists", func(t *testing.T) {
		s.clients = make(map[string]*tbClient)
		key := "new-user"

		c := rl.getOrCreateClient(s, key, now)
		assert.NotNil(t, c)
		assert.Equal(t, float64(100), c.tokens)
		assert.Equal(t, now, c.lastAccess)
		assert.Contains(t, s.clients, key)
	})

	t.Run("should_return_existing_tbClient_if_exists", func(t *testing.T) {
		key := "existing-user"
		existing := &tbClient{tokens: 50, lastAccess: now.Add(-time.Hour)}
		s.clients[key] = existing

		c := rl.getOrCreateClient(s, key, now)
		assert.Same(t, existing, c)
		assert.Equal(t, float64(50), c.tokens) // Should NOT have reset tokens
	})
}

func TestEviction(t *testing.T) {
	t.Run("should_evict_inactive_full_buckets", func(t *testing.T) {
		// Use a very short refill window for testing
		window := 50 * time.Millisecond
		rl, _ := NewTBLazyRefillRL(context.Background(), 10, window, "ms", 10, 1)
		s := rl.shards[0]

		ctx := context.Background()
		now := time.Now()
		key := "evict-me"

		// 1. Create a client by calling Allow
		rl.Allow(ctx, RateLimitRequest{Key: key, Cost: 1, CurrentTime: now})
		assert.Contains(t, s.clients, key)

		// 2. Wait for it to refill and exceed 2 * window
		// The eviction loop runs every 2 * window (100ms here)
		time.Sleep(250 * time.Millisecond)

		// 3. Check if evicted
		s.mu.Lock()
		_, exists := s.clients[key]
		s.mu.Unlock()
		assert.False(t, exists, "Client should have been evicted")
		
		rl.Close()
	})

	t.Run("should_not_evict_active_buckets", func(t *testing.T) {
		window := 100 * time.Millisecond
		rl, _ := NewTBLazyRefillRL(context.Background(), 10, window, "ms", 10, 1)
		s := rl.shards[0]

		ctx := context.Background()
		key := "keep-me"

		// Keep calling Allow to update lastAccess
		for i := 0; i < 5; i++ {
			rl.Allow(ctx, RateLimitRequest{Key: key, Cost: 1, CurrentTime: time.Now()})
			time.Sleep(50 * time.Millisecond)
		}

		s.mu.Lock()
		_, exists := s.clients[key]
		s.mu.Unlock()
		assert.True(t, exists, "Active client should NOT have been evicted")
		
		rl.Close()
	})
}
