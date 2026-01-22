package domain

import (
	"time"

	"github.com/airyra/airyra/pkg/idgen"
)

// TaskStatus represents the current state of a task.
type TaskStatus string

const (
	StatusOpen       TaskStatus = "open"
	StatusInProgress TaskStatus = "in_progress"
	StatusBlocked    TaskStatus = "blocked"
	StatusDone       TaskStatus = "done"
)

// Priority constants for convenience.
const (
	PriorityCritical = 0
	PriorityHigh     = 1
	PriorityNormal   = 2 // Default priority
	PriorityLow      = 3
	PriorityLowest   = 4
)

// ValidStatuses contains all valid task status values.
var ValidStatuses = []TaskStatus{StatusOpen, StatusInProgress, StatusBlocked, StatusDone}

// IsValid checks if the status is a valid task status.
func (s TaskStatus) IsValid() bool {
	for _, v := range ValidStatuses {
		if s == v {
			return true
		}
	}
	return false
}

// Task represents a unit of work in the system.
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

// ValidPriority checks if the priority value is within valid range (0-4).
func ValidPriority(p int) bool {
	return p >= 0 && p <= 4
}

// NewTask creates a new task with the given title and default values.
// Default status is StatusOpen, default priority is 2 (normal).
func NewTask(title string) *Task {
	now := time.Now()
	return &Task{
		ID:        idgen.MustGenerate(),
		Title:     title,
		Status:    StatusOpen,
		Priority:  PriorityNormal,
		CreatedAt: now,
		UpdatedAt: now,
	}
}

// NewTaskWithPriority creates a new task with the given title and priority.
func NewTaskWithPriority(title string, priority int) *Task {
	task := NewTask(title)
	task.Priority = priority
	return task
}

// SetDescription sets the task description.
func (t *Task) SetDescription(desc string) {
	t.Description = &desc
}

// SetParentID sets the task's parent ID.
func (t *Task) SetParentID(parentID string) {
	t.ParentID = &parentID
}

// SetSpecID sets the task's spec ID.
func (t *Task) SetSpecID(specID string) {
	t.SpecID = &specID
}
