package database

import (
	"context"
	"database/sql"
	_ "embed"
	"fmt"
	"log/slog"
)

//go:embed fixtures/fixtures.sql
var fixturesSQL string

// Seed loads test/fixture data into a PostgreSQL database.
// Intended for development and CI only.
func Seed(ctx context.Context, db *sql.DB, logger *slog.Logger) error {
	if _, err := db.ExecContext(ctx, fixturesSQL); err != nil {
		return fmt.Errorf("seed: %w", err)
	}
	logger.Info("seed data applied")
	return nil
}
