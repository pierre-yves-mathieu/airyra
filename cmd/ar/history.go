package main

import (
	"context"
	"os"

	"github.com/spf13/cobra"
)

var historyCmd = &cobra.Command{
	Use:   "history <id>",
	Short: "Show task history",
	Long:  `Display the audit history for a specific task.`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		taskID := args[0]

		c, err := getClient()
		if err != nil {
			handleError(err)
		}

		entries, err := c.GetTaskHistory(context.Background(), taskID)
		if err != nil {
			handleError(err)
		}

		printHistory(os.Stdout, entries, jsonOutput)
	},
}

var logCmd = &cobra.Command{
	Use:   "log",
	Short: "Show recent activity",
	Long: `Display recent activity across all tasks.

This command lists recent audit entries to show what has been happening
in the project.`,
	Run: func(cmd *cobra.Command, args []string) {
		// The API doesn't have a dedicated "recent activity" endpoint,
		// so we need to fetch tasks and show a summary. For now, we'll
		// display a message indicating this limitation.
		c, err := getClient()
		if err != nil {
			handleError(err)
		}

		// Get recent tasks (sorted by updated_at typically)
		result, err := c.ListTasks(context.Background(), "", 1, 10)
		if err != nil {
			handleError(err)
		}

		if len(result.Data) == 0 {
			printSuccess(os.Stdout, "No recent activity", jsonOutput)
			return
		}

		// Show recently updated tasks as a proxy for activity
		printTaskList(os.Stdout, result.Data, result.Pagination, jsonOutput)
	},
}

func init() {
	rootCmd.AddCommand(historyCmd)
	rootCmd.AddCommand(logCmd)
}
