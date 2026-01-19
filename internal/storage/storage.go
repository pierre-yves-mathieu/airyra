// Package storage defines the interfaces for the Airyra storage layer.
package storage

import (
	"context"
	"errors"
	"fmt"
	"time"
)

// Common sentinel errors for storage operations.
var (
	// ErrNotFound is returned when a requested resource does not exist.
	ErrNotFound = errors.New("resource not found")

	// ErrAlreadyClaimed is returned when attempting to claim a task that is
	// already being worked on by another agent.
	ErrAlreadyClaimed = errors.New("task already claimed by another agent")

	// ErrNotOwner is returned when an agent attempts to modify a task they
	// do not own (e.g., releasing or completing a task claimed by another agent).
	ErrNotOwner = errors.New("agent is not the owner of this task")

	// ErrInvalidTransition is returned when a state transition is not allowed
	// (e.g., attempting to mark a blocked task as done).
	ErrInvalidTransition = errors.New("invalid state transition")

	// ErrCycleDetected is returned when adding a dependency would create a
	// circular dependency chain.
	ErrCycleDetected = errors.New("dependency cycle detected")

	// ErrValidationFailed is returned when input validation fails.
	ErrValidationFailed = errors.New("validation failed")
)

// ValidationError provides detailed information about a validation failure.
type ValidationError struct {
	Field   string
	Message string
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("validation failed: %s: %s", e.Field, e.Message)
}

func (e *ValidationError) Is(target error) bool {
	return target == ErrValidationFailed
}

// TransitionError provides context about an invalid state transition.
type TransitionError struct {
	TaskID      string
	FromStatus  string
	ToStatus    string
	Description string
}

func (e *TransitionError) Error() string {
	if e.Description != "" {
		return fmt.Sprintf("invalid transition for task %s from %s to %s: %s",
			e.TaskID, e.FromStatus, e.ToStatus, e.Description)
	}
	return fmt.Sprintf("invalid transition for task %s from %s to %s",
		e.TaskID, e.FromStatus, e.ToStatus)
}

func (e *TransitionError) Is(target error) bool {
	return target == ErrInvalidTransition
}

// CycleError provides details about a detected dependency cycle.
type CycleError struct {
	ChildID  string
	ParentID string
	Path     []string
}

func (e *CycleError) Error() string {
	return fmt.Sprintf("adding dependency from %s to %s would create a cycle: %v",
		e.ChildID, e.ParentID, e.Path)
}

func (e *CycleError) Is(target error) bool {
	return target == ErrCycleDetected
}

// Note: Task, Dependency, and AuditEntry structs are defined in models.go

// ListOptions specifies filtering and pagination options for listing tasks.
type ListOptions struct {
	Status   *string
	ParentID *string
	Priority *int
	Page     int
	PerPage  int
}

// Normalize ensures ListOptions has valid default values.
func (o *ListOptions) Normalize() {
	if o.Page < 1 {
		o.Page = 1
	}
	if o.PerPage < 1 {
		o.PerPage = 50
	}
	if o.PerPage > 100 {
		o.PerPage = 100
	}
}

// AuditQueryOptions specifies filtering options for querying audit logs.
type AuditQueryOptions struct {
	Action    *string
	ChangedBy *string
	Since     *time.Time
	Until     *time.Time
}

// Store is the main interface for accessing all repositories.
// It provides access to task, dependency, and audit repositories,
// and supports transactional operations.
type Store interface {
	// Tasks returns the TaskRepository for task operations.
	Tasks() TaskRepository

	// Dependencies returns the DependencyRepository for dependency operations.
	Dependencies() DependencyRepository

	// AuditLogs returns the AuditRepository for audit log operations.
	AuditLogs() AuditRepository

	// WithTx executes the given function within a database transaction.
	// If fn returns an error, the transaction is rolled back.
	// If fn returns nil, the transaction is committed.
	WithTx(ctx context.Context, fn func(TxStore) error) error

	// Close releases any resources held by the store.
	Close() error
}

