package service

import (
	"encoding/json"
	"errors"

	"github.com/arjunjgowda/rate-limitting/internal/domain"
	"gofr.dev/pkg/gofr"
)

type UserService struct {
	store     Store
	txManager TransactionManager // New dependency
	userTopic string
}

func NewUserService(s Store, tm TransactionManager, topic string) *UserService {
	return &UserService{
		store:     s,
		txManager: tm,
		userTopic: topic,
	}
}

func (s *UserService) Login(ctx *gofr.Context, username, password string) (string, error) {
	userID, err := s.store.VerifyCredentials(ctx, username, password)
	if err != nil {
		return "", err
	}
	return "token_for_" + userID, nil
}

func (s *UserService) GetBalance(ctx *gofr.Context, userID string) (float64, error) {
	return s.store.GetBalance(ctx, userID)
}

func (s *UserService) GetUserInfo(ctx *gofr.Context, userID string) (*domain.User, error) {
	return s.store.GetUser(ctx, userID)
}

func (s *UserService) CreateUser(ctx *gofr.Context, user *domain.User) (string, error) {
	// Trigger the Domain Validation
	if err := user.Validate(); err != nil {
		return "", err
	}

	userID, err := s.store.CreateUser(ctx, user)
	if err != nil {
		return "", err
	}

	// Publish Event to Kafka
	// 1. Prepare message using domain event struct
	event := domain.NewUserCreatedEvent(userID, user.Username)
	msg, _ := json.Marshal(event)

	// 2. Publish using the PubSub client
	err = ctx.PubSub.Publish(ctx, s.userTopic, msg)

	if err != nil {
		ctx.Logger.Errorf("failed to publish user-created event to topic %s: %v. Saving to outbox.", s.userTopic, err)

		_ = s.store.SaveToOutbox(ctx, s.userTopic, msg)
	}
	return userID, nil
}

func (s *UserService) Transfer(ctx *gofr.Context, fromID, toID string, amount float64) error {
	// 1. Basic Validation
	if amount <= 0 {
		return errors.New("transfer amount must be positive")
	}
	if fromID == toID {
		return errors.New("cannot transfer to self")
	}

	// 2. Execute within an abstract transaction
	return s.txManager.WithTransaction(ctx, func(txCtx *gofr.Context) error {
		// 3. Fetch and Lock both users in consistent order (prevents deadlocks)
		firstID := fromID
		if fromID > toID {
			firstID = toID
		}

		var fromUser, toUser *domain.User
		var err error

		if firstID == fromID {
			fromUser, err = s.store.GetForUpdate(txCtx, fromID)
			if err != nil {
				return err
			}
			toUser, err = s.store.GetForUpdate(txCtx, toID)
			if err != nil {
				return err
			}
		} else {
			toUser, err = s.store.GetForUpdate(txCtx, toID)
			if err != nil {
				return err
			}
			fromUser, err = s.store.GetForUpdate(txCtx, fromID)
			if err != nil {
				return err
			}
		}

		// 4. REUSE Domain Logic
		if err := fromUser.Withdraw(amount); err != nil {
			return err
		}
		toUser.Deposit(amount)

		// 5. Persist Changes
		if err := s.store.UpdateBalance(txCtx, fromID, fromUser.Balance); err != nil {
			return err
		}
		return s.store.UpdateBalance(txCtx, toID, toUser.Balance)
	})
}
