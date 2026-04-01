package web

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/redis/go-redis/v9"
	"github.com/rxritet/Specto/internal/config"
	"github.com/rxritet/Specto/internal/domain"
	"github.com/rxritet/Specto/internal/service"
)

// Router holds references shared by all HTTP handlers and exposes a
// configured http.Handler ready to be plugged into net/http.Server.
type Router struct {
	Mux     *http.ServeMux
	Logger  *slog.Logger
	Tasks   *service.TaskService
	Users   *service.UserService
	Banking *service.BankingService // Добавлено для банковских операций
	Audit   domain.AuditLogger
	auth    *sessionManager
}

// NewRouter creates the application ServeMux, registers all routes using
// Go 1.22+ pattern matching, and wraps the mux with the middleware stack.
func NewRouter(cfg *config.Config, logger *slog.Logger, tasks *service.TaskService, users *service.UserService, banking *service.BankingService, redisClient *redis.Client, auditLogger domain.AuditLogger) http.Handler {
	redisConn := redisClient

	r := &Router{
		Mux:     http.NewServeMux(),
		Logger:  logger,
		Tasks:   tasks,
		Users:   users,
		Banking: banking,
		Audit:   auditLogger,
		auth:    newSessionManager(cfg, redisConn),
	}

	r.routes()

	middlewares := []Middleware{
		Recovery(logger),
		SecureHeaders(),
		Logging(logger),
	}

	if redisConn != nil {
		middlewares = append([]Middleware{RedisRateLimit(redisConn, cfg.RateLimitPerMinute, logger)}, middlewares...)
	}
	if auditLogger != nil {
		middlewares = append([]Middleware{AuditTrail(auditLogger, logger)}, middlewares...)
	}

	// Middleware stack (outermost → innermost):
	//   AuditTrail(optional) → RedisRateLimit(optional) → Recovery → SecureHeaders → Logging → Mux
	return Chain(
		r.Mux,
		middlewares...,
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

	// Banking API.
	rt.Mux.Handle("GET /accounts", rt.requireAuth(http.HandlerFunc(rt.handleAccountList)))
	rt.Mux.Handle("POST /accounts", rt.requireAuth(http.HandlerFunc(rt.handleAccountCreate)))
	rt.Mux.Handle("POST /accounts/{id}/transfer", rt.requireAuth(http.HandlerFunc(rt.handleTransferCreate)))
}

// ---------- Handlers ----------

func (rt *Router) handleAccountList(w http.ResponseWriter, r *http.Request) {
	user, ok := currentUser(r)
	if !ok {
		rt.respondError(w, http.StatusUnauthorized, authenticationRequiredMessage)
		return
	}
	accounts, err := rt.Banking.GetUserAccounts(r.Context(), user.ID)
	if err != nil {
		rt.handleServiceError(w, err)
		return
	}
	rt.respondJSON(w, http.StatusOK, accounts)
}

func (rt *Router) handleAccountCreate(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Currency string `json:"currency"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		rt.respondError(w, http.StatusBadRequest, err.Error())
		return
	}
	user, ok := currentUser(r)
	if !ok {
		rt.respondError(w, http.StatusUnauthorized, authenticationRequiredMessage)
		return
	}
	acc, err := rt.Banking.CreateAccount(r.Context(), user.ID, req.Currency)
	if err != nil {
		rt.handleServiceError(w, err)
		return
	}
	rt.respondJSON(w, http.StatusCreated, acc)
}

func (rt *Router) handleTransferCreate(w http.ResponseWriter, r *http.Request) {
	id, err := pathInt64(r, "id")
	if err != nil {
		rt.respondError(w, http.StatusBadRequest, "invalid account id")
		return
	}

	var req domain.CreateTransferRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		rt.respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	user, ok := currentUser(r)
	if !ok {
		rt.respondError(w, http.StatusUnauthorized, authenticationRequiredMessage)
		return
	}
	transfer, err := rt.Banking.Transfer(r.Context(), user.ID, req, id)
	if err != nil {
		rt.handleServiceError(w, err)
		return
	}
	rt.respondJSON(w, http.StatusCreated, transfer)
}

// handleHealth responds with a simple JSON status.
func (rt *Router) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}
