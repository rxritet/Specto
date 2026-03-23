package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"testing"

	"github.com/rxritet/Specto/internal/domain"
)

var testCtx = context.Background()

const (
	errExpectedNoError     = "expected no error, got %v"
	errExpectedNotFound    = "expected *NotFoundError, got %T: %v"
	errSetup               = "setup: %v"
	errExpectedMissingTask = "expected error for missing task"
	titleFindMe            = "Find me"
)

// ---------- In-memory mock repositories ----------

// memUserRepo is a minimal in-memory UserRepository for testing.
type memUserRepo struct {
	mu    sync.Mutex
	users map[int64]*domain.User
	seq   int64
}

func newMemUserRepo() *memUserRepo {
	return &memUserRepo{users: make(map[int64]*domain.User)}
}

func (r *memUserRepo) Create(_ context.Context, u *domain.User) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.seq++
	u.ID = r.seq
	clone := *u
	r.users[u.ID] = &clone
	return nil
}

func (r *memUserRepo) GetByID(_ context.Context, id int64) (*domain.User, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	u, ok := r.users[id]
	if !ok {
		return nil, domain.NewNotFoundError("user", fmt.Sprintf("id=%d", id))
	}
	clone := *u
	return &clone, nil
}

func (r *memUserRepo) GetByEmail(_ context.Context, email string) (*domain.User, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, u := range r.users {
		if u.Email == email {
			clone := *u
			return &clone, nil
		}
	}
	return nil, domain.NewNotFoundError("user", email)
}

func (r *memUserRepo) Update(_ context.Context, u *domain.User) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.users[u.ID]; !ok {
		return domain.NewNotFoundError("user", fmt.Sprintf("id=%d", u.ID))
	}
	clone := *u
	r.users[u.ID] = &clone
	return nil
}

func (r *memUserRepo) Delete(_ context.Context, id int64) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.users, id)
	return nil
}

// memTaskRepo is a minimal in-memory TaskRepository for testing.
type memTaskRepo struct {
	mu    sync.Mutex
	tasks map[int64]*domain.Task
	seq   int64
}

func newMemTaskRepo() *memTaskRepo {
	return &memTaskRepo{tasks: make(map[int64]*domain.Task)}
}

func (r *memTaskRepo) Create(_ context.Context, t *domain.Task) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.seq++
	t.ID = r.seq
	clone := *t
	r.tasks[t.ID] = &clone
	return nil
}

func (r *memTaskRepo) GetByID(_ context.Context, id int64) (*domain.Task, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	t, ok := r.tasks[id]
	if !ok {
		return nil, domain.NewNotFoundError("task", fmt.Sprintf("id=%d", id))
	}
	clone := *t
	return &clone, nil
}

func (r *memTaskRepo) ListByUser(_ context.Context, userID int64) ([]domain.Task, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var result []domain.Task
	for _, t := range r.tasks {
		if t.UserID == userID {
			result = append(result, *t)
		}
	}
	return result, nil
}

func (r *memTaskRepo) Update(_ context.Context, t *domain.Task) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.tasks[t.ID]; !ok {
		return domain.NewNotFoundError("task", fmt.Sprintf("id=%d", t.ID))
	}
	clone := *t
	r.tasks[t.ID] = &clone
	return nil
}

func (r *memTaskRepo) Delete(_ context.Context, id int64) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.tasks[id]; !ok {
		return domain.NewNotFoundError("task", fmt.Sprintf("id=%d", id))
	}
	delete(r.tasks, id)
	return nil
}

// ---------- Helper ----------

// newTestTaskService wires up a TaskService with in-memory repos and a
// pre-seeded user (id=1).
func newTestTaskService(t *testing.T) (*TaskService, *memUserRepo, *memTaskRepo) {
	t.Helper()

	users := newMemUserRepo()
	tasks := newMemTaskRepo()
	logger := slog.Default()

	// Seed a user so task creation has a valid owner.
	_ = users.Create(testCtx, &domain.User{
		Email:    "test@example.com",
		Name:     "Test User",
		Password: "hashed",
	})

	svc := NewTaskService(tasks, users, logger)
	return svc, users, tasks
}

// ---------- Tests: Create ----------