// TxStore provides access to repositories within a transaction context.
// It is passed to the function executed by Store.WithTx.
type TxStore interface {
	// Tasks returns the TaskRepository for task operations within the transaction.
	Tasks() TaskRepository

	// Dependencies returns the DependencyRepository for dependency operations within the transaction.
	Dependencies() DependencyRepository

	// AuditLogs returns the AuditRepository for audit log operations within the transaction.
	AuditLogs() AuditRepository
}

// TaskRepository defines operations for managing tasks.
type TaskRepository interface {
	// Create inserts a new task into the store.
	Create(ctx context.Context, task *Task) error

	// Get retrieves a task by its ID.
	// Returns ErrNotFound if the task does not exist.
	Get(ctx context.Context, id string) (*Task, error)

	// List returns tasks matching the specified options.
	// Returns the matching tasks, total count of matching tasks, and any error.
	List(ctx context.Context, opts ListOptions) ([]*Task, int, error)

	// ListReady returns tasks that have no unmet dependencies and are ready to be worked on.
	// A task is ready if all its parent dependencies are in the "done" status.
	// Returns the matching tasks, total count of matching tasks, and any error.
	ListReady(ctx context.Context, opts ListOptions) ([]*Task, int, error)

	// Update modifies an existing task.
	// Returns ErrNotFound if the task does not exist.
	Update(ctx context.Context, task *Task) error

	// Delete removes a task by its ID.
	// Returns ErrNotFound if the task does not exist.
	Delete(ctx context.Context, id string) error

	// Claim atomically transitions a task from "open" to "in_progress" and assigns it to an agent.
	// Returns ErrNotFound if the task does not exist.
	// Returns ErrAlreadyClaimed if the task is already claimed.
	// Returns ErrInvalidTransition if the task is not in "open" status.
	Claim(ctx context.Context, id, agentID string) error

	// Release transitions a task from "in_progress" back to "open" and removes the agent assignment.
	// Returns ErrNotFound if the task does not exist.
	// Returns ErrNotOwner if the task is not claimed by the specified agent.
	// Returns ErrInvalidTransition if the task is not in "in_progress" status.
	Release(ctx context.Context, id, agentID string) error

	// MarkDone transitions a task from "in_progress" to "done".
	// Returns ErrNotFound if the task does not exist.
	// Returns ErrNotOwner if the task is not claimed by the specified agent.
	// Returns ErrInvalidTransition if the task is not in "in_progress" status.
	MarkDone(ctx context.Context, id, agentID string) error

	// Block transitions a task to "blocked" status from any state.
	// Returns ErrNotFound if the task does not exist.
	Block(ctx context.Context, id string) error

	// Unblock transitions a task from "blocked" back to "open" status.
	// Returns ErrNotFound if the task does not exist.
	// Returns ErrInvalidTransition if the task is not in "blocked" status.
	Unblock(ctx context.Context, id string) error
}

// DependencyRepository defines operations for managing task dependencies.
type DependencyRepository interface {
	// Add creates a dependency relationship where childID depends on parentID.
	// The child task cannot be marked as ready until the parent task is done.
	// Returns ErrCycleDetected if adding this dependency would create a cycle.
	Add(ctx context.Context, childID, parentID string) error

	// Remove deletes a dependency relationship.
	Remove(ctx context.Context, childID, parentID string) error

	// ListForTask returns all dependencies for a given task.
	// This includes both dependencies where the task is the child (things it depends on)
	// and where it is the parent (things that depend on it).
	ListForTask(ctx context.Context, taskID string) ([]Dependency, error)

	// CheckCycle determines if adding a dependency from childID to parentID would create a cycle.
	// Returns hasCycle=true and the path if a cycle would be created.
	CheckCycle(ctx context.Context, childID, parentID string) (hasCycle bool, path []string, err error)
}

// AuditRepository defines operations for audit logging.
type AuditRepository interface {
	// Log records an audit entry.
	Log(ctx context.Context, entry *AuditEntry) error

	// ListForTask returns all audit entries for a specific task, ordered by timestamp descending.
	ListForTask(ctx context.Context, taskID string) ([]*AuditEntry, error)

	// Query returns audit entries matching the specified options, ordered by timestamp descending.
	Query(ctx context.Context, opts AuditQueryOptions) ([]*AuditEntry, error)
}
