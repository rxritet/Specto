package database

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"log/slog"
	"sort"
	"strings"
)

//go:embed migrations/*.up.sql
var migrationsFS embed.FS

// Migrate runs all embedded *.up.sql migrations sequentially inside a
// transaction. It is safe to call on every server start — migrations use
// IF NOT EXISTS / idempotent DDL.
//
// For BoltDB the function is a no-op (schema-less store).
func Migrate(ctx context.Context, db *sql.DB, logger *slog.Logger) error {
	entries, err := migrationsFS.ReadDir("migrations")
	if err != nil {
		return fmt.Errorf("read migrations dir: %w", err)
	}

	// Collect and sort only *.up.sql files.
	var files []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".up.sql") {
			files = append(files, e.Name())
		}
	}
	sort.Strings(files)

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin migration tx: %w", err)
	}

	for _, name := range files {
		data, err := migrationsFS.ReadFile("migrations/" + name)
		if err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("read %s: %w", name, err)
		}

		if _, err := tx.ExecContext(ctx, string(data)); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("exec %s: %w", name, err)
		}
		logger.Info("migration applied", "file", name)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit migrations: %w", err)
	}
	return nil
}
