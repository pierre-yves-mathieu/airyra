// Package airyra provides a Go SDK for interacting with the Airyra task management server.
//
// Airyra is a lightweight task management system designed for AI agents and automation
// workflows. This SDK allows Go applications to use Airyra as a library dependency.
//
// # Getting Started
//
// First, ensure the Airyra server is running. Then create a client:
//
//	client, err := airyra.NewClient(
//	    airyra.WithProject("my-project"),
//	    airyra.WithAgentID("agent-001"),
//	)
//	if err != nil {
//	    log.Fatal(err)
//	}
//
// # Creating and Managing Tasks
//
// Create a new task:
//
//	task, err := client.CreateTask(ctx, "Implement feature X",
//	    airyra.WithDescription("Add the new feature"),
//	    airyra.WithPriority(airyra.PriorityHigh),
//	)
//
// List tasks with filtering:
//
//	tasks, err := client.ListTasks(ctx,
//	    airyra.WithStatus(airyra.StatusOpen),
//	    airyra.WithPage(1),
//	    airyra.WithPerPage(10),
//	)
//
// Get tasks ready to work on (no unfinished dependencies):
//
//	ready, err := client.ListReadyTasks(ctx)
//
// # Task Lifecycle
//
// Claim a task to work on it:
//
//	task, err := client.ClaimTask(ctx, taskID)
//
// Mark a task as blocked:
//
//	task, err := client.BlockTask(ctx, taskID)
//
// Unblock a task:
//
//	task, err := client.UnblockTask(ctx, taskID)
//
// Complete a task:
//
//	task, err := client.CompleteTask(ctx, taskID)
//
// Release a task (give up ownership):
//
//	task, err := client.ReleaseTask(ctx, taskID, false)
//
// # Dependencies
//
// Add a dependency (child waits for parent):
//
//	err := client.AddDependency(ctx, childTaskID, parentTaskID)
//
// Remove a dependency:
//
//	err := client.RemoveDependency(ctx, childTaskID, parentTaskID)
//
// List dependencies for a task:
//
//	deps, err := client.ListDependencies(ctx, taskID)
//
// # Audit History
//
// Get the change history for a task:
//
//	history, err := client.GetTaskHistory(ctx, taskID)
//
// # Error Handling
//
// The SDK provides typed errors with helper functions:
//
//	task, err := client.ClaimTask(ctx, taskID)
//	if err != nil {
//	    if airyra.IsTaskNotFound(err) {
//	        // Task doesn't exist
//	    } else if airyra.IsAlreadyClaimed(err) {
//	        // Task is claimed by another agent
//	    } else if airyra.IsServerNotRunning(err) {
//	        // Server is not reachable
//	    }
//	}
//
// # Configuration Options
//
// Client options:
//
//	airyra.WithProject(name)        // Required: project name
//	airyra.WithAgentID(id)          // Required: agent ID for ownership
//	airyra.WithHost(host)           // Optional: server host (default: localhost)
//	airyra.WithPort(port)           // Optional: server port (default: 7432)
//	airyra.WithTimeout(duration)    // Optional: HTTP timeout (default: 30s)
//
// CreateTask options:
//
//	airyra.WithDescription(desc)    // Task description
//	airyra.WithPriority(priority)   // Task priority (0-4)
//	airyra.WithParentID(id)         // Parent task ID
//
// UpdateTask options:
//
//	airyra.WithTitle(title)              // New title
//	airyra.WithUpdateDescription(desc)   // New description
//	airyra.WithUpdatePriority(priority)  // New priority
//
// ListTasks options:
//
//	airyra.WithStatus(status)       // Filter by status
//	airyra.WithPage(page)           // Page number (default: 1)
//	airyra.WithPerPage(perPage)     // Items per page (default: 20)
package airyra
