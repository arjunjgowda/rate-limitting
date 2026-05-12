package handlers

import (
	"errors"

	"github.com/arjunjgowda/rate-limitting/internal/domain"
	"github.com/arjunjgowda/rate-limitting/internal/service"
	"github.com/arjunjgowda/rate-limitting/pkg/validator"
	"gofr.dev/pkg/gofr"
	"gofr.dev/pkg/gofr/http"
	gofrHTTP "gofr.dev/pkg/gofr/http"
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
		return nil, gofrHTTP.ErrorMissingParam{Params: []string{"username", "password"}}
	}

	res, err := h.userService.Login(ctx, req.Username, req.Password)
	if err != nil {
		if errors.Is(err, domain.ErrInvalidCredentials) {
			return nil, gofrHTTP.ErrorInvalidParam{Params: []string{"username/password"}}
		}
		return nil, err
	}

	return res, nil
}

func (h *UserHandler) CheckBalance(ctx *gofr.Context) (interface{}, error) {
	// Reading from query parameter ?id=...
	userID := ctx.Param("id")
	if userID == "" {
		return nil, gofrHTTP.ErrorMissingParam{Params: []string{"id"}}
	}

	if !validator.IsUUID(userID) {
		return nil, gofrHTTP.ErrorInvalidParam{Params: []string{"id"}}
	}

	res, err := h.userService.GetBalance(ctx, userID)
	if err != nil {
		if errors.Is(err, domain.ErrUserNotFound) {
			return nil, gofrHTTP.ErrorEntityNotFound{Name: "user", Value: userID}
		}
		return nil, err
	}

	return res, nil
}

func (h *UserHandler) GetUserInfo(ctx *gofr.Context) (interface{}, error) {
	userID := ctx.Request.PathParam("id")

	if !validator.IsUUID(userID) {
		return nil, gofrHTTP.ErrorInvalidParam{Params: []string{"id"}}
	}

	res, err := h.userService.GetUserInfo(ctx, userID)
	if err != nil {
		if errors.Is(err, domain.ErrUserNotFound) {
			return nil, gofrHTTP.ErrorEntityNotFound{Name: "user", Value: userID}
		}
		return nil, err
	}

	return res, nil
}

func (h *UserHandler) CreateUser(ctx *gofr.Context) (interface{}, error) {
	var user domain.User

	if err := ctx.Bind(&user); err != nil {
		return nil, err
	}

	if user.Username == "" {
		return nil, gofrHTTP.ErrorMissingParam{Params: []string{"username"}}
	}
	if user.Email == "" {
		return nil, gofrHTTP.ErrorMissingParam{Params: []string{"email"}}
	}

	res, err := h.userService.CreateUser(ctx, &user)
	if err != nil {
		if errors.Is(err, domain.ErrUserAlreadyExists) {
			return nil, gofrHTTP.ErrorEntityAlreadyExist{}
		}
		return nil, err
	}

	return res, nil
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
		return nil, http.ErrorInvalidParam{Params: []string{"fromId", "toId"}}
	}

	if req.Amount <= 0 {
		return nil, http.ErrorInvalidParam{Params: []string{"amount"}}
	}

	err := h.userService.Transfer(ctx, req.FromID, req.ToID, req.Amount)
	if err != nil {
		return nil, err
	}

	return map[string]string{"status": "success", "message": "transfer completed"}, nil
}
