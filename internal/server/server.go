package server

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/rxritet/Specto/internal/config"
)

// Server wraps the standard net/http.Server with application dependencies.
type Server struct {
	cfg    *config.Config
	http   *http.Server
	logger *slog.Logger
}

// New creates a Server configured with the given Config and logger.
func New(cfg *config.Config, logger *slog.Logger) *Server {
	s := &Server{
		cfg:    cfg,
		logger: logger,
	}

	mux := http.NewServeMux()
	s.routes(mux)

	s.http = &http.Server{
		Addr:         cfg.Addr(),
		Handler:      mux,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	return s
}

// routes registers all HTTP endpoints on the given mux.
func (s *Server) routes(mux *http.ServeMux) {
	mux.HandleFunc("GET /health", s.handleHealth)
}

// Run starts the HTTP listener. It blocks until the server stops.
func (s *Server) Run() error {
	s.logger.Info("starting http server", "addr", s.http.Addr)
	return s.http.ListenAndServe()
}

// handleHealth responds with a simple JSON status.
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}
