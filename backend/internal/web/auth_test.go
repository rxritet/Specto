package web

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/rxritet/Specto/internal/config"
	"github.com/rxritet/Specto/internal/domain"
	"github.com/rxritet/Specto/internal/service"
)

const (
	errExpectedStatus = "expected status %d, got %d"
	headerContentType = "Content-Type"
	contentTypeJSON   = "application/json"
)

type authTestUserRepo struct {
	users map[int64]*domain.User
	seq   int64
}

func newAuthTestUserRepo() *authTestUserRepo {
	return &authTestUserRepo{users: make(map[int64]*domain.User)}
}

func (r *authTestUserRepo) Create(_ context.Context, user *domain.User) error {
	r.seq++
	user.ID = r.seq
	clone := *user
	r.users[user.ID] = &clone
	return nil
}

func (r *authTestUserRepo) GetByID(_ context.Context, id int64) (*domain.User, error) {
	user, ok := r.users[id]
	if !ok {
		return nil, domain.NewNotFoundError("user", fmt.Sprintf("id=%d", id))
	}
	clone := *user
	return &clone, nil
}

func (r *authTestUserRepo) GetByEmail(_ context.Context, email string) (*domain.User, error) {
	for _, user := range r.users {
		if user.Email == email {
			clone := *user
			return &clone, nil
		}
	}
	return nil, domain.NewNotFoundError("user", email)
}

func (r *authTestUserRepo) Update(_ context.Context, user *domain.User) error {
	clone := *user
	r.users[user.ID] = &clone
	return nil
}

func (r *authTestUserRepo) Delete(_ context.Context, id int64) error {
	delete(r.users, id)
	return nil
}

type authTestTaskRepo struct {
	tasks map[int64]*domain.Task
	seq   int64
}

func newAuthTestTaskRepo() *authTestTaskRepo {
	return &authTestTaskRepo{tasks: make(map[int64]*domain.Task)}
}

func (r *authTestTaskRepo) Create(_ context.Context, task *domain.Task) error {
	r.seq++
	task.ID = r.seq
	now := time.Now()
	task.CreatedAt = now
	task.UpdatedAt = now
	clone := *task
	r.tasks[task.ID] = &clone
	return nil
}

func (r *authTestTaskRepo) GetByID(_ context.Context, id int64) (*domain.Task, error) {
	task, ok := r.tasks[id]
	if !ok {
		return nil, domain.NewNotFoundError("task", fmt.Sprintf("id=%d", id))
	}
	clone := *task
	return &clone, nil
}

func (r *authTestTaskRepo) ListByUser(_ context.Context, userID int64) ([]domain.Task, error) {
	var out []domain.Task
	for _, task := range r.tasks {
		if task.UserID == userID {
			out = append(out, *task)
		}
	}
	return out, nil
}

func (r *authTestTaskRepo) Update(_ context.Context, task *domain.Task) error {
	clone := *task
	r.tasks[task.ID] = &clone
	return nil
}

func (r *authTestTaskRepo) Delete(_ context.Context, id int64) error {
	delete(r.tasks, id)
	return nil
}

func TestProtectedRouteRequiresAuthentication(t *testing.T) {
	router := newAuthTestRouter()
	req := httptest.NewRequest(http.MethodGet, "/tasks", nil)
	rr := httptest.NewRecorder()

	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf(errExpectedStatus, http.StatusUnauthorized, rr.Code)
	}
}

func TestAuthFlowAndTaskOwnership(t *testing.T) {
	router := newAuthTestRouter()

	aliceCookie := registerAndLogin(t, router, "alice@example.com", "Alice", "password123")
	aliceTaskID := createTask(t, router, aliceCookie, map[string]any{
		"user_id": 999,
		"title":   "Alice Task",
		"status":  "todo",
	})

	meReq := httptest.NewRequest(http.MethodGet, "/auth/me", nil)
	meReq.AddCookie(aliceCookie)
	meRec := httptest.NewRecorder()
	router.ServeHTTP(meRec, meReq)
	if meRec.Code != http.StatusOK {
		t.Fatalf(errExpectedStatus, http.StatusOK, meRec.Code)
	}

	bobCookie := registerAndLogin(t, router, "bob@example.com", "Bob", "password123")
	bobReq := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/tasks/%d", aliceTaskID), nil)
	bobReq.AddCookie(bobCookie)
	bobRec := httptest.NewRecorder()
	router.ServeHTTP(bobRec, bobReq)
	if bobRec.Code != http.StatusNotFound {
		t.Fatalf(errExpectedStatus, http.StatusNotFound, bobRec.Code)
	}

	logoutReq := httptest.NewRequest(http.MethodPost, "/auth/logout", nil)
	logoutReq.AddCookie(aliceCookie)
	logoutRec := httptest.NewRecorder()
	router.ServeHTTP(logoutRec, logoutReq)
	if logoutRec.Code != http.StatusOK {
		t.Fatalf(errExpectedStatus, http.StatusOK, logoutRec.Code)
	}
}

func newAuthTestRouter() http.Handler {
	userRepo := newAuthTestUserRepo()
	taskRepo := newAuthTestTaskRepo()
	logger := noopLogger()
	userSvc := service.NewUserService(userRepo, logger)
	taskSvc := service.NewTaskService(taskRepo, userRepo, logger)
	cfg := &config.Config{
		AuthSecret:        "test-secret",
		AuthSessionTTL:    time.Hour,
		AuthSecureCookies: false,
	}
	return NewRouter(cfg, logger, taskSvc, userSvc)
}

func registerAndLogin(t *testing.T, router http.Handler, email, name, password string) *http.Cookie {
	t.Helper()
	body, err := json.Marshal(map[string]string{
		"email":    email,
		"name":     name,
		"password": password,
	})
	if err != nil {
		t.Fatalf("marshal register request: %v", err)
	}

	registerReq := httptest.NewRequest(http.MethodPost, "/auth/register", bytes.NewReader(body))
	registerReq.Header.Set(headerContentType, contentTypeJSON)
	registerRec := httptest.NewRecorder()
	router.ServeHTTP(registerRec, registerReq)
	if registerRec.Code != http.StatusCreated {
		t.Fatalf(errExpectedStatus, http.StatusCreated, registerRec.Code)
	}

	loginBody, err := json.Marshal(map[string]string{
		"email":    email,
		"password": password,
	})
	if err != nil {
		t.Fatalf("marshal login request: %v", err)
	}

	loginReq := httptest.NewRequest(http.MethodPost, "/auth/login", bytes.NewReader(loginBody))
	loginReq.Header.Set(headerContentType, contentTypeJSON)
	loginRec := httptest.NewRecorder()
	router.ServeHTTP(loginRec, loginReq)
	if loginRec.Code != http.StatusOK {
		t.Fatalf(errExpectedStatus, http.StatusOK, loginRec.Code)
	}

	return loginRec.Result().Cookies()[0]
}

func createTask(t *testing.T, router http.Handler, sessionCookie *http.Cookie, payload map[string]any) int64 {
	t.Helper()
	body, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal task request: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/tasks", bytes.NewReader(body))
	req.Header.Set(headerContentType, contentTypeJSON)
	req.AddCookie(sessionCookie)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	if rr.Code != http.StatusCreated {
		t.Fatalf(errExpectedStatus, http.StatusCreated, rr.Code)
	}

	var task domain.Task
	if err := json.NewDecoder(rr.Body).Decode(&task); err != nil {
		t.Fatalf("decode task response: %v", err)
	}
	if task.UserID == 999 {
		t.Fatal("expected authenticated user id to override request user_id")
	}
	return task.ID
}
