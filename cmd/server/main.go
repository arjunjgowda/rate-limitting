package main

import (
	"github.com/arjunjgowda/rate-limitting/internal/api/grpc"
	"github.com/arjunjgowda/rate-limitting/internal/api/http"
	"github.com/arjunjgowda/rate-limitting/internal/migrations"
	"github.com/arjunjgowda/rate-limitting/internal/service"
	"github.com/arjunjgowda/rate-limitting/internal/store"
	"github.com/arjunjgowda/rate-limitting/pkg/ratelimit"
	"gofr.dev/pkg/gofr"
)

func main() {
	app := gofr.New()

	// 1. Initialize Infrastructure (DB Store)
	userStore := store.NewUserStore()
	txnManager := store.NewTransactionManager()

	// 2. Initialize Business Logic
	userTopic := app.Config.GetOrDefault("KAFKA_USER_TOPIC", "system-design")
	userService := service.NewUserService(userStore, txnManager, userTopic)

	var rl ratelimit.RateLimiter

	// 4. Register Protocol Handlers
	http.RegisterHandlers(app, userService,
		http.WithRateLimiter(rl),
	)

	grpc.RegisterHandlers(app, userService,
		grpc.WithRateLimiter(rl),
	)

	// 5. Run Database Migrations
	app.Migrate(migrations.All())

	// 6. Start Server
	app.Run()
}
