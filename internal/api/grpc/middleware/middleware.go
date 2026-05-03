package middleware

import (
	"context"
	"time"

	"github.com/arjunjgowda/rate-limitting/internal/api/common"
	"gofr.dev/pkg/gofr/logging"
	"google.golang.org/grpc"
)

// LoggingInterceptor returns a gRPC interceptor that uses GoFr's logging.Logger.
func LoggingInterceptor(logger logging.Logger) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		start := time.Now()

		resp, err := handler(ctx, req)

		// Pass the official GoFr logging.Logger to the shared logic
		common.LogRequest(logger, "gRPC", info.FullMethod, time.Since(start))

		return resp, err
	}
}
