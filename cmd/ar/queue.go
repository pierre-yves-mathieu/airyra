package main

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var readyCmd = &cobra.Command{
	Use:   "ready",
	Short: "List ready tasks",
	Long: `List all tasks that are ready to be worked on.

Ready tasks are open tasks with no unfinished dependencies.
Tasks are sorted by priority (highest first).`,
	Run: func(cmd *cobra.Command, args []string) {
		page, _ := cmd.Flags().GetInt("page")
		perPage, _ := cmd.Flags().GetInt("per-page")

		c, err := getClient()
		if err != nil {
			handleError(err)
		}

		result, err := c.ListReadyTasks(context.Background(), page, perPage)
		if err != nil {
			handleError(err)
		}

		printTaskList(os.Stdout, result.Data, result.Pagination, jsonOutput)
	},
}

var nextCmd = &cobra.Command{
	Use:   "next",
	Short: "Get the next task to work on",
	Long: `Get the highest-priority ready task.

This returns the single most important task that is ready to be worked on.`,
	Run: func(cmd *cobra.Command, args []string) {
		c, err := getClient()
		if err != nil {
			handleError(err)
		}

		result, err := c.ListReadyTasks(context.Background(), 1, 1)
		if err != nil {
			handleError(err)
		}

		if len(result.Data) == 0 {
			printSuccess(os.Stdout, "No ready tasks", jsonOutput)
			return
		}

		printTask(os.Stdout, result.Data[0], jsonOutput)

		if !jsonOutput && result.Pagination.Total > 1 {
			fmt.Fprintf(os.Stderr, "\n(%d more ready tasks)\n", result.Pagination.Total-1)
		}
	},
}

func init() {
	rootCmd.AddCommand(readyCmd)
	rootCmd.AddCommand(nextCmd)

	readyCmd.Flags().Int("page", 1, "Page number")
	readyCmd.Flags().Int("per-page", 50, "Items per page")
}
