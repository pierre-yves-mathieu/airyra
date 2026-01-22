package main

import (
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"text/tabwriter"

	"github.com/airyra/airyra/internal/client"
	"github.com/airyra/airyra/internal/domain"
)

// printTask prints a single task to the writer
func printTask(w io.Writer, task *domain.Task, jsonOutput bool) {
	if jsonOutput {
		enc := json.NewEncoder(w)
		enc.SetIndent("", "  ")
		enc.Encode(task)
		return
	}

	// Table format
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	fmt.Fprintf(tw, "ID:\t%s\n", task.ID)
	fmt.Fprintf(tw, "Title:\t%s\n", task.Title)
	fmt.Fprintf(tw, "Status:\t%s\n", task.Status)
	fmt.Fprintf(tw, "Priority:\t%s\n", priorityString(task.Priority))
	if task.Description != nil && *task.Description != "" {
		fmt.Fprintf(tw, "Description:\t%s\n", *task.Description)
	}
	if task.ParentID != nil && *task.ParentID != "" {
		fmt.Fprintf(tw, "Parent:\t%s\n", *task.ParentID)
	}
	if task.ClaimedBy != nil && *task.ClaimedBy != "" {
		fmt.Fprintf(tw, "Claimed By:\t%s\n", *task.ClaimedBy)
	}
	if task.ClaimedAt != nil {
		fmt.Fprintf(tw, "Claimed At:\t%s\n", task.ClaimedAt.Format("2006-01-02 15:04:05"))
	}
	fmt.Fprintf(tw, "Created:\t%s\n", task.CreatedAt.Format("2006-01-02 15:04:05"))
	fmt.Fprintf(tw, "Updated:\t%s\n", task.UpdatedAt.Format("2006-01-02 15:04:05"))
	tw.Flush()
}

// printTaskList prints a list of tasks with pagination info
func printTaskList(w io.Writer, tasks []*domain.Task, pagination *client.Pagination, jsonOutput bool) {
	if jsonOutput {
		enc := json.NewEncoder(w)
		enc.SetIndent("", "  ")
		enc.Encode(map[string]interface{}{
			"data": tasks,
			"pagination": map[string]interface{}{
				"page":        pagination.Page,
				"per_page":    pagination.PerPage,
				"total":       pagination.Total,
				"total_pages": pagination.TotalPages,
			},
		})
		return
	}

	if len(tasks) == 0 {
		fmt.Fprintln(w, "No tasks found")
		return
	}

	// Table format
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	fmt.Fprintf(tw, "ID\tTITLE\tSTATUS\tPRIORITY\n")
	fmt.Fprintf(tw, "--\t-----\t------\t--------\n")
	for _, task := range tasks {
		fmt.Fprintf(tw, "%s\t%s\t%s\t%s\n",
			task.ID, truncate(task.Title, 40), task.Status, priorityString(task.Priority))
	}
	tw.Flush()

	// Pagination info
	if pagination.TotalPages > 1 {
		fmt.Fprintf(w, "\nPage %d of %d (%d total tasks)\n",
			pagination.Page, pagination.TotalPages, pagination.Total)
	}
}

// printDependencies prints task dependencies
func printDependencies(w io.Writer, taskID string, deps []domain.Dependency, jsonOutput bool) {
	if jsonOutput {
		enc := json.NewEncoder(w)
		enc.SetIndent("", "  ")
		enc.Encode(deps)
		return
	}

	if len(deps) == 0 {
		fmt.Fprintf(w, "Task %s has no dependencies\n", taskID)
		return
	}

	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	fmt.Fprintf(tw, "TYPE\tTASK ID\n")
	fmt.Fprintf(tw, "----\t-------\n")
	for _, dep := range deps {
		if dep.ChildID == taskID {
			fmt.Fprintf(tw, "depends on\t%s\n", dep.ParentID)
		} else if dep.ParentID == taskID {
			fmt.Fprintf(tw, "blocks\t%s\n", dep.ChildID)
		}
	}
	tw.Flush()
}

// printHistory prints task history/audit entries
func printHistory(w io.Writer, entries []domain.AuditEntry, jsonOutput bool) {
	if jsonOutput {
		enc := json.NewEncoder(w)
		enc.SetIndent("", "  ")
		enc.Encode(entries)
		return
	}

	if len(entries) == 0 {
		fmt.Fprintln(w, "No history found")
		return
	}

	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	fmt.Fprintf(tw, "TIME\tACTION\tFIELD\tOLD\tNEW\tBY\n")
	fmt.Fprintf(tw, "----\t------\t-----\t---\t---\t--\n")
	for _, entry := range entries {
		field := ""
		if entry.Field != nil {
			field = *entry.Field
		}
		oldVal := ""
		if entry.OldValue != nil {
			oldVal = truncate(*entry.OldValue, 20)
		}
		newVal := ""
		if entry.NewValue != nil {
			newVal = truncate(*entry.NewValue, 20)
		}
		fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t%s\t%s\n",
			entry.ChangedAt.Format("2006-01-02 15:04:05"),
			entry.Action,
			field,
			oldVal,
			newVal,
			truncate(entry.ChangedBy, 30))
	}
	tw.Flush()
}

