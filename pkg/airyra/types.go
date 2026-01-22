package airyra

import "time"

// TaskStatus represents the current state of a task.
type TaskStatus string

const (
	// StatusOpen indicates a task is open and not yet claimed.
	StatusOpen TaskStatus = "open"
	// StatusInProgress indicates a task is being worked on.
	StatusInProgress TaskStatus = "in_progress"
	// StatusBlocked indicates a task is blocked and cannot proceed.
	StatusBlocked TaskStatus = "blocked"
	// StatusDone indicates a task is completed.
	StatusDone TaskStatus = "done"
)

// Priority constants for task priority levels.
const (
	PriorityCritical = 0
	PriorityHigh     = 1
	PriorityNormal   = 2
	PriorityLow      = 3
	PriorityLowest   = 4
)

// Task represents a unit of work in the Airyra system.
type Task struct {
	ID          string     `json:"id"`
	ParentID    *string    `json:"parent_id,omitempty"`
	SpecID      *string    `json:"spec_id,omitempty"`
	Title       string     `json:"title"`
	Description *string    `json:"description,omitempty"`
	Status      TaskStatus `json:"status"`
	Priority    int        `json:"priority"`
	ClaimedBy   *string    `json:"claimed_by,omitempty"`
	ClaimedAt   *time.Time `json:"claimed_at,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

// TaskList represents a paginated list of tasks.
type TaskList struct {
	Tasks      []*Task `json:"data"`
	Page       int     `json:"page"`
	PerPage    int     `json:"per_page"`
	Total      int     `json:"total"`
	TotalPages int     `json:"total_pages"`
}

// Dependency represents a dependency relationship between tasks.
// The child task depends on the parent task (child is blocked until parent is done).
type Dependency struct {
	ChildID  string `json:"child_id"`
	ParentID string `json:"parent_id"`
}

// AuditAction represents the type of action recorded in an audit entry.
type AuditAction string

const (
	// ActionCreate indicates a task was created.
	ActionCreate AuditAction = "create"
	// ActionUpdate indicates a task was updated.
	ActionUpdate AuditAction = "update"
	// ActionDelete indicates a task was deleted.
	ActionDelete AuditAction = "delete"
	// ActionClaim indicates a task was claimed.
	ActionClaim AuditAction = "claim"
	// ActionRelease indicates a task was released.
	ActionRelease AuditAction = "release"
)

// AuditEntry represents a single change in the audit log.
type AuditEntry struct {
	ID        int64       `json:"id"`
	TaskID    string      `json:"task_id"`
	Action    AuditAction `json:"action"`
	Field     *string     `json:"field,omitempty"`
	OldValue  *string     `json:"old_value,omitempty"`
	NewValue  *string     `json:"new_value,omitempty"`
	ChangedAt time.Time   `json:"changed_at"`
	ChangedBy string      `json:"changed_by"`
}

// paginatedTaskResponse is the raw JSON structure for paginated task responses.
type paginatedTaskResponse struct {
	Data       []*Task            `json:"data"`
	Pagination paginationResponse `json:"pagination"`
}

// paginationResponse is the raw JSON structure for pagination metadata.
type paginationResponse struct {
	Page       int `json:"page"`
	PerPage    int `json:"per_page"`
	Total      int `json:"total"`
	TotalPages int `json:"total_pages"`
}

// createTaskRequest is the JSON request body for creating a task.
type createTaskRequest struct {
	Title       string  `json:"title"`
	Description *string `json:"description,omitempty"`
	Priority    *int    `json:"priority,omitempty"`
	ParentID    *string `json:"parent_id,omitempty"`
}

// updateTaskRequest is the JSON request body for updating a task.
type updateTaskRequest struct {
	Title       *string `json:"title,omitempty"`
	Description *string `json:"description,omitempty"`
	Priority    *int    `json:"priority,omitempty"`
}

// addDependencyRequest is the JSON request body for adding a dependency.
type addDependencyRequest struct {
	ParentID string `json:"parent_id"`
}

// SpecStatus represents the current state of a spec.
type SpecStatus string

const (
	// SpecStatusDraft indicates a spec has no tasks.
	SpecStatusDraft SpecStatus = "draft"
	// SpecStatusActive indicates a spec has incomplete tasks.
	SpecStatusActive SpecStatus = "active"
	// SpecStatusDone indicates all tasks in the spec are done.
	SpecStatusDone SpecStatus = "done"
	// SpecStatusCancelled indicates the spec is manually cancelled.
	SpecStatusCancelled SpecStatus = "cancelled"
)

// Spec represents an epic-like entity for grouping related tasks.
type Spec struct {
	ID          string     `json:"id"`
	Title       string     `json:"title"`
	Description *string    `json:"description,omitempty"`
	Status      SpecStatus `json:"status"`
	TaskCount   int        `json:"task_count"`
	DoneCount   int        `json:"done_count"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

// SpecList represents a paginated list of specs.
type SpecList struct {
	Specs      []*Spec `json:"data"`
	Page       int     `json:"page"`
	PerPage    int     `json:"per_page"`
	Total      int     `json:"total"`
	TotalPages int     `json:"total_pages"`
}

// SpecDependency represents a dependency relationship between specs.
type SpecDependency struct {
	ChildID  string `json:"child_id"`
	ParentID string `json:"parent_id"`
}

// paginatedSpecResponse is the raw JSON structure for paginated spec responses.
type paginatedSpecResponse struct {
	Data       []*Spec            `json:"data"`
	Pagination paginationResponse `json:"pagination"`
}

// createSpecRequest is the JSON request body for creating a spec.
type createSpecRequest struct {
	Title       string  `json:"title"`
	Description *string `json:"description,omitempty"`
}

// updateSpecRequest is the JSON request body for updating a spec.
type updateSpecRequest struct {
	Title       *string `json:"title,omitempty"`
	Description *string `json:"description,omitempty"`
}

// addSpecDependencyRequest is the JSON request body for adding a spec dependency.
type addSpecDependencyRequest struct {
	ParentID string `json:"parent_id"`
}

