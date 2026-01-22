package main

import (
	"context"
	"fmt"
	"os"

	"github.com/airyra/airyra/internal/client"
	"github.com/spf13/cobra"
)

var specCmd = &cobra.Command{
	Use:   "spec",
	Short: "Manage specs (epics)",
	Long:  `Commands for managing specs - epic-like entities for grouping related tasks.`,
}

var specNewCmd = &cobra.Command{
	Use:   "new <title>",
	Short: "Create a new spec",
	Long:  `Create a new spec with the given title.`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		description, _ := cmd.Flags().GetString("description")

		c, err := getClient()
		if err != nil {
			handleError(err)
		}

		spec, err := c.CreateSpec(context.Background(), args[0], description)
		if err != nil {
			handleError(err)
		}

		printSpec(os.Stdout, spec, jsonOutput)
	},
}

var specListCmd = &cobra.Command{
	Use:   "list",
	Short: "List specs",
	Long:  `List specs with optional filtering by status.`,
	Run: func(cmd *cobra.Command, args []string) {
		status, _ := cmd.Flags().GetString("status")
		page, _ := cmd.Flags().GetInt("page")
		perPage, _ := cmd.Flags().GetInt("per-page")

		c, err := getClient()
		if err != nil {
			handleError(err)
		}

		result, err := c.ListSpecs(context.Background(), status, page, perPage)
		if err != nil {
			handleError(err)
		}

		printSpecList(os.Stdout, result.Data, result.Pagination, jsonOutput)
	},
}

var specShowCmd = &cobra.Command{
	Use:   "show <id>",
	Short: "Show spec details",
	Long:  `Display detailed information about a spec.`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		showTasks, _ := cmd.Flags().GetBool("tasks")

		c, err := getClient()
		if err != nil {
			handleError(err)
		}

		spec, err := c.GetSpec(context.Background(), args[0])
		if err != nil {
			handleError(err)
		}

		printSpec(os.Stdout, spec, jsonOutput)

		if showTasks {
			fmt.Fprintln(os.Stdout)
			fmt.Fprintln(os.Stdout, "Tasks:")
			tasks, err := c.ListSpecTasks(context.Background(), args[0], 1, 100)
			if err != nil {
				handleError(err)
			}
			printTaskList(os.Stdout, tasks.Data, tasks.Pagination, jsonOutput)
		}
	},
}

var specEditCmd = &cobra.Command{
	Use:   "edit <id>",
	Short: "Edit a spec",
	Long:  `Edit a spec's title or description.`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		title, _ := cmd.Flags().GetString("title")
		description, _ := cmd.Flags().GetString("description")

		var updates client.SpecUpdates

		if cmd.Flags().Changed("title") {
			updates.Title = &title
		}
		if cmd.Flags().Changed("description") {
			updates.Description = &description
		}

		c, err := getClient()
		if err != nil {
			handleError(err)
		}

		spec, err := c.UpdateSpec(context.Background(), args[0], updates)
		if err != nil {
			handleError(err)
		}

		printSpec(os.Stdout, spec, jsonOutput)
	},
}

var specCancelCmd = &cobra.Command{
	Use:   "cancel <id>",
	Short: "Cancel a spec",
	Long:  `Cancel a spec. Cancelled specs have status 'cancelled'.`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		c, err := getClient()
		if err != nil {
			handleError(err)
		}

		spec, err := c.CancelSpec(context.Background(), args[0])
		if err != nil {
			handleError(err)
		}

		printSpec(os.Stdout, spec, jsonOutput)
	},
}

var specReopenCmd = &cobra.Command{
	Use:   "reopen <id>",
	Short: "Reopen a cancelled spec",
	Long:  `Reopen a cancelled spec to make it active again.`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		c, err := getClient()
		if err != nil {
			handleError(err)
		}

		spec, err := c.ReopenSpec(context.Background(), args[0])
		if err != nil {
			handleError(err)
		}

		printSpec(os.Stdout, spec, jsonOutput)
	},
}

var specDeleteCmd = &cobra.Command{
	Use:   "delete <id>",
	Short: "Delete a spec",
	Long:  `Delete a spec by ID. Tasks belonging to the spec will have their spec_id cleared.`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		c, err := getClient()
		if err != nil {
			handleError(err)
		}

		err = c.DeleteSpec(context.Background(), args[0])
		if err != nil {
			handleError(err)
		}

		printSuccess(os.Stdout, fmt.Sprintf("Spec %s deleted", args[0]), jsonOutput)
	},
}

// Spec dependency commands
var specDepCmd = &cobra.Command{
	Use:   "dep",
	Short: "Manage spec dependencies",
	Long:  `Commands for managing spec dependencies.`,
}

var specDepAddCmd = &cobra.Command{
	Use:   "add <child-id> <parent-id>",
	Short: "Add a spec dependency",
	Long:  `Add a dependency where child spec depends on parent spec.`,
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		c, err := getClient()
		if err != nil {
			handleError(err)
		}

		err = c.AddSpecDependency(context.Background(), args[0], args[1])
		if err != nil {
			handleError(err)
		}

		printSuccess(os.Stdout, fmt.Sprintf("Dependency added: %s depends on %s", args[0], args[1]), jsonOutput)
	},
}

var specDepRmCmd = &cobra.Command{
	Use:   "rm <child-id> <parent-id>",
	Short: "Remove a spec dependency",
	Long:  `Remove a dependency between specs.`,
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		c, err := getClient()
		if err != nil {
			handleError(err)
		}

		err = c.RemoveSpecDependency(context.Background(), args[0], args[1])
		if err != nil {
			handleError(err)
		}

		printSuccess(os.Stdout, fmt.Sprintf("Dependency removed: %s no longer depends on %s", args[0], args[1]), jsonOutput)
	},
}

var specDepListCmd = &cobra.Command{
	Use:   "list <id>",
	Short: "List spec dependencies",
	Long:  `List all dependencies for a spec.`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		c, err := getClient()
		if err != nil {
			handleError(err)
		}

		deps, err := c.ListSpecDependencies(context.Background(), args[0])
		if err != nil {
			handleError(err)
		}

		printSpecDependencies(os.Stdout, args[0], deps, jsonOutput)
	},
}

func init() {
	rootCmd.AddCommand(specCmd)
	specCmd.AddCommand(specNewCmd)
	specCmd.AddCommand(specListCmd)
	specCmd.AddCommand(specShowCmd)
	specCmd.AddCommand(specEditCmd)
	specCmd.AddCommand(specCancelCmd)
	specCmd.AddCommand(specReopenCmd)
	specCmd.AddCommand(specDeleteCmd)
	specCmd.AddCommand(specDepCmd)

	specDepCmd.AddCommand(specDepAddCmd)
	specDepCmd.AddCommand(specDepRmCmd)
	specDepCmd.AddCommand(specDepListCmd)

	// New command flags
	specNewCmd.Flags().StringP("description", "d", "", "Spec description")

	// List command flags
	specListCmd.Flags().String("status", "", "Filter by status (draft, active, done, cancelled)")
	specListCmd.Flags().Int("page", 1, "Page number")
	specListCmd.Flags().Int("per-page", 50, "Items per page")

	// Show command flags
	specShowCmd.Flags().Bool("tasks", false, "Show tasks belonging to spec")

	// Edit command flags
	specEditCmd.Flags().StringP("title", "t", "", "New title")
	specEditCmd.Flags().StringP("description", "d", "", "New description")
}
