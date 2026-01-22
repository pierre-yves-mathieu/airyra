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
	SpecID      *string    // optional, for spec grouping
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

// Spec represents an epic-like entity for grouping related tasks.
type Spec struct {
	ID           string    // sp-xxxx format
	Title        string    // required
	Description  *string   // optional
	ManualStatus *string   // only "cancelled" or nil
	TaskCount    int       // computed: count of tasks in this spec
	DoneCount    int       // computed: count of done tasks in this spec
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// SpecDependency represents a dependency between specs.
// ChildID is blocked by ParentID.
type SpecDependency struct {
	ChildID  string // the blocked spec
	ParentID string // the blocking spec
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

// SpecStatus constants
const (
	SpecStatusDraft     = "draft"
	SpecStatusActive    = "active"
	SpecStatusDone      = "done"
	SpecStatusCancelled = "cancelled"
)

// ValidSpecStatuses contains all valid spec statuses
var ValidSpecStatuses = []string{
	SpecStatusDraft,
	SpecStatusActive,
	SpecStatusDone,
	SpecStatusCancelled,
}

// ComputeStatus calculates the spec status based on task counts and manual status.
func (s *Spec) ComputeStatus() string {
	if s.ManualStatus != nil && *s.ManualStatus == "cancelled" {
		return SpecStatusCancelled
	}
	if s.TaskCount == 0 {
		return SpecStatusDraft
	}
	if s.DoneCount == s.TaskCount {
		return SpecStatusDone
	}
	return SpecStatusActive
}

// IsCancelled returns true if the spec is manually cancelled.
func (s *Spec) IsCancelled() bool {
	return s.ManualStatus != nil && *s.ManualStatus == "cancelled"
}

// IsValidSpecStatus checks if the given status is a valid spec status
func IsValidSpecStatus(status string) bool {
	for _, s := range ValidSpecStatuses {
		if s == status {
			return true
		}
	}
	return false
}

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
