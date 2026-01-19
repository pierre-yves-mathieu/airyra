package storage

import (
	"errors"
	"strings"
	"time"
)

// Task represents a task in the Airyra system.
type Task struct {
	ID          string     // ar-xxxx format
	ParentID    *string    // optional, for hierarchy
	Title       string     // required
	Description *string    // optional
	Status      string     // open/in_progress/blocked/done
	Priority    int        // 0-4, default 2
	ClaimedBy   *string    // agent ID when in_progress
	ClaimedAt   *time.Time // when the task was claimed
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// Dependency represents a dependency between tasks.
// ChildID is blocked by ParentID.
type Dependency struct {
	ChildID  string // the blocked task
	ParentID string // the blocking task
}

// AuditEntry represents an audit log entry for task changes.
type AuditEntry struct {
	ID        int64     // auto-increment
	TaskID    string    // the task that was changed
	Action    string    // create, update, delete, claim, release
	Field     *string   // which field changed (optional)
	OldValue  *string   // JSON representation of old value (optional)
	NewValue  *string   // JSON representation of new value (optional)
	ChangedAt time.Time // when the change occurred
	ChangedBy string    // who made the change (agent ID or system)
}

// Status constants for Task
const (
	StatusOpen       = "open"
	StatusInProgress = "in_progress"
	StatusBlocked    = "blocked"
	StatusDone       = "done"
)

// Priority constants
const (
	PriorityLowest  = 0
	PriorityLow     = 1
	PriorityMedium  = 2
	PriorityHigh    = 3
	PriorityHighest = 4

	PriorityMin = 0
	PriorityMax = 4
)

// Action constants for AuditEntry
const (
	ActionCreate  = "create"
	ActionUpdate  = "update"
	ActionDelete  = "delete"
	ActionClaim   = "claim"
	ActionRelease = "release"
)

// ValidStatuses contains all valid task statuses
var ValidStatuses = []string{
	StatusOpen,
	StatusInProgress,
	StatusBlocked,
	StatusDone,
}

// ValidActions contains all valid audit actions
var ValidActions = []string{
	ActionCreate,
	ActionUpdate,
	ActionDelete,
	ActionClaim,
	ActionRelease,
}

// Validate checks if the Task has valid field values
func (t *Task) Validate() error {
	// Title is required
	if strings.TrimSpace(t.Title) == "" {
		return errors.New("title is required")
	}

	// Priority must be in range 0-4
	if t.Priority < PriorityMin || t.Priority > PriorityMax {
		return errors.New("priority must be between 0 and 4")
	}

	// Status must be valid
	if !isValidStatus(t.Status) {
		return errors.New("invalid status")
	}

	return nil
}

// isValidStatus checks if the given status is a valid task status
func isValidStatus(status string) bool {
	for _, s := range ValidStatuses {
		if s == status {
			return true
		}
	}
	return false
}

// IsValidAction checks if the given action is a valid audit action
func IsValidAction(action string) bool {
	for _, a := range ValidActions {
		if a == action {
			return true
		}
	}
	return false
}

// StringPtr is a helper function to create a pointer to a string
func StringPtr(s string) *string {
	return &s
}

// TimePtr is a helper function to create a pointer to a time.Time
func TimePtr(t time.Time) *time.Time {
	return &t
}
