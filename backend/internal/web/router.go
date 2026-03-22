package web

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/rxritet/Specto/internal/config"
	"github.com/rxritet/Specto/internal/service"
)

// Router holds references shared by all HTTP handlers and exposes a
// configured http.Handler ready to be plugged into net/http.Server.
type Router struct {
	Mux    *http.ServeMux
	Logger *slog.Logger
	Tasks  *service.TaskService
	Users  *service.UserService
	auth   *sessionManager
}

// NewRouter creates the application ServeMux, registers all routes using
// Go 1.22+ pattern matching, and wraps the mux with the middleware stack.
func NewRouter(cfg *config.Config, logger *slog.Logger, tasks *service.TaskService, users *service.UserService) http.Handler {
	r := &Router{
		Mux:    http.NewServeMux(),
		Logger: logger,
		Tasks:  tasks,
		Users:  users,
		auth:   newSessionManager(cfg),
	}

	r.routes()

	// Middleware stack (outermost → innermost):
	//   Recovery → SecureHeaders → Logging → Mux
	return Chain(
		r.Mux,
		Recovery(logger),
		SecureHeaders(),
		Logging(logger),
	)
}

// routes registers every application endpoint.
// Method + path patterns use Go 1.22+ enhanced routing.
func (rt *Router) routes() {
	// Health-check — always available.
	rt.Mux.HandleFunc("GET /health", rt.handleHealth)
	rt.Mux.HandleFunc("POST /auth/register", rt.handleAuthRegister)
	rt.Mux.HandleFunc("POST /auth/login", rt.handleAuthLogin)
	rt.Mux.HandleFunc("POST /auth/logout", rt.handleAuthLogout)
	rt.Mux.Handle("GET /auth/me", rt.requireAuth(http.HandlerFunc(rt.handleAuthMe)))

	// Task CRUD.
	rt.Mux.Handle("GET /tasks", rt.requireAuth(http.HandlerFunc(rt.handleTaskList)))
	rt.Mux.Handle("POST /tasks", rt.requireAuth(http.HandlerFunc(rt.handleTaskCreate)))
	rt.Mux.Handle("GET /tasks/stats", rt.requireAuth(http.HandlerFunc(rt.handleTaskStats)))
	rt.Mux.Handle("GET /tasks/{id}", rt.requireAuth(http.HandlerFunc(rt.handleTaskGet)))
	rt.Mux.Handle("PUT /tasks/{id}", rt.requireAuth(http.HandlerFunc(rt.handleTaskUpdate)))
	rt.Mux.Handle("DELETE /tasks/{id}", rt.requireAuth(http.HandlerFunc(rt.handleTaskDelete)))
}

// ---------- Handlers ----------

// handleHealth responds with a simple JSON status.
func (rt *Router) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}
