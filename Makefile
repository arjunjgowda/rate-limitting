# Build and run targets
.PHONY: all build run test clean generate swagger refresh benchmark

all: build

build: generate
	go build -o server cmd/server/main.go

run: build
	./server

test:
	go test ./...

clean:
	rm -f server
	rm -rf internal/api/grpc/pb/*

# gRPC Code Generation
generate:
	@echo "Generating gRPC code..."
	@mkdir -p internal/api/grpc/pb
	@protoc --go_out=. --go-grpc_out=. api/proto/user.proto
	@mv github.com/arjunjgowda/rate-limitting/internal/api/grpc/pb/* internal/api/grpc/pb/
	@rm -rf github.com

# Swagger Sync (YAML to JSON)
swagger:
	@echo "Syncing Swagger JSON..."
	@mkdir -p static
	@npx -y yq -o json api/openapi/openapi.yaml > static/openapi.json

# Unified refresh target for all contracts
refresh: generate swagger
	@echo "All contracts refreshed (gRPC, Swagger)"

# Benchmarking
REQS ?= 100000000
USERS ?= 1000000

benchmark:
	@echo "Running all benchmark variations (${USERS} users, ${REQS} requests each)..."
	@go run examples/comparison/main.go --algo lazy --test-mode 1 --run-id lazy-m1 --users ${USERS} --requests ${REQS}
	@go run examples/comparison/main.go --algo fixed --test-mode 1 --run-id fixed-m1 --users ${USERS} --requests ${REQS}
	@go run examples/comparison/main.go --algo lazy --test-mode 2 --run-id lazy-m2 --users ${USERS} --requests ${REQS}
	@go run examples/comparison/main.go --algo fixed --test-mode 2 --run-id fixed-m2 --users ${USERS} --requests ${REQS}
	@echo "Benchmarks complete. Results stored in examples/comparison/results.txt"
