package web

import (
	"io"
	"log/slog"
)

// noopLogger returns a slog.Logger that discards output (for tests).
func noopLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{Level: slog.LevelError + 1}))
}
