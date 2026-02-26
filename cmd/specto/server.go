package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

var serverCmd = &cobra.Command{
	Use:   "server",
	Short: "Start the Specto HTTP server",
	Long:  "Launches the Specto web server on the configured address and port.",
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("server: not yet implemented")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(serverCmd)
}