// printError prints an error message
func printError(w io.Writer, err error, jsonOutput bool) {
	if jsonOutput {
		enc := json.NewEncoder(w)
		enc.SetIndent("", "  ")
		enc.Encode(map[string]interface{}{
			"error": map[string]interface{}{
				"message": err.Error(),
			},
		})
		return
	}

	fmt.Fprintf(w, "Error: %s\n", err.Error())
}

// printSuccess prints a success message
func printSuccess(w io.Writer, message string, jsonOutput bool) {
	if jsonOutput {
		enc := json.NewEncoder(w)
		enc.SetIndent("", "  ")
		enc.Encode(map[string]interface{}{
			"message": message,
		})
		return
	}

	fmt.Fprintln(w, message)
}

// priorityString converts a priority int to a human-readable string
func priorityString(priority int) string {
	switch priority {
	case 0:
		return "critical"
	case 1:
		return "high"
	case 2:
		return "normal"
	case 3:
		return "low"
	case 4:
		return "lowest"
	default:
		return strconv.Itoa(priority)
	}
}

// truncate truncates a string to the specified length
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

// printSpec prints a single spec to the writer
func printSpec(w io.Writer, spec *client.Spec, jsonOutput bool) {
	if jsonOutput {
		enc := json.NewEncoder(w)
		enc.SetIndent("", "  ")
		enc.Encode(spec)
		return
	}

	// Table format
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	fmt.Fprintf(tw, "ID:\t%s\n", spec.ID)
	fmt.Fprintf(tw, "Title:\t%s\n", spec.Title)
	fmt.Fprintf(tw, "Status:\t%s\n", spec.Status)
	if spec.Description != nil && *spec.Description != "" {
		fmt.Fprintf(tw, "Description:\t%s\n", *spec.Description)
	}
	fmt.Fprintf(tw, "Tasks:\t%d/%d done\n", spec.DoneCount, spec.TaskCount)
	fmt.Fprintf(tw, "Created:\t%s\n", spec.CreatedAt)
	fmt.Fprintf(tw, "Updated:\t%s\n", spec.UpdatedAt)
	tw.Flush()
}

// printSpecList prints a list of specs with pagination info
func printSpecList(w io.Writer, specs []*client.Spec, pagination *client.Pagination, jsonOutput bool) {
	if jsonOutput {
		enc := json.NewEncoder(w)
		enc.SetIndent("", "  ")
		enc.Encode(map[string]interface{}{
			"data": specs,
			"pagination": map[string]interface{}{
				"page":        pagination.Page,
				"per_page":    pagination.PerPage,
				"total":       pagination.Total,
				"total_pages": pagination.TotalPages,
			},
		})
		return
	}

	if len(specs) == 0 {
		fmt.Fprintln(w, "No specs found")
		return
	}

	// Table format
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	fmt.Fprintf(tw, "ID\tTITLE\tSTATUS\tPROGRESS\n")
	fmt.Fprintf(tw, "--\t-----\t------\t--------\n")
	for _, spec := range specs {
		progress := fmt.Sprintf("%d/%d", spec.DoneCount, spec.TaskCount)
		fmt.Fprintf(tw, "%s\t%s\t%s\t%s\n",
			spec.ID, truncate(spec.Title, 40), spec.Status, progress)
	}
	tw.Flush()

	// Pagination info
	if pagination.TotalPages > 1 {
		fmt.Fprintf(w, "\nPage %d of %d (%d total specs)\n",
			pagination.Page, pagination.TotalPages, pagination.Total)
	}
}

// printSpecDependencies prints spec dependencies
func printSpecDependencies(w io.Writer, specID string, deps []client.SpecDependency, jsonOutput bool) {
	if jsonOutput {
		enc := json.NewEncoder(w)
		enc.SetIndent("", "  ")
		enc.Encode(deps)
		return
	}

	if len(deps) == 0 {
		fmt.Fprintf(w, "Spec %s has no dependencies\n", specID)
		return
	}

	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	fmt.Fprintf(tw, "TYPE\tSPEC ID\n")
	fmt.Fprintf(tw, "----\t-------\n")
	for _, dep := range deps {
		if dep.ChildID == specID {
			fmt.Fprintf(tw, "depends on\t%s\n", dep.ParentID)
		} else if dep.ParentID == specID {
			fmt.Fprintf(tw, "blocks\t%s\n", dep.ChildID)
		}
	}
	tw.Flush()
}
