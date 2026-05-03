package http

import (
	"github.com/arjunjgowda/rate-limitting/internal/api/http/handlers"
	"github.com/arjunjgowda/rate-limitting/internal/api/http/middleware"
	"github.com/arjunjgowda/rate-limitting/internal/service"
	"github.com/arjunjgowda/rate-limitting/pkg/ratelimit"
	"gofr.dev/pkg/gofr"
)

// Option defines a functional option for the HTTP router.
type Option func(*config)

type config struct {
	rateLimiter ratelimit.RateLimiter
	// Add future optional dependencies here (e.g. auth, metrics)
}

// WithRateLimiter is a functional option to inject a rate limiter.
func WithRateLimiter(rl ratelimit.RateLimiter) Option {
	return func(c *config) {
		c.rateLimiter = rl
	}
}

// RegisterHandlers sets up all HTTP routes and middlewares.
func RegisterHandlers(app *gofr.App, svc service.Service, opts ...Option) {
	// 1. Process Options
	cfg := &config{}
	for _, opt := range opts {
		opt(cfg)
	}

	// 2. Initialize Handlers
	userHandler := handlers.NewUserHandler(svc)

	// 3. Register Global Middlewares
	app.UseMiddleware(middleware.RequestLogger(app.Logger()))
	
	if cfg.rateLimiter != nil {
		app.UseMiddleware(middleware.RateLimiter(cfg.rateLimiter))
	}

	// 4. Register HTTP Routes
	app.POST("/login", userHandler.Login)
	app.GET("/check-balance", userHandler.CheckBalance)
	app.GET("/user/{id}", userHandler.GetUserInfo)
	app.POST("/user", userHandler.CreateUser)
	app.POST("/transfer", userHandler.Transfer)
}
