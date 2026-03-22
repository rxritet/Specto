package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/rxritet/Specto/internal/config"
	"github.com/rxritet/Specto/internal/database"
	"github.com/rxritet/Specto/internal/domain"
	"github.com/rxritet/Specto/internal/service"
	"github.com/spf13/cobra"
)

var seedCmd = &cobra.Command{
	Use:   "seed",
	Short: "Populate the database with test fixture data",
	Long:  "Creates fixture users and tasks through the service layer.",
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

		userSvc := service.NewUserService(userRepo, logger)
		taskSvc := service.NewTaskService(taskRepo, userRepo, logger)

		return seedData(userSvc, taskSvc, logger)
	},
}

func init() {
	rootCmd.AddCommand(seedCmd)
}

// seedData creates fixture users and tasks through the service layer.
func seedData(userSvc *service.UserService, taskSvc *service.TaskService, logger *slog.Logger) error {
	fixtureUsers := []domain.User{
		{Email: "alice@example.com", Name: "Alice", Password: "$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy"},
		{Email: "bob@example.com", Name: "Bob", Password: "$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy"},
	}

	for i := range fixtureUsers {
		if err := userSvc.Create(context.Background(), &fixtureUsers[i]); err != nil {
			// Skip duplicates; the service returns a ConflictError.
			if _, ok := errors.AsType[*domain.ConflictError](err); ok {
				logger.Info("seed: user already exists, skipping", "email", fixtureUsers[i].Email)
				continue
			}
			return fmt.Errorf("seed user %s: %w", fixtureUsers[i].Email, err)
		}
	}

	alice, err := userSvc.GetByEmail(context.Background(), "alice@example.com")
	if err != nil {
		return fmt.Errorf("lookup alice: %w", err)
	}
	bob, err := userSvc.GetByEmail(context.Background(), "bob@example.com")
	if err != nil {
		return fmt.Errorf("lookup bob: %w", err)
	}

	fixtureTasks := []domain.Task{
		{UserID: alice.ID, Title: "Buy groceries", Description: "Milk, eggs, bread", Status: domain.TaskStatusTodo},
		{UserID: alice.ID, Title: "Write report", Description: "Q4 financial summary", Status: domain.TaskStatusInProgress},
		{UserID: bob.ID, Title: "Fix landing page", Description: "Update hero section copy", Status: domain.TaskStatusTodo},
		{UserID: bob.ID, Title: "Deploy v2", Description: "Tag release and deploy", Status: domain.TaskStatusDone},
	}

	for i := range fixtureTasks {
		if err := taskSvc.Create(context.Background(), &fixtureTasks[i]); err != nil {
			return fmt.Errorf("seed task %q: %w", fixtureTasks[i].Title, err)
		}
	}

	logger.Info("seed data applied", "users", len(fixtureUsers), "tasks", len(fixtureTasks))
	return nil
}
