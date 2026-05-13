# Token Bucket Comparison Benchmark

This tool compares **Lazy Refill** and **Fixed Interval** Token Bucket strategies under high-load scenarios.

## Test Modes

| Mode | Name | Description |
|------|------|-------------|
| `1` | **Pre-populated** | Pre-creates 1M users (ensures map is fully allocated) then sends 5M random requests. |
| `2` | **Purely Random** | Sends 5M random requests directly. Users are created lazily upon first access. |

## How to Run

Results are automatically appended to `examples/comparison/results.txt`.

### Run All Benchmarks
```bash
# Default (1M users, 50M requests)
make benchmark

# Custom parameters
make benchmark USERS=500000 REQS=10000000
```

### Mode 1: Pre-populated (Recommended for Latency testing)
```bash
go run examples/comparison/main.go --algo lazy --test-mode 1 --run-id lazy-mode-1
go run examples/comparison/main.go --algo fixed --test-mode 1 --run-id fixed-mode-1
```

### Mode 2: Purely Random (Tests creation + refill overhead)
```bash
go run examples/comparison/main.go --algo lazy --test-mode 2 --run-id lazy-mode-2
go run examples/comparison/main.go --algo fixed --test-mode 2 --run-id fixed-mode-2
```

## Configuration Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--run-id` | `run-HHMMSS` | Unique identifier for the benchmark run |
| `--test-mode` | `1` | `1` (Pre-populate) or `2` (Pure Random) |
| `--users` | `1000000` | Size of the user pool |
| `--requests` | `5000000` | Total requests to send |
| `--shards` | `64` | Number of shards (power of 2) |
| `--algo` | `lazy` | `lazy` or `fixed` |
| `--output` | `results.txt` | File to store results |

## Results Format

The `results.txt` file stores tab-separated entries:
`[Timestamp] [Run ID] [Algo] [Mode] [Users] [Requests] [Shards] [Avg Latency] [Memory Allocation]`
