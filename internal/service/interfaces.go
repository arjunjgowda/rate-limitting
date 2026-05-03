package service

import (
	"github.com/arjunjgowda/rate-limitting/internal/domain"
	"gofr.dev/pkg/gofr"
)

// Store defines the contract for data persistence.
type Store interface {
	VerifyCredentials(ctx *gofr.Context, username, password string) (string, error)
	GetBalance(ctx *gofr.Context, userID string) (float64, error)
	GetUser(ctx *gofr.Context, userID string) (*domain.User, error)
	CreateUser(ctx *gofr.Context, user *domain.User) (string, error)
	SaveToOutbox(ctx *gofr.Context, topic string, payload []byte) error
	UpdateBalance(ctx *gofr.Context, userID string, amount float64) error
	GetForUpdate(ctx *gofr.Context, userID string) (*domain.User, error)
}

// TransactionManager abstracts away the DB-specific transaction logic.
type TransactionManager interface {
	WithTransaction(ctx *gofr.Context, fn func(ctx *gofr.Context) error) error
}

// Service defines the contract that API handlers consume.
type Service interface {
	Login(ctx *gofr.Context, username, password string) (string, error)
	GetBalance(ctx *gofr.Context, userID string) (float64, error)
	GetUserInfo(ctx *gofr.Context, userID string) (*domain.User, error)
	CreateUser(ctx *gofr.Context, user *domain.User) (string, error)
	Transfer(ctx *gofr.Context, fromID, toID string, amount float64) error
}
