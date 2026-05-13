package main

import (
	"context"
	"flag"
	"fmt"
	"math/rand"
	"os"
	"runtime"
	"time"

	"github.com/arjunjgowda/rate-limitting/pkg/ratelimit"
)

func main() {
	runID := flag.String("run-id", "run-"+time.Now().Format("150405"), "Unique ID for this run")
	userCount := flag.Int("users", 1000000, "Number of unique users in the pool")
	requestCount := flag.Int("requests", 5000000, "Total number of requests to simulate")
	shardCount := flag.Uint("shards", 64, "Number of shards (power of 2)")
	algo := flag.String("algo", "lazy", "Algorithm: 'lazy' or 'fixed'")
	mode := flag.Int("test-mode", 1, "Mode: 1 (Pre-populate then 5M reqs) or 2 (5M random reqs)")
	outputFile := flag.String("output", "examples/comparison/results.txt", "File to append results")
	flag.Parse()

	fmt.Println("==================================================")
	fmt.Printf("TOKEN BUCKET BENCHMARK: %s\n", *runID)
	fmt.Printf("Algorithm: %s | Mode: %d | Users: %d | Requests: %d\n", *algo, *mode, *userCount, *requestCount)
	fmt.Println("==================================================")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var rl ratelimit.RateLimiter
	var err error

	// 1. Initialize
	if *algo == "lazy" {
		rl, err = ratelimit.NewTBLazyRefillRL(ctx, 3, time.Second, "seconds", 100, *shardCount)
	} else {
		rl, err = ratelimit.NewTBFixedCounter(ctx, 3, time.Second, "seconds", 100, *shardCount)
	}
	if err != nil {
		panic(err)
	}
	defer rl.Close()

	// 2. Pre-populate if Mode 1
	if *mode == 1 {
		fmt.Printf("Mode 1: Pre-populating %d users...\n", *userCount)
		for i := 0; i < *userCount; i++ {
			key := fmt.Sprintf("user-%d", i)
			_, _ = rl.Allow(ctx, ratelimit.RateLimitRequest{
				Key:         key,
				Cost:        1,
				CurrentTime: time.Now(),
			})
			if i > 0 && i%500000 == 0 {
				fmt.Printf("  ... %d users created\n", i)
			}
		}
	}

	// 3. Execute Benchmarking Loop
	fmt.Printf("Sending %d random requests across user pool...\n", *requestCount)
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	
	start := time.Now()
	for i := 0; i < *requestCount; i++ {
		userID := r.Intn(*userCount)
		key := fmt.Sprintf("user-%d", userID)
		_, _ = rl.Allow(ctx, ratelimit.RateLimitRequest{
			Key:         key,
			Cost:        1,
			CurrentTime: time.Now(),
		})
		
		if i > 0 && i%1000000 == 0 {
			fmt.Printf("  ... %d requests processed\n", i)
		}
	}
	elapsed := time.Since(start)
	avgLatency := elapsed / time.Duration(*requestCount)

	// 4. Collect Stats
	fmt.Println("Performing Final GC to get clean baseline...")
	runtime.GC() // Force GC
	
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	finalMem := bToMb(m.Alloc)

	fmt.Printf("\nBenchmark Complete:\n")
	fmt.Printf("  Total Time:  %v\n", elapsed)
	fmt.Printf("  Avg Latency: %v\n", avgLatency)
	fmt.Printf("  Final Mem:   %d MiB (After GC)\n", finalMem)

	// 5. Log to File
	logResults(*outputFile, *runID, *algo, *mode, *userCount, *requestCount, *shardCount, elapsed, avgLatency, finalMem, bToMb(m.TotalAlloc), bToMb(m.Sys), m.NumGC)
	fmt.Printf("\nResults appended to %s\n", *outputFile)
	fmt.Println("==================================================")
}

func logResults(file, runID, algo string, mode, users, requests int, shards uint, totalTime, latency time.Duration, mem, totalAlloc, sys uint64, numGC uint32) {
	f, err := os.OpenFile(file, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Printf("!! Error opening result file: %v\n", err)
		return
	}
	defer f.Close()

	timestamp := time.Now().Format(time.RFC3339)
	line := fmt.Sprintf("%s\tID:%s\tAlgo:%-5s\tMode:%d\tUsers:%d\tReqs:%d\tShards:%d\tTime:%-10v\tLat:%-8v\tMem:%dMiB\tTotAlloc:%dMiB\tSys:%dMiB\tGCs:%d\n",
		timestamp, runID, algo, mode, users, requests, shards, totalTime, latency, mem, totalAlloc, sys, numGC)

	_, _ = f.WriteString(line)
}

func bToMb(b uint64) uint64 {
	return b / 1024 / 1024
}
