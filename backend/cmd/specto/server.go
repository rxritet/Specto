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
		if err := cfg.Validate(); err != nil {
			return fmt.Errorf("invalid config: %w", err)
		}

		logger := slog.Default()

		pg, err := database.OpenPostgres(cfg.PostgresDSN, logger)
		if err != nil {
			return fmt.Errorf("open postgres: %w", err)
		}
		defer pg.Close()

		if err := database.Migrate(context.Background(), pg, logger); err != nil {
			return fmt.Errorf("migrate: %w", err)
		}

		redisClient, err := database.OpenRedis(cfg.RedisAddr, cfg.RedisPass, cfg.RedisDB, logger)
		if err != nil {
			return fmt.Errorf("open redis: %w", err)
		}
		defer redisClient.Close()

		boltDB, err := database.OpenBolt(cfg.BoltPath, logger)
		if err != nil {
			return fmt.Errorf("open bolt: %w", err)
		}
		defer boltDB.Close()

		auditLogger := database.NewBoltAuditLogger(boltDB, logger)

		var (
			userRepo domain.UserRepository
			taskRepo domain.TaskRepository
		)

		// PostgreSQL remains the source of truth for core domain data.
		userRepo = database.NewPgUserRepo(pg)
		taskRepo = database.NewPgTaskRepo(pg)

		userSvc := service.NewUserService(userRepo, logger)
		taskSvc := service.NewTaskService(taskRepo, userRepo, logger)

		srv := server.New(cfg, logger, taskSvc, userSvc, redisClient, auditLogger)
		return srv.Run()
	},
}

func init() {
	rootCmd.AddCommand(serverCmd)
}
