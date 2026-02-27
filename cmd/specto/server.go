package main

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/rxritet/Specto/internal/config"
	"github.com/rxritet/Specto/internal/database"
	"github.com/rxritet/Specto/internal/domain"
	"github.com/rxritet/Specto/internal/server"
	"github.com/rxritet/Specto/internal/service"
	"github.com/spf13/cobra"
)

var serverCmd = &cobra.Command{
	Use:   "server",
	Short: "Start the Specto HTTP server",
	Long:  "Launches the Specto web server on the configured address and port.",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg := config.Load()
		logger := slog.Default()

		var (
			userRepo domain.UserRepository
			taskRepo domain.TaskRepository
		)

		switch cfg.DBProvider {
		case "postgres":
			db, err := database.OpenPostgres(cfg.DBDsn, logger)
			if err != nil {
				return fmt.Errorf("open postgres: %w", err)
			}
			defer db.Close()

			if err := database.Migrate(context.Background(), db, logger); err != nil {
				return fmt.Errorf("migrate: %w", err)
			}

			userRepo = database.NewPgUserRepo(db)
			taskRepo = database.NewPgTaskRepo(db)

		case "bolt":
			bdb, err := database.OpenBolt(cfg.DBPath, logger)
			if err != nil {
				return fmt.Errorf("open bolt: %w", err)
			}
			defer bdb.Close()

			userRepo = database.NewBoltUserRepo(bdb)
			taskRepo = database.NewBoltTaskRepo(bdb)

		default:
			return fmt.Errorf("unknown db provider: %s", cfg.DBProvider)
		}

		taskSvc := service.NewTaskService(taskRepo, userRepo, logger)

		srv := server.New(cfg, logger, taskSvc)
		return srv.Run()
	},
}

func init() {
	rootCmd.AddCommand(serverCmd)
}
