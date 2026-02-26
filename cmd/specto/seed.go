package main

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/rxritet/Specto/internal/config"
	"github.com/rxritet/Specto/internal/database"
	"github.com/rxritet/Specto/internal/domain"
	"github.com/spf13/cobra"
)

var seedCmd = &cobra.Command{
	Use:   "seed",
	Short: "Populate the database with test fixture data",
	Long:  "Loads fixtures.sql (Postgres) or creates fixture objects via repositories (BoltDB).",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg := config.Load()
		logger := slog.Default()

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
			return database.Seed(context.Background(), db, logger)

		case "bolt":
			bdb, err := database.OpenBolt(cfg.DBPath, logger)
			if err != nil {
				return fmt.Errorf("open bolt: %w", err)
			}
			defer bdb.Close()

			return seedBolt(
				database.NewBoltUserRepo(bdb),
				database.NewBoltTaskRepo(bdb),
				logger,
			)

		default:
			return fmt.Errorf("unknown db provider: %s", cfg.DBProvider)
		}
	},
}

func init() {
	rootCmd.AddCommand(seedCmd)
}

// seedBolt creates fixture data via BoltDB repositories.
func seedBolt(users domain.UserRepository, tasks domain.TaskRepository, logger *slog.Logger) error {
	fixtureUsers := []domain.User{
		{Email: "alice@example.com", Name: "Alice", Password: "$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy"},
		{Email: "bob@example.com", Name: "Bob", Password: "$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy"},
	}

	for i := range fixtureUsers {
		// Skip if user already exists.
		if _, err := users.GetByEmail(fixtureUsers[i].Email); err == nil {
			logger.Info("seed: user already exists, skipping", "email", fixtureUsers[i].Email)
			continue
		}
		if err := users.Create(&fixtureUsers[i]); err != nil {
			return fmt.Errorf("seed user %s: %w", fixtureUsers[i].Email, err)
		}
		logger.Info("seed: created user", "email", fixtureUsers[i].Email, "id", fixtureUsers[i].ID)
	}

	alice, _ := users.GetByEmail("alice@example.com")
	bob, _ := users.GetByEmail("bob@example.com")

	fixtureTasks := []domain.Task{
		{UserID: alice.ID, Title: "Buy groceries", Description: "Milk, eggs, bread", Status: domain.TaskStatusTodo},
		{UserID: alice.ID, Title: "Write report", Description: "Q4 financial summary", Status: domain.TaskStatusInProgress},
		{UserID: bob.ID, Title: "Fix landing page", Description: "Update hero section copy", Status: domain.TaskStatusTodo},
		{UserID: bob.ID, Title: "Deploy v2", Description: "Tag release and deploy", Status: domain.TaskStatusDone},
	}

	for i := range fixtureTasks {
		if err := tasks.Create(&fixtureTasks[i]); err != nil {
			return fmt.Errorf("seed task %q: %w", fixtureTasks[i].Title, err)
		}
		logger.Info("seed: created task", "title", fixtureTasks[i].Title)
	}

	logger.Info("bolt seed data applied")
	return nil
}
