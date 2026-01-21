package main

import (
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "airyra",
	Short: "Airyra task tracker",
	Long:  `A task tracker for AI agent coordination with atomic task claiming.`,
}

// Global flags
var jsonOutput bool

func init() {
	rootCmd.PersistentFlags().BoolVar(&jsonOutput, "json", false, "Output as JSON")
}

// Execute runs the root command
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(ExitGeneralError)
	}
}
