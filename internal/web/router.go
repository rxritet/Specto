package web

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/rxritet/Specto/internal/service"
)

// Router holds references shared by all HTTP handlers and exposes a
// configured http.Handler ready to be plugged into net/http.Server.
type Router struct {
	Mux    *http.ServeMux
	Logger *slog.Logger
	Tasks  *service.TaskService
}

// NewRouter creates the application ServeMux, registers all routes using
// Go 1.22+ pattern matching, and wraps the mux with the middleware stack.
func NewRouter(logger *slog.Logger, tasks *service.TaskService) http.Handler {
	r := &Router{
		Mux:    http.NewServeMux(),
		Logger: logger,
		Tasks:  tasks,
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

	// Task CRUD.
	rt.Mux.HandleFunc("GET /tasks", rt.handleTaskList)
	rt.Mux.HandleFunc("POST /tasks", rt.handleTaskCreate)
	rt.Mux.HandleFunc("GET /tasks/stats", rt.handleTaskStats)
	rt.Mux.HandleFunc("GET /tasks/{id}", rt.handleTaskGet)
	rt.Mux.HandleFunc("PUT /tasks/{id}", rt.handleTaskUpdate)
	rt.Mux.HandleFunc("DELETE /tasks/{id}", rt.handleTaskDelete)
}

// ---------- Handlers ----------

// handleHealth responds with a simple JSON status.
func (rt *Router) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}
