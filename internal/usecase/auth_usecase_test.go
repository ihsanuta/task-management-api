package usecase_test

import (
	"context"
	"testing"
	"time"

	"github.com/ihsanuta/task-management-api/internal/delivery/http/dto"
	"github.com/ihsanuta/task-management-api/internal/repository/mock"
	"github.com/ihsanuta/task-management-api/internal/usecase"
	"github.com/ihsanuta/task-management-api/pkg/apperror"
	"github.com/ihsanuta/task-management-api/pkg/jwtutil"
)

func newAuthUC() *usecase.AuthUsecase {
	users := mock.NewUserRepository()
	teams := mock.NewTeamRepository()
	jwtManager := jwtutil.NewManager("test-secret", time.Hour)
	return usecase.NewAuthUsecase(users, teams, jwtManager)
}

func TestRegister_Success(t *testing.T) {
	uc := newAuthUC()
	res, err := uc.Register(context.Background(), dto.RegisterRequest{
		Name: "Alice", Email: "alice@example.com", Password: "supersecret1",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Token == "" {
		t.Fatal("expected a JWT token to be returned")
	}
	if res.User.Email != "alice@example.com" {
		t.Fatalf("unexpected email: %s", res.User.Email)
	}
}

func TestRegister_DuplicateEmail(t *testing.T) {
	uc := newAuthUC()
	ctx := context.Background()
	req := dto.RegisterRequest{Name: "Alice", Email: "dup@example.com", Password: "supersecret1"}
	if _, err := uc.Register(ctx, req); err != nil {
		t.Fatalf("first register should succeed: %v", err)
	}
	_, err := uc.Register(ctx, req)
	if err != apperror.ErrEmailAlreadyRegistered {
		t.Fatalf("expected ErrEmailAlreadyRegistered, got %v", err)
	}
}

func TestRegister_WeakPassword(t *testing.T) {
	uc := newAuthUC()
	_, err := uc.Register(context.Background(), dto.RegisterRequest{
		Name: "Bob", Email: "bob@example.com", Password: "short",
	})
	if err == nil {
		t.Fatal("expected validation error for short password")
	}
}

func TestLogin_Success(t *testing.T) {
	uc := newAuthUC()
	ctx := context.Background()
	_, _ = uc.Register(ctx, dto.RegisterRequest{Name: "Carol", Email: "carol@example.com", Password: "supersecret1"})

	res, err := uc.Login(ctx, dto.LoginRequest{Email: "carol@example.com", Password: "supersecret1"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Token == "" {
		t.Fatal("expected a token")
	}
}

func TestLogin_WrongPassword(t *testing.T) {
	uc := newAuthUC()
	ctx := context.Background()
	_, _ = uc.Register(ctx, dto.RegisterRequest{Name: "Dave", Email: "dave@example.com", Password: "supersecret1"})

	_, err := uc.Login(ctx, dto.LoginRequest{Email: "dave@example.com", Password: "wrongpassword"})
	if err != apperror.ErrInvalidCredentials {
		t.Fatalf("expected ErrInvalidCredentials, got %v", err)
	}
}
