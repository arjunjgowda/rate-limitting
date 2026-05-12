package ratelimit

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"gofr.dev/pkg/gofr/config"
)

const (
	defaultMaxTokenLimit string = "100"
	defaultRefillRate     string = "3"
	defaultShardCount    string = "64"
)

// tbConfig holds common configuration for token bucket variations
type tbConfig struct {
	refillRate    uint
	maxTokenLimit uint
	shardCount    uint
	window        time.Duration
	windowStr     string
}

var factories = map[string]func(context.Context, config.Config) (RateLimiter, error){
	"token_bucket_lazy_refill":     buildTBLazyRefill,
	"token_bucket_fixed_interval": buildTBFixedInterval,
}

func NewRateLimiter(ctx context.Context, conf config.Config) (RateLimiter, error) {
	algo := conf.GetOrDefault("RATE_LIMIT_ALGO", "token_bucket_lazy_refill")

	builder, exists := factories[algo]
	if !exists {
		return nil, fmt.Errorf("unknown rate limit algorithm: %s", algo)
	}
	return builder(ctx, conf)
}

func buildTBLazyRefill(ctx context.Context, conf config.Config) (RateLimiter, error) {
	c, err := parseTBConfig(conf)
	if err != nil {
		return nil, err
	}

	return NewTBLazyRefillRL(ctx, c.refillRate, c.window, c.windowStr, c.maxTokenLimit, c.shardCount)
}

func buildTBFixedInterval(ctx context.Context, conf config.Config) (RateLimiter, error) {
	c, err := parseTBConfig(conf)
	if err != nil {
		return nil, err
	}

	return NewTBFixedCounter(ctx, c.refillRate, c.window, c.windowStr, c.maxTokenLimit, c.shardCount)
}

func parseTBConfig(conf config.Config) (*tbConfig, error) {
	rr, err := getUintConfig(conf, "REFILL_RATE", defaultRefillRate)
	if err != nil {
		return nil, err
	}
	mtl, err := getUintConfig(conf, "MAX_TOKEN_LIMIT", defaultMaxTokenLimit)
	if err != nil {
		return nil, err
	}
	sc, err := getUintConfig(conf, "SHARD_COUNT", defaultShardCount)
	if err != nil {
		return nil, err
	}

	windowStr := conf.GetOrDefault("REFILL_WINDOW", "seconds")
	window := parseWindow(windowStr)

	return &tbConfig{
		refillRate:    rr,
		maxTokenLimit: mtl,
		shardCount:    sc,
		window:        window,
		windowStr:     windowStr,
	}, nil
}

func parseWindow(windowStr string) time.Duration {
	switch strings.ToLower(windowStr) {
	case "seconds", "sec", "s":
		return time.Second
	case "minutes", "min", "m":
		return time.Minute
	case "hours", "hr", "h":
		return time.Hour
	case "days", "d":
		return 24 * time.Hour
	default:
		return time.Second
	}
}

func getUintConfig(conf config.Config, key, defaultVal string) (uint, error) {
	valStr := conf.GetOrDefault(key, defaultVal)
	val, err := strconv.ParseUint(valStr, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid %s: %w", key, err)
	}

	if val > uint64(^uint(0)) {
		return 0, fmt.Errorf("%s overflows uint", key)
	}

	return uint(val), nil
}
