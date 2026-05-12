package ratelimit

import (
	"sync"
	"time"

	"github.com/cespare/xxhash/v2"
)

// tbClient stores the token state for a single client
type tbClient struct {
	tokens     float64   // float64 for precision math in lazy refill
	lastAccess time.Time // Used for eviction and retry calculations
}

// tbShard groups clients to reduce lock contention
type tbShard struct {
	mu      sync.Mutex
	clients map[string]*tbClient
}

// tbBase contains common fields and methods for all token bucket variants
type tbBase struct {
	shards        []*tbShard
	shardCount    uint
	maxTokenLimit uint
	refillRate    uint
	refillWindow  time.Duration
	windowUnit    string
}

func newTBBase(maxTokenLimit, refillRate, shardCount uint, refillWindow time.Duration, windowUnit string) tbBase {
	shards := make([]*tbShard, shardCount)
	for i := range shardCount {
		shards[i] = &tbShard{clients: make(map[string]*tbClient)}
	}

	return tbBase{
		shards:        shards,
		shardCount:    shardCount,
		maxTokenLimit: maxTokenLimit,
		refillRate:    refillRate,
		refillWindow:  refillWindow,
		windowUnit:    windowUnit,
	}
}

func (b *tbBase) getShard(key string) *tbShard {
	hash := xxhash.Sum64String(key)
	return b.shards[hash&(uint64(b.shardCount)-1)]
}

func (b *tbBase) getOrCreateClient(s *tbShard, key string, now time.Time) *tbClient {
	if c, ok := s.clients[key]; ok {
		return c
	}

	c := &tbClient{
		tokens:     float64(b.maxTokenLimit),
		lastAccess: now,
	}
	s.clients[key] = c
	return c
}
