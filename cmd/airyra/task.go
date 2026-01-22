package main

import (
	"context"
	"fmt"
	"os"

	"github.com/airyra/airyra/internal/client"
	"github.com/spf13/cobra"
)

var createCmd = &cobra.Command{
	Use:   "create <title>",
	Short: "Create a new task",
	Long: `Create a new task with the given title.

Priority can be specified as a number (0-4) or name:
  0 / critical - Highest priority
  1 / high     - High priority
  2 / normal   - Normal priority (default)
  3 / low      - Low priority
  4 / lowest   - Lowest priority`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		if err := validateCreateArgs(args); err != nil {
			handleError(err)
		}

		priorityStr, _ := cmd.Flags().GetString("priority")
		description, _ := cmd.Flags().GetString("description")
		parentID, _ := cmd.Flags().GetString("parent")
		specID, _ := cmd.Flags().GetString("spec")

		priority := 2 // default
		if priorityStr != "" {
			p, err := parsePriority(priorityStr)
			if err != nil {
				handleError(err)
			}
			priority = p
		}

		c, err := getClient()
		if err != nil {
			handleError(err)
		}

		task, err := c.CreateTask(context.Background(), args[0], description, priority, parentID, specID)
		if err != nil {
			handleError(err)
		}

		printTask(os.Stdout, task, jsonOutput)
	},
}

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List tasks",
	Long:  `List tasks with optional filtering by status.`,
	Run: func(cmd *cobra.Command, args []string) {
		status, _ := cmd.Flags().GetString("status")
		page, _ := cmd.Flags().GetInt("page")
		perPage, _ := cmd.Flags().GetInt("per-page")

		c, err := getClient()
		if err != nil {
			handleError(err)
		}

		result, err := c.ListTasks(context.Background(), status, page, perPage)
		if err != nil {
			handleError(err)
		}

		printTaskList(os.Stdout, result.Data, result.Pagination, jsonOutput)
	},
}

var showCmd = &cobra.Command{
	Use:   "show <id>",
	Short: "Show task details",
	Long:  `Display detailed information about a task.`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		c, err := getClient()
		if err != nil {
			handleError(err)
		}

		task, err := c.GetTask(context.Background(), args[0])
		if err != nil {
			handleError(err)
		}

		printTask(os.Stdout, task, jsonOutput)
	},
}

var editCmd = &cobra.Command{
	Use:   "edit <id>",
	Short: "Edit a task",
	Long:  `Edit a task's title, description, or priority.`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		title, _ := cmd.Flags().GetString("title")
		description, _ := cmd.Flags().GetString("description")
		priorityStr, _ := cmd.Flags().GetString("priority")

		var updates client.TaskUpdates

		if cmd.Flags().Changed("title") {
			updates.Title = &title
		}
		if cmd.Flags().Changed("description") {
			updates.Description = &description
		}
		if cmd.Flags().Changed("priority") {
			p, err := parsePriority(priorityStr)
			if err != nil {
				handleError(err)
			}
			updates.Priority = &p
		}

		c, err := getClient()
		if err != nil {
			handleError(err)
		}

		task, err := c.UpdateTask(context.Background(), args[0], updates)
		if err != nil {
			handleError(err)
		}

		printTask(os.Stdout, task, jsonOutput)
	},
}

var deleteCmd = &cobra.Command{
	Use:   "delete <id>",
	Short: "Delete a task",
	Long:  `Delete a task by ID.`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		c, err := getClient()
		if err != nil {
			handleError(err)
		}

		err = c.DeleteTask(context.Background(), args[0])
		if err != nil {
			handleError(err)
		}

		printSuccess(os.Stdout, fmt.Sprintf("Task %s deleted", args[0]), jsonOutput)
	},
}

func init() {
	rootCmd.AddCommand(createCmd)
	rootCmd.AddCommand(listCmd)
	rootCmd.AddCommand(showCmd)
	rootCmd.AddCommand(editCmd)
	rootCmd.AddCommand(deleteCmd)

	// Create command flags
	createCmd.Flags().StringP("priority", "p", "", "Task priority (0-4 or critical/high/normal/low/lowest)")
	createCmd.Flags().StringP("description", "d", "", "Task description")
	createCmd.Flags().String("parent", "", "Parent task ID")
	createCmd.Flags().String("spec", "", "Spec ID to assign task to")

	// List command flags
	listCmd.Flags().String("status", "", "Filter by status (open, in_progress, blocked, done)")
	listCmd.Flags().Int("page", 1, "Page number")
	listCmd.Flags().Int("per-page", 50, "Items per page")

	// Edit command flags
	editCmd.Flags().StringP("title", "t", "", "New title")
	editCmd.Flags().StringP("description", "d", "", "New description")
	editCmd.Flags().StringP("priority", "p", "", "New priority")
}

// validateCreateArgs validates the arguments for the create command
func validateCreateArgs(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("title is required")
	}
	if args[0] == "" {
		return fmt.Errorf("title cannot be empty")
	}
	return nil
}
