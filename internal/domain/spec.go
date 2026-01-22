package domain

import (
	"time"

	"github.com/airyra/airyra/pkg/idgen"
)

// SpecStatus represents the current state of a spec.
type SpecStatus string

const (
	SpecStatusDraft     SpecStatus = "draft"
	SpecStatusActive    SpecStatus = "active"
	SpecStatusDone      SpecStatus = "done"
	SpecStatusCancelled SpecStatus = "cancelled"
)

// ValidSpecStatuses contains all valid spec status values.
var ValidSpecStatuses = []SpecStatus{SpecStatusDraft, SpecStatusActive, SpecStatusDone, SpecStatusCancelled}

// IsValid checks if the status is a valid spec status.
func (s SpecStatus) IsValid() bool {
	for _, v := range ValidSpecStatuses {
		if s == v {
			return true
		}
	}
	return false
}

// Spec represents an epic-like entity for grouping related tasks.
type Spec struct {
	ID           string     `json:"id"`
	Title        string     `json:"title"`
	Description  *string    `json:"description,omitempty"`
	ManualStatus *string    `json:"manual_status,omitempty"` // Only "cancelled" or nil
	TaskCount    int        `json:"task_count"`
	DoneCount    int        `json:"done_count"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
}

// ComputeStatus calculates the spec status based on task counts and manual status.
func (s *Spec) ComputeStatus() SpecStatus {
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

// Status returns the computed status as a string.
func (s *Spec) Status() SpecStatus {
	return s.ComputeStatus()
}

// NewSpec creates a new spec with the given title and default values.
func NewSpec(title string) *Spec {
	now := time.Now()
	return &Spec{
		ID:        idgen.MustGenerateWithPrefix("sp"),
		Title:     title,
		TaskCount: 0,
		DoneCount: 0,
		CreatedAt: now,
		UpdatedAt: now,
	}
}

// SetDescription sets the spec description.
func (s *Spec) SetDescription(desc string) {
	s.Description = &desc
}

// Cancel marks the spec as cancelled.
func (s *Spec) Cancel() {
	cancelled := "cancelled"
	s.ManualStatus = &cancelled
}

// Reopen removes the cancelled status.
func (s *Spec) Reopen() {
	s.ManualStatus = nil
}

// IsCancelled returns true if the spec is manually cancelled.
func (s *Spec) IsCancelled() bool {
	return s.ManualStatus != nil && *s.ManualStatus == "cancelled"
}
