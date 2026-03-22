package main

import (
	"log/slog"
	"os"

	"github.com/rxritet/Specto/internal/logging"
	"go.opentelemetry.io/contrib/bridges/otelslog"
)

func main() {
	// Structured JSON to stdout.
	jsonHandler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})

	// OTel bridge — forwards log records to OpenTelemetry when a
	// LoggerProvider is configured; no-ops otherwise.
	otelHandler := otelslog.NewHandler("specto")

	// Fan-out: every log record goes to both handlers.
	logger := slog.New(logging.NewFanoutHandler(jsonHandler, otelHandler))
	slog.SetDefault(logger)

	Execute()
}