func TestTaskServiceCreateSuccess(t *testing.T) {
	svc, _, _ := newTestTaskService(t)

	task := &domain.Task{
		UserID:      1,
		Title:       "Write tests",
		Description: "Cover CRUD methods",
	}

	if err := svc.Create(testCtx, task); err != nil {
		t.Fatalf(errExpectedNoError, err)
	}

	if task.ID == 0 {
		t.Fatal("expected task ID to be assigned")
	}
	if task.Status != domain.TaskStatusTodo {
		t.Fatalf("expected default status %q, got %q", domain.TaskStatusTodo, task.Status)
	}
}

func TestTaskServiceCreateDefaultStatus(t *testing.T) {
	svc, _, _ := newTestTaskService(t)

	task := &domain.Task{
		UserID: 1,
		Title:  "No explicit status",
	}

	if err := svc.Create(testCtx, task); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if task.Status != domain.TaskStatusTodo {
		t.Fatalf("expected default status todo, got %q", task.Status)
	}
}

func TestTaskServiceCreateValidationErrorEmptyTitle(t *testing.T) {
	svc, _, _ := newTestTaskService(t)

	task := &domain.Task{UserID: 1, Title: ""}

	err := svc.Create(testCtx, task)
	if err == nil {
		t.Fatal("expected validation error, got nil")
	}

	if _, ok := errors.AsType[*domain.ValidationError](err); !ok {
		t.Fatalf("expected *ValidationError, got %T: %v", err, err)
	}
}

func TestTaskServiceCreateValidationErrorInvalidStatus(t *testing.T) {
	svc, _, _ := newTestTaskService(t)

	task := &domain.Task{
		UserID: 1,
		Title:  "Bad status",
		Status: "canceled",
	}

	err := svc.Create(testCtx, task)
	if err == nil {
		t.Fatal("expected validation error for invalid status")
	}

	ve, ok := errors.AsType[*domain.ValidationError](err)
	if !ok {
		t.Fatalf("expected *ValidationError, got %T: %v", err, err)
	}
	if ve.Field != "status" {
		t.Fatalf("expected field 'status', got %q", ve.Field)
	}
}

func TestTaskServiceCreateNotFoundErrorNoUser(t *testing.T) {
	svc, _, _ := newTestTaskService(t)

	task := &domain.Task{
		UserID: 999, // non-existent user
		Title:  "Orphan task",
	}

	err := svc.Create(testCtx, task)
	if err == nil {
		t.Fatal("expected error for non-existent user")
	}

	if _, ok := errors.AsType[*domain.NotFoundError](err); !ok {
		t.Fatalf(errExpectedNotFound, err, err)
	}
}

// ---------- Tests: GetByID ----------

func TestTaskServiceGetByIDSuccess(t *testing.T) {
	svc, _, _ := newTestTaskService(t)

	created := &domain.Task{UserID: 1, Title: titleFindMe}
	if err := svc.Create(testCtx, created); err != nil {
		t.Fatalf(errSetup, err)
	}

	got, err := svc.GetByID(testCtx, created.ID)
	if err != nil {
		t.Fatalf(errExpectedNoError, err)
	}
	if got.Title != titleFindMe {
		t.Fatalf("expected title %q, got %q", titleFindMe, got.Title)
	}
}

func TestTaskServiceGetByIDNotFound(t *testing.T) {
	svc, _, _ := newTestTaskService(t)

	_, err := svc.GetByID(testCtx, 42)
	if err == nil {
		t.Fatal(errExpectedMissingTask)
	}

	nf, ok := errors.AsType[*domain.NotFoundError](err)
	if !ok {
		t.Fatalf(errExpectedNotFound, err, err)
	}
	if nf.Entity != "task" {
		t.Fatalf("expected entity 'task', got %q", nf.Entity)
	}
}

// ---------- Tests: ListByUser ----------

func TestTaskServiceListByUserSuccess(t *testing.T) {
	svc, _, _ := newTestTaskService(t)

	for _, title := range []string{"A", "B", "C"} {
		if err := svc.Create(testCtx, &domain.Task{UserID: 1, Title: title}); err != nil {
			t.Fatalf(errSetup, err)
		}
	}

	tasks, err := svc.ListByUser(testCtx, 1)
	if err != nil {
		t.Fatalf(errExpectedNoError, err)
	}
	if len(tasks) != 3 {
		t.Fatalf("expected 3 tasks, got %d", len(tasks))
	}
}

