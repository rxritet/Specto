package service

import (
	"errors"
	"fmt"
	"log/slog"
	"strings"

	"github.com/rxritet/Specto/internal/domain"
)

// UserService encapsulates business logic around users.
type UserService struct {
	repo   domain.UserRepository
	logger *slog.Logger
}

// NewUserService returns a ready-to-use UserService.
func NewUserService(repo domain.UserRepository, logger *slog.Logger) *UserService {
	return &UserService{repo: repo, logger: logger}
}

// Create validates the input and persists a new user.
func (s *UserService) Create(user *domain.User) error {
	if err := s.validateUser(user); err != nil {
		return err
	}

	// Guard against duplicate email.
	if existing, _ := s.repo.GetByEmail(user.Email); existing != nil {
		return domain.NewConflictError("user", fmt.Sprintf("email %q already exists", user.Email))
	}

	if err := s.repo.Create(user); err != nil {
		return fmt.Errorf("create user: %w", err)
	}

	s.logger.Info("user created", "id", user.ID, "email", user.Email)
	return nil
}

// GetByID returns a single user or a NotFoundError.
func (s *UserService) GetByID(id int64) (*domain.User, error) {
	user, err := s.repo.GetByID(id)
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
func (s *UserService) GetByEmail(email string) (*domain.User, error) {
	user, err := s.repo.GetByEmail(email)
	if err != nil {
		if _, ok := errors.AsType[*domain.NotFoundError](err); ok {
			return nil, err
		}
		return nil, domain.NewNotFoundError("user", email)
	}
	return user, nil
}

// Update validates and persists changes to an existing user.
func (s *UserService) Update(user *domain.User) error {
	if user.ID == 0 {
		return domain.NewValidationError("id", "must be set")
	}

	if err := s.validateUser(user); err != nil {
		return err
	}

	// Make sure the entity exists before attempting an update.
	if _, err := s.GetByID(user.ID); err != nil {
		return err
	}

	if err := s.repo.Update(user); err != nil {
		return fmt.Errorf("update user: %w", err)
	}

	s.logger.Info("user updated", "id", user.ID)
	return nil
}

// Delete removes a user by id.
func (s *UserService) Delete(id int64) error {
	if _, err := s.GetByID(id); err != nil {
		return err
	}

	if err := s.repo.Delete(id); err != nil {
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
		return domain.NewValidationError("email", "must not be empty")
	}
	if !strings.Contains(u.Email, "@") {
		return domain.NewValidationError("email", "invalid format")
	}
	if u.Name == "" {
		return domain.NewValidationError("name", "must not be empty")
	}
	if u.Password == "" {
		return domain.NewValidationError("password", "must not be empty")
	}
	return nil
}
