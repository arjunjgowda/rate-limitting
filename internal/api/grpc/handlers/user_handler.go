package handlers

import (
	"context"
	"fmt"
	"github.com/arjunjgowda/rate-limitting/internal/api/grpc/pb"
	"github.com/arjunjgowda/rate-limitting/internal/service"
	"gofr.dev/pkg/gofr"
)

type UserHandler struct {
	pb.UnimplementedUserServiceServer
	svc service.Service
}

func NewUserHandler(svc service.Service) *UserHandler {
	return &UserHandler{svc: svc}
}

func (h *UserHandler) Login(ctx context.Context, req *pb.LoginRequest) (*pb.LoginResponse, error) {
	// GoFr's gRPC implementation passes *gofr.Context as context.Context
	gofrCtx, ok := ctx.(*gofr.Context)
	if !ok {
		return nil, fmt.Errorf("invalid context type")
	}

	// Call business logic service
	// Note: Our service.Login returns (string, error) where string is the token/ID
	token, err := h.svc.Login(gofrCtx, req.Username, req.Password)
	if err != nil {
		return nil, err
	}

	return &pb.LoginResponse{
		Id:       "unknown", // In a real app, we'd return user info
		Username: req.Username,
		Token:    token,
	}, nil
}
