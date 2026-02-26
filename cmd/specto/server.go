package main

import (
	"log/slog"

	"github.com/rxritet/Specto/internal/config"
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

		srv := server.New(cfg, logger)
		return srv.Run()
	},
}

func init() {
	rootCmd.AddCommand(serverCmd)
}
