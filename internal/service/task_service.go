package service

import (
	"errors"
	"fmt"
	"log/slog"
	"strings"

	"github.com/rxritet/Specto/internal/domain"
)

// TaskService encapsulates business logic around tasks.
type TaskService struct {
	tasks  domain.TaskRepository
	users  domain.UserRepository
	logger *slog.Logger
}

// NewTaskService returns a ready-to-use TaskService.
func NewTaskService(tasks domain.TaskRepository, users domain.UserRepository, logger *slog.Logger) *TaskService {
	return &TaskService{tasks: tasks, users: users, logger: logger}
}

// Create validates the input and persists a new task.
func (s *TaskService) Create(task *domain.Task) error {
	if err := s.validateTask(task); err != nil {
		return err
	}

	// Verify the owning user exists.
	if _, err := s.users.GetByID(task.UserID); err != nil {
		return domain.NewNotFoundError("user", fmt.Sprintf("id=%d", task.UserID))
	}

	if err := s.tasks.Create(task); err != nil {
		return fmt.Errorf("create task: %w", err)
	}

	s.logger.Info("task created", "id", task.ID, "user_id", task.UserID)
	return nil
}

// GetByID returns a single task or a NotFoundError.
func (s *TaskService) GetByID(id int64) (*domain.Task, error) {
	task, err := s.tasks.GetByID(id)
	if err != nil {
		if _, ok := errors.AsType[*domain.NotFoundError](err); ok {
			return nil, err
		}
		return nil, domain.NewNotFoundError("task", fmt.Sprintf("id=%d", id))
	}
	return task, nil
}

// ListByUser returns all tasks for the given user.
func (s *TaskService) ListByUser(userID int64) ([]domain.Task, error) {
	// Verify the user exists so we can distinguish "no tasks" from
	// "user does not exist".
	if _, err := s.users.GetByID(userID); err != nil {
		return nil, domain.NewNotFoundError("user", fmt.Sprintf("id=%d", userID))
	}

	tasks, err := s.tasks.ListByUser(userID)
	if err != nil {
		return nil, fmt.Errorf("list tasks: %w", err)
	}
	return tasks, nil
}

// Update validates and persists changes to an existing task.
func (s *TaskService) Update(task *domain.Task) error {
	if task.ID == 0 {
		return domain.NewValidationError("id", "must be set")
	}

	if err := s.validateTask(task); err != nil {
		return err
	}

	// Ensure the task exists.
	if _, err := s.GetByID(task.ID); err != nil {
		return err
	}

	if err := s.tasks.Update(task); err != nil {
		return fmt.Errorf("update task: %w", err)
	}

	s.logger.Info("task updated", "id", task.ID)
	return nil
}

// Delete removes a task by id.
func (s *TaskService) Delete(id int64) error {
	if _, err := s.GetByID(id); err != nil {
		return err
	}

	if err := s.tasks.Delete(id); err != nil {
		return fmt.Errorf("delete task: %w", err)
	}

	s.logger.Info("task deleted", "id", id)
	return nil
}

// validateTask runs common field validations.
func (s *TaskService) validateTask(t *domain.Task) error {
	t.Title = strings.TrimSpace(t.Title)
	t.Description = strings.TrimSpace(t.Description)

	if t.Title == "" {
		return domain.NewValidationError("title", "must not be empty")
	}
	if t.UserID == 0 {
		return domain.NewValidationError("user_id", "must be set")
	}

	// Default status.
	if t.Status == "" {
		t.Status = domain.TaskStatusTodo
	}

	if !isValidStatus(t.Status) {
		return domain.NewValidationError("status",
			fmt.Sprintf("invalid value %q; allowed: todo, in_progress, done", t.Status))
	}

	return nil
}

// isValidStatus checks whether a TaskStatus value is one of the allowed enum values.
func isValidStatus(s domain.TaskStatus) bool {
	switch s {
	case domain.TaskStatusTodo, domain.TaskStatusInProgress, domain.TaskStatusDone:
		return true
	}
	return false
}
