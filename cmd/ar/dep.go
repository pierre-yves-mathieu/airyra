package main

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var depCmd = &cobra.Command{
	Use:   "dep",
	Short: "Manage task dependencies",
	Long:  `Commands for managing dependencies between tasks.`,
}

var depAddCmd = &cobra.Command{
	Use:   "add <child> <parent>",
	Short: "Add a dependency",
	Long: `Add a dependency between two tasks.

The child task will depend on the parent task. The child task cannot
be started until the parent task is completed.`,
	Args: cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		childID := args[0]
		parentID := args[1]

		c, err := getClient()
		if err != nil {
			handleError(err)
		}

		err = c.AddDependency(context.Background(), childID, parentID)
		if err != nil {
			handleError(err)
		}

		printSuccess(os.Stdout, fmt.Sprintf("Added dependency: %s depends on %s", childID, parentID), jsonOutput)
	},
}

var depRmCmd = &cobra.Command{
	Use:   "rm <child> <parent>",
	Short: "Remove a dependency",
	Long:  `Remove a dependency between two tasks.`,
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		childID := args[0]
		parentID := args[1]

		c, err := getClient()
		if err != nil {
			handleError(err)
		}

		err = c.RemoveDependency(context.Background(), childID, parentID)
		if err != nil {
			handleError(err)
		}

		printSuccess(os.Stdout, fmt.Sprintf("Removed dependency: %s no longer depends on %s", childID, parentID), jsonOutput)
	},
}

var depListCmd = &cobra.Command{
	Use:   "list <id>",
	Short: "List dependencies",
	Long:  `List all dependencies for a task.`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		taskID := args[0]

		c, err := getClient()
		if err != nil {
			handleError(err)
		}

		deps, err := c.ListDependencies(context.Background(), taskID)
		if err != nil {
			handleError(err)
		}

		printDependencies(os.Stdout, taskID, deps, jsonOutput)
	},
}

func init() {
	rootCmd.AddCommand(depCmd)

	depCmd.AddCommand(depAddCmd)
	depCmd.AddCommand(depRmCmd)
	depCmd.AddCommand(depListCmd)
}
