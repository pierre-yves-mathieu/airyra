package main

import (
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "ar",
	Short: "Airyra task tracker CLI",
	Long:  `A CLI for the Airyra distributed task tracking system.`,
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
