package main

import (
	"fmt"
	"os"

	"github.com/airyra/airyra/internal/config"
	"github.com/spf13/cobra"
)

var initCmd = &cobra.Command{
	Use:   "init <name>",
	Short: "Initialize a new airyra project",
	Long: `Create an airyra.toml configuration file in the current directory.

The project name is used to identify this project when communicating with
the airyra server.`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		host, _ := cmd.Flags().GetString("host")
		port, _ := cmd.Flags().GetInt("port")

		if err := runInit(args[0], host, port); err != nil {
			handleError(err)
		}

		printSuccess(os.Stdout, fmt.Sprintf("Created %s for project '%s'", config.ConfigFileName, args[0]), jsonOutput)
	},
}

func init() {
	rootCmd.AddCommand(initCmd)

	initCmd.Flags().String("host", "", "Server host")
	initCmd.Flags().Int("port", 0, "Server port")
}

// runInit creates the airyra.toml configuration file
func runInit(name, host string, port int) error {
	if name == "" {
		return fmt.Errorf("project name is required")
	}

	// Check if config already exists
	configPath := config.ConfigFileName
	if _, err := os.Stat(configPath); err == nil {
		return fmt.Errorf("%s already exists in this directory", config.ConfigFileName)
	}

	// Build config content
	content := fmt.Sprintf("project = %q\n", name)

	if host != "" || port != 0 {
		content += "\n[server]\n"
		if host != "" {
			content += fmt.Sprintf("host = %q\n", host)
		}
		if port != 0 {
			content += fmt.Sprintf("port = %d\n", port)
		}
	}

	// Write config file
	if err := os.WriteFile(configPath, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}
