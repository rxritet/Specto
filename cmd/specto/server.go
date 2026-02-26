package main

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/rxritet/Specto/internal/config"
	"github.com/rxritet/Specto/internal/database"
	"github.com/rxritet/Specto/internal/server"
	"github.com/spf13/cobra"
)

var serverCmd = &cobra.Command{
	Use:   "server",
	Short: "Start the Specto HTTP server",
	Long:  "Launches the Specto web server on the configured address and port.",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg := config.Load()
		logger := slog.Default()

		// Run SQL migrations for PostgreSQL provider.
		if cfg.DBProvider == "postgres" {
			db, err := database.OpenPostgres(cfg.DBDsn, logger)
			if err != nil {
				return fmt.Errorf("open postgres: %w", err)
			}
			defer db.Close()

			if err := database.Migrate(context.Background(), db, logger); err != nil {
				return fmt.Errorf("migrate: %w", err)
			}
		}

		srv := server.New(cfg, logger)
		return srv.Run()
	},
}

func init() {
	rootCmd.AddCommand(serverCmd)
}
