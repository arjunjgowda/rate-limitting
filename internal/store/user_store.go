package store

import (
	"database/sql"
	"errors"
	"strings"

	"github.com/arjunjgowda/rate-limitting/internal/domain"
	"github.com/arjunjgowda/rate-limitting/internal/service"
	"gofr.dev/pkg/gofr"
)

// Interface guard
var _ service.Store = (*UserStore)(nil)

type UserStore struct{}

func NewUserStore() *UserStore {
	return &UserStore{}
}

func (s *UserStore) GetBalance(ctx *gofr.Context, userID string) (float64, error) {
	var balance float64
	query := "SELECT balance FROM users WHERE id = $1"

	err := ctx.SQL.QueryRowContext(ctx, query, userID).Scan(&balance)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0, domain.ErrUserNotFound
		}
		return 0, err
	}

	return balance, nil
}

func (s *UserStore) VerifyCredentials(ctx *gofr.Context, username, password string) (string, error) {
	var userID string
	query := "SELECT id FROM users WHERE username = $1 AND password = $2"

	err := ctx.SQL.QueryRowContext(ctx, query, username, password).Scan(&userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", domain.ErrInvalidCredentials
		}
		return "", err
	}

	return userID, nil
}

func (s *UserStore) GetUser(ctx *gofr.Context, userID string) (*domain.User, error) {
	var u domain.User
	query := "SELECT id, username, balance, email FROM users WHERE id = $1"

	err := ctx.SQL.QueryRowContext(ctx, query, userID).Scan(&u.ID, &u.Username, &u.Balance, &u.Email)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrUserNotFound
		}
		return nil, err
	}

	return &u, nil
}

func (s *UserStore) GetForUpdate(ctx *gofr.Context, userID string) (*domain.User, error) {
	var u domain.User
	query := "SELECT id, balance FROM users WHERE id = $1 FOR UPDATE"

	// GoFr's ctx.SQL automatically uses the transaction if one is started on the context.
	err := ctx.SQL.QueryRowContext(ctx, query, userID).Scan(&u.ID, &u.Balance)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrUserNotFound
		}
		return nil, err
	}

	return &u, nil
}

func (s *UserStore) CreateUser(ctx *gofr.Context, user *domain.User) (string, error) {
	query := "INSERT INTO users (id, username, password, balance, email) VALUES ($1, $2, $3, $4, $5) RETURNING id"

	var id string
	err := ctx.SQL.QueryRowContext(ctx, query, user.ID, user.Username, "initial_pass", user.Balance, user.Email).Scan(&id)
	if err != nil {
		// Check for duplicate key violation (Postgres code 23505)
		if strings.Contains(err.Error(), "duplicate key") || strings.Contains(err.Error(), "23505") {
			return "", domain.ErrUserAlreadyExists
		}
		return "", err
	}

	return id, nil
}

func (s *UserStore) SaveToOutbox(ctx *gofr.Context, topic string, payload []byte) error {
	query := "INSERT INTO outbox (topic, payload) VALUES ($1, $2)"
	_, err := ctx.SQL.ExecContext(ctx, query, topic, payload)
	return err
}

func (s *UserStore) UpdateBalance(ctx *gofr.Context, userID string, balance float64) error {
	query := "UPDATE users SET balance = $1 WHERE id = $2"
	_, err := ctx.SQL.ExecContext(ctx, query, balance, userID)
	return err
}
