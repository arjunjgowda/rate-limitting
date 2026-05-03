package interceptors

import (
	"context"
	"time"

	"github.com/arjunjgowda/rate-limitting/pkg/ratelimit"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"
)

// RateLimiter returns a gRPC UnaryServerInterceptor that enforces rate limits.
func RateLimiter(rl ratelimit.RateLimiter) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		// 1. Identify the caller (Key)
		// For gRPC, we can use the peer's IP address
		key := "unknown"
		if p, ok := peer.FromContext(ctx); ok {
			key = p.Addr.String()
		}

		// 2. Prepare the rate limit request
		rlReq := ratelimit.RateLimitRequest{
			Key:         key,
			Cost:        1,
			CurrentTime: time.Now(),
		}

		// 3. Ask the RateLimiter
		result, err := rl.Allow(ctx, rlReq)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "rate limit check failed: %v", err)
		}

		// 4. Reject if not allowed
		if !result.Allowed {
			return nil, status.Errorf(codes.ResourceExhausted, "rate limit exceeded. try again in %v", result.RetryAfter)
		}

		// 5. Proceed with the actual RPC call
		return handler(ctx, req)
	}
}
