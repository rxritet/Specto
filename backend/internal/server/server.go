package server

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/rxritet/Specto/internal/config"
	"github.com/rxritet/Specto/internal/domain"
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
func New(cfg *config.Config, logger *slog.Logger, tasks *service.TaskService, users *service.UserService, redisClient *redis.Client, auditLogger domain.AuditLogger) *Server {
	s := &Server{
		cfg:    cfg,
		logger: logger,
	}

	handler := web.NewRouter(cfg, logger, tasks, users, redisClient, auditLogger)

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
	errCh := make(chan error, 1)

	go func() {
		err := s.http.ListenAndServe()
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
			return
		}
		close(errCh)
	}()

	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(signals)

	select {
	case err := <-errCh:
		return err
	case sig := <-signals:
		s.logger.Info("shutdown signal received", "signal", sig.String())
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := s.http.Shutdown(ctx); err != nil {
		return err
	}

	if err, ok := <-errCh; ok {
		return err
	}

	s.logger.Info("http server stopped")
	return nil
}
