package main

import (
	"context"
	"os"

	"github.com/spf13/cobra"
)

var claimCmd = &cobra.Command{
	Use:   "claim <id>",
	Short: "Claim a task",
	Long:  `Claim a task to start working on it. Changes status from open to in_progress.`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		c, err := getClient()
		if err != nil {
			handleError(err)
		}

		task, err := c.ClaimTask(context.Background(), args[0])
		if err != nil {
			handleError(err)
		}

		printTask(os.Stdout, task, jsonOutput)
	},
}

var doneCmd = &cobra.Command{
	Use:   "done <id>",
	Short: "Mark a task as done",
	Long:  `Mark a task as complete. Changes status from in_progress to done.`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		c, err := getClient()
		if err != nil {
			handleError(err)
		}

		task, err := c.CompleteTask(context.Background(), args[0])
		if err != nil {
			handleError(err)
		}

		printTask(os.Stdout, task, jsonOutput)
	},
}

var releaseCmd = &cobra.Command{
	Use:   "release <id>",
	Short: "Release a claimed task",
	Long: `Release a task you have claimed. Changes status from in_progress to open.

Use --force to release a task claimed by another agent.`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		force, _ := cmd.Flags().GetBool("force")

		c, err := getClient()
		if err != nil {
			handleError(err)
		}

		task, err := c.ReleaseTask(context.Background(), args[0], force)
		if err != nil {
			handleError(err)
		}

		printTask(os.Stdout, task, jsonOutput)
	},
}

var blockCmd = &cobra.Command{
	Use:   "block <id>",
	Short: "Block a task",
	Long:  `Mark a task as blocked. Changes status from in_progress to blocked.`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		c, err := getClient()
		if err != nil {
			handleError(err)
		}

		task, err := c.BlockTask(context.Background(), args[0])
		if err != nil {
			handleError(err)
		}

		printTask(os.Stdout, task, jsonOutput)
	},
}

var unblockCmd = &cobra.Command{
	Use:   "unblock <id>",
	Short: "Unblock a task",
	Long:  `Unblock a blocked task. Changes status from blocked to open.`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		c, err := getClient()
		if err != nil {
			handleError(err)
		}

		task, err := c.UnblockTask(context.Background(), args[0])
		if err != nil {
			handleError(err)
		}

		printTask(os.Stdout, task, jsonOutput)
	},
}

func init() {
	rootCmd.AddCommand(claimCmd)
	rootCmd.AddCommand(doneCmd)
	rootCmd.AddCommand(releaseCmd)
	rootCmd.AddCommand(blockCmd)
	rootCmd.AddCommand(unblockCmd)

	releaseCmd.Flags().Bool("force", false, "Force release a task claimed by another agent")
}
