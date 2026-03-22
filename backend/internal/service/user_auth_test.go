package service

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"strings"
	"testing"

	"github.com/rxritet/Specto/internal/domain"
)

const (
	authTestEmail         = "auth@example.com"
	authTestName          = "Auth User"
	errRegisterUserFailed = "register user: %v"
)

func TestUserServiceRegisterHashesPassword(t *testing.T) {
	ctx := context.Background()
	repo := newMemUserRepo()
	svc := NewUserService(repo, noopServiceLogger())

	user := &domain.User{
		Email:    authTestEmail,
		Name:     authTestName,
		Password: "password123",
	}

	if err := svc.Register(ctx, user); err != nil {
		t.Fatalf(errRegisterUserFailed, err)
	}
	if user.Password == "password123" {
		t.Fatal("expected password to be hashed")
	}
	if !strings.HasPrefix(user.Password, "$2") {
		t.Fatalf("expected bcrypt hash, got %q", user.Password)
	}
}

func TestUserServiceAuthenticateSuccess(t *testing.T) {
	ctx := context.Background()
	repo := newMemUserRepo()
	svc := NewUserService(repo, noopServiceLogger())

	user := &domain.User{
		Email:    authTestEmail,
		Name:     authTestName,
		Password: "password123",
	}
	if err := svc.Register(ctx, user); err != nil {
		t.Fatalf(errRegisterUserFailed, err)
	}

	authenticated, err := svc.Authenticate(ctx, user.Email, "password123")
	if err != nil {
		t.Fatalf("authenticate user: %v", err)
	}
	if authenticated.ID != user.ID {
		t.Fatalf("expected user id %d, got %d", user.ID, authenticated.ID)
	}
}

func TestUserServiceAuthenticateInvalidCredentials(t *testing.T) {
	ctx := context.Background()
	repo := newMemUserRepo()
	svc := NewUserService(repo, noopServiceLogger())

	user := &domain.User{
		Email:    authTestEmail,
		Name:     authTestName,
		Password: "password123",
	}
	if err := svc.Register(ctx, user); err != nil {
		t.Fatalf(errRegisterUserFailed, err)
	}

	_, err := svc.Authenticate(ctx, user.Email, "wrong-password")
	if err == nil {
		t.Fatal("expected authentication error")
	}
	if _, ok := errors.AsType[*domain.UnauthorizedError](err); !ok {
		t.Fatalf("expected unauthorized error, got %T: %v", err, err)
	}
}

func noopServiceLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}
