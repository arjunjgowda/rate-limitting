package handlers

import (
	"errors"

	"github.com/arjunjgowda/rate-limitting/internal/domain"
	"github.com/arjunjgowda/rate-limitting/internal/service"
	"github.com/arjunjgowda/rate-limitting/pkg/validator"
	"gofr.dev/pkg/gofr"
)

type UserHandler struct {
	userService service.Service
}

func NewUserHandler(s service.Service) *UserHandler {
	return &UserHandler{userService: s}
}

func (h *UserHandler) Login(ctx *gofr.Context) (interface{}, error) {
	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}

	if err := ctx.Bind(&req); err != nil {
		return nil, err
	}

	if req.Username == "" || req.Password == "" {
		return nil, errors.New("username and password are required")
	}

	return h.userService.Login(ctx, req.Username, req.Password)
}

func (h *UserHandler) CheckBalance(ctx *gofr.Context) (interface{}, error) {
	// Reading from query parameter ?id=...
	userID := ctx.Param("id")
	if userID == "" {
		return nil, errors.New("user id is required")
	}

	if !validator.IsUUID(userID) {
		return nil, errors.New("invalid user id format: expected UUID")
	}

	return h.userService.GetBalance(ctx, userID)
}

func (h *UserHandler) GetUserInfo(ctx *gofr.Context) (interface{}, error) {
	userID := ctx.Param("id")

	if !validator.IsUUID(userID) {
		return nil, errors.New("invalid user id format: expected UUID")
	}

	return h.userService.GetUserInfo(ctx, userID)
}

func (h *UserHandler) CreateUser(ctx *gofr.Context) (interface{}, error) {
	var user domain.User

	if err := ctx.Bind(&user); err != nil {
		return nil, err
	}

	if user.Username == "" {
		return nil, errors.New("username is required")
	}
	if user.Email == "" {
		return nil, errors.New("email is required")
	}

	return h.userService.CreateUser(ctx, &user)
}

func (h *UserHandler) Transfer(ctx *gofr.Context) (interface{}, error) {
	var req struct {
		FromID string  `json:"fromId"`
		ToID   string  `json:"toId"`
		Amount float64 `json:"amount"`
	}

	if err := ctx.Bind(&req); err != nil {
		return nil, err
	}

	if !validator.IsUUID(req.FromID) || !validator.IsUUID(req.ToID) {
		return nil, errors.New("invalid user id format: expected UUID")
	}

	if req.Amount <= 0 {
		return nil, errors.New("amount must be greater than zero")
	}

	err := h.userService.Transfer(ctx, req.FromID, req.ToID, req.Amount)
	if err != nil {
		return nil, err
	}

	return map[string]string{"status": "success", "message": "transfer completed"}, nil
}
