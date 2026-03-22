package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"

	"github.com/rxritet/Specto/internal/domain"
	"golang.org/x/crypto/bcrypt"
)

const emptyValueMessage = "must not be empty"
const invalidCredentialsMessage = "invalid credentials"

// UserService encapsulates business logic around users.
type UserService struct {
	repo   domain.UserRepository
	logger *slog.Logger
}

// NewUserService returns a ready-to-use UserService.
func NewUserService(repo domain.UserRepository, logger *slog.Logger) *UserService {
	return &UserService{repo: repo, logger: logger}
}

// Register validates user input, hashes the password and creates the user.
func (s *UserService) Register(ctx context.Context, user *domain.User) error {
	if err := s.validateRegistration(user); err != nil {
		return err
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(user.Password), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("hash password: %w", err)
	}

	user.Password = string(hash)
	return s.Create(ctx, user)
}

// Authenticate checks user credentials and returns the authenticated user.
func (s *UserService) Authenticate(ctx context.Context, email, password string) (*domain.User, error) {
	email = strings.TrimSpace(email)
	if email == "" || password == "" {
		return nil, domain.NewUnauthorizedError(invalidCredentialsMessage)
	}

	user, err := s.repo.GetByEmail(ctx, email)
	if err != nil {
		return nil, domain.NewUnauthorizedError(invalidCredentialsMessage)
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password)); err != nil {
		return nil, domain.NewUnauthorizedError(invalidCredentialsMessage)
	}

	return user, nil
}

// Create validates the input and persists a new user.
func (s *UserService) Create(ctx context.Context, user *domain.User) error {
	if err := s.validateUser(user); err != nil {
		return err
	}

	// Guard against duplicate email.
	if existing, _ := s.repo.GetByEmail(ctx, user.Email); existing != nil {
		return domain.NewConflictError("user", fmt.Sprintf("email %q already exists", user.Email))
	}

	if err := s.repo.Create(ctx, user); err != nil {
		return fmt.Errorf("create user: %w", err)
	}

	s.logger.Info("user created", "id", user.ID, "email", user.Email)
	return nil
}

// GetByID returns a single user or a NotFoundError.
func (s *UserService) GetByID(ctx context.Context, id int64) (*domain.User, error) {
	user, err := s.repo.GetByID(ctx, id)
	if err != nil {
		// Wrap a generic repo error into a typed NotFoundError so callers
		// can use errors.AsType[*domain.NotFoundError](err).
		if _, ok := errors.AsType[*domain.NotFoundError](err); ok {
			return nil, err
		}
		return nil, domain.NewNotFoundError("user", fmt.Sprintf("id=%d", id))
	}
	return user, nil
}

// GetByEmail returns a single user looked up by email.
func (s *UserService) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	user, err := s.repo.GetByEmail(ctx, email)
	if err != nil {
		if _, ok := errors.AsType[*domain.NotFoundError](err); ok {
			return nil, err
		}
		return nil, domain.NewNotFoundError("user", email)
	}
	return user, nil
}

// Update validates and persists changes to an existing user.
func (s *UserService) Update(ctx context.Context, user *domain.User) error {
	if user.ID == 0 {
		return domain.NewValidationError("id", "must be set")
	}

	if err := s.validateUser(user); err != nil {
		return err
	}

	// Make sure the entity exists before attempting an update.
	if _, err := s.GetByID(ctx, user.ID); err != nil {
		return err
	}

	if err := s.repo.Update(ctx, user); err != nil {
		return fmt.Errorf("update user: %w", err)
	}

	s.logger.Info("user updated", "id", user.ID)
	return nil
}

// Delete removes a user by id.
func (s *UserService) Delete(ctx context.Context, id int64) error {
	if _, err := s.GetByID(ctx, id); err != nil {
		return err
	}

	if err := s.repo.Delete(ctx, id); err != nil {
		return fmt.Errorf("delete user: %w", err)
	}

	s.logger.Info("user deleted", "id", id)
	return nil
}

// validateUser runs common field validations.
func (s *UserService) validateUser(u *domain.User) error {
	u.Email = strings.TrimSpace(u.Email)
	u.Name = strings.TrimSpace(u.Name)

	if u.Email == "" {
		return domain.NewValidationError("email", emptyValueMessage)
	}
	if !strings.Contains(u.Email, "@") {
		return domain.NewValidationError("email", "invalid format")
	}
	if u.Name == "" {
		return domain.NewValidationError("name", emptyValueMessage)
	}
	if u.Password == "" {
		return domain.NewValidationError("password", emptyValueMessage)
	}
	return nil
}

func (s *UserService) validateRegistration(u *domain.User) error {
	if err := s.validateUser(u); err != nil {
		return err
	}
	if len(u.Password) < 8 {
		return domain.NewValidationError("password", "must be at least 8 characters")
	}
	return nil
}