func TestTaskServiceListByUserNotFoundUser(t *testing.T) {
	svc, _, _ := newTestTaskService(t)

	_, err := svc.ListByUser(testCtx, 999)
	if err == nil {
		t.Fatal("expected error for non-existent user")
	}

	if _, ok := errors.AsType[*domain.NotFoundError](err); !ok {
		t.Fatalf(errExpectedNotFound, err, err)
	}
}

// ---------- Tests: Update ----------

func TestTaskServiceUpdateSuccess(t *testing.T) {
	svc, _, _ := newTestTaskService(t)

	task := &domain.Task{UserID: 1, Title: "Original"}
	if err := svc.Create(testCtx, task); err != nil {
		t.Fatalf(errSetup, err)
	}

	task.Title = "Updated"
	task.Status = domain.TaskStatusDone
	if err := svc.Update(testCtx, task); err != nil {
		t.Fatalf(errExpectedNoError, err)
	}

	got, _ := svc.GetByID(testCtx, task.ID)
	if got.Title != "Updated" {
		t.Fatalf("expected title %q, got %q", "Updated", got.Title)
	}
	if got.Status != domain.TaskStatusDone {
		t.Fatalf("expected status %q, got %q", domain.TaskStatusDone, got.Status)
	}
}

func TestTaskServiceUpdateNotFound(t *testing.T) {
	svc, _, _ := newTestTaskService(t)

	err := svc.Update(testCtx, &domain.Task{ID: 999, UserID: 1, Title: "Ghost", Status: domain.TaskStatusTodo})
	if err == nil {
		t.Fatal(errExpectedMissingTask)
	}

	if _, ok := errors.AsType[*domain.NotFoundError](err); !ok {
		t.Fatalf(errExpectedNotFound, err, err)
	}
}

// ---------- Tests: Delete ----------

func TestTaskServiceDeleteSuccess(t *testing.T) {
	svc, _, _ := newTestTaskService(t)

	task := &domain.Task{UserID: 1, Title: "Delete me"}
	if err := svc.Create(testCtx, task); err != nil {
		t.Fatalf(errSetup, err)
	}

	if err := svc.Delete(testCtx, task.ID); err != nil {
		t.Fatalf(errExpectedNoError, err)
	}

	_, err := svc.GetByID(testCtx, task.ID)
	if _, ok := errors.AsType[*domain.NotFoundError](err); !ok {
		t.Fatalf("expected *NotFoundError after delete, got %T: %v", err, err)
	}
}

func TestTaskServiceDeleteNotFound(t *testing.T) {
	svc, _, _ := newTestTaskService(t)

	err := svc.Delete(testCtx, 999)
	if err == nil {
		t.Fatal(errExpectedMissingTask)
	}

	if _, ok := errors.AsType[*domain.NotFoundError](err); !ok {
		t.Fatalf(errExpectedNotFound, err, err)
	}
}

// ---------- Tests: StatsByUser (SIMD / generic) ----------

func TestTaskServiceStatsByUser(t *testing.T) {
	svc, _, _ := newTestTaskService(t)

	// Expected status mix: 2 pending, 1 in_progress, 1 done.
	for _, tt := range []struct {
		title  string
		status domain.TaskStatus
	}{
		{"T1", domain.TaskStatusTodo},
		{"T2", domain.TaskStatusTodo},
		{"T3", domain.TaskStatusInProgress},
		{"T4", domain.TaskStatusDone},
	} {
		if err := svc.Create(testCtx, &domain.Task{UserID: 1, Title: tt.title, Status: tt.status}); err != nil {
			t.Fatalf(errSetup, err)
		}
	}

	stats, err := svc.StatsByUser(testCtx, 1)
	if err != nil {
		t.Fatalf(errExpectedNoError, err)
	}

	if stats.Total != 4 {
		t.Fatalf("expected total=4, got %d", stats.Total)
	}
	if stats.TodoCount != 2 {
		t.Fatalf("expected todo=2, got %d", stats.TodoCount)
	}
	if stats.InProgCount != 1 {
		t.Fatalf("expected in_progress=1, got %d", stats.InProgCount)
	}
	if stats.DoneCount != 1 {
		t.Fatalf("expected done=1, got %d", stats.DoneCount)
	}
	if stats.TodoPct != 50 {
		t.Fatalf("expected todo_pct=50, got %.2f", stats.TodoPct)
	}
}
