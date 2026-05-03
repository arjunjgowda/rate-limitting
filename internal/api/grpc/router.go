package grpc

import (
	"github.com/arjunjgowda/rate-limitting/internal/api/grpc/handlers"
	"github.com/arjunjgowda/rate-limitting/internal/api/grpc/interceptors"
	"github.com/arjunjgowda/rate-limitting/internal/api/grpc/pb"
	"github.com/arjunjgowda/rate-limitting/internal/service"
	"github.com/arjunjgowda/rate-limitting/pkg/ratelimit"
	"gofr.dev/pkg/gofr"
)

// Option defines a functional option for the gRPC router.
type Option func(*config)

type config struct {
	rateLimiter ratelimit.RateLimiter
}

// WithRateLimiter is a functional option to inject a rate limiter.
func WithRateLimiter(rl ratelimit.RateLimiter) Option {
	return func(c *config) {
		c.rateLimiter = rl
	}
}

// RegisterHandlers sets up the gRPC service and its interceptors.
func RegisterHandlers(app *gofr.App, svc service.Service, opts ...Option) {
	// 1. Process Options
	cfg := &config{}
	for _, opt := range opts {
		opt(cfg)
	}

	// 2. Initialize Handlers
	userHandler := handlers.NewUserHandler(svc)

	// 3. Register gRPC Interceptors
	if cfg.rateLimiter != nil {
		app.AddGRPCUnaryInterceptors(interceptors.RateLimiter(cfg.rateLimiter))
	}

	// 4. Register gRPC Service
	// Register the service using GoFr's RegisterService method
	app.RegisterService(&pb.UserService_ServiceDesc, userHandler)
}
