package server

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/rxritet/Specto/internal/config"
	"github.com/rxritet/Specto/internal/service"
	"github.com/rxritet/Specto/internal/web"
)

// Server wraps the standard net/http.Server with application dependencies.
type Server struct {
	cfg    *config.Config
	http   *http.Server
	logger *slog.Logger
}

// New creates a Server configured with the given Config and logger.
func New(cfg *config.Config, logger *slog.Logger, tasks *service.TaskService) *Server {
	s := &Server{
		cfg:    cfg,
		logger: logger,
	}

	handler := web.NewRouter(logger, tasks)

	s.http = &http.Server{
		Addr:         cfg.Addr(),
		Handler:      handler,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	return s
}

// Run starts the HTTP listener. It blocks until the server stops.
func (s *Server) Run() error {
	s.logger.Info("starting http server", "addr", s.http.Addr)
	return s.http.ListenAndServe()
}
