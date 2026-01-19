package domain

import (
	"testing"
	"time"
)

func TestTaskStatus_IsValid(t *testing.T) {
	tests := []struct {
		name   string
		status TaskStatus
		want   bool
	}{
		{"StatusOpen is valid", StatusOpen, true},
		{"StatusInProgress is valid", StatusInProgress, true},
		{"StatusBlocked is valid", StatusBlocked, true},
		{"StatusDone is valid", StatusDone, true},
		{"empty string is invalid", TaskStatus(""), false},
		{"random string is invalid", TaskStatus("random"), false},
		{"similar but wrong is invalid", TaskStatus("Open"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.status.IsValid(); got != tt.want {
				t.Errorf("TaskStatus.IsValid() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestValidStatuses_ContainsAllStatuses(t *testing.T) {
	expected := []TaskStatus{StatusOpen, StatusInProgress, StatusBlocked, StatusDone}
	if len(ValidStatuses) != len(expected) {
		t.Errorf("ValidStatuses has %d items, want %d", len(ValidStatuses), len(expected))
	}
	for _, s := range expected {
		found := false
		for _, v := range ValidStatuses {
			if v == s {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("ValidStatuses does not contain %s", s)
		}
	}
}

func TestValidPriority(t *testing.T) {
	tests := []struct {
		name     string
		priority int
		want     bool
	}{
		{"priority 0 (critical) is valid", 0, true},
		{"priority 1 is valid", 1, true},
		{"priority 2 (default) is valid", 2, true},
		{"priority 3 is valid", 3, true},
		{"priority 4 (low) is valid", 4, true},
		{"priority -1 is invalid", -1, false},
		{"priority 5 is invalid", 5, false},
		{"priority 100 is invalid", 100, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ValidPriority(tt.priority); got != tt.want {
				t.Errorf("ValidPriority(%d) = %v, want %v", tt.priority, got, tt.want)
			}
		})
	}
}

func TestNewTask(t *testing.T) {
	title := "Test Task"
	task := NewTask(title)

	// Check ID is not empty
	if task.ID == "" {
		t.Error("NewTask() should generate a non-empty ID")
	}

	// Check title is set
	if task.Title != title {
		t.Errorf("NewTask() Title = %v, want %v", task.Title, title)
	}

	// Check default status is open
	if task.Status != StatusOpen {
		t.Errorf("NewTask() Status = %v, want %v", task.Status, StatusOpen)
	}

	// Check default priority is 2
	if task.Priority != 2 {
		t.Errorf("NewTask() Priority = %v, want %v", task.Priority, 2)
	}

	// Check ParentID is nil
	if task.ParentID != nil {
		t.Error("NewTask() ParentID should be nil")
	}

	// Check Description is nil
	if task.Description != nil {
		t.Error("NewTask() Description should be nil")
	}

	// Check ClaimedBy is nil
	if task.ClaimedBy != nil {
		t.Error("NewTask() ClaimedBy should be nil")
	}

	// Check ClaimedAt is nil
	if task.ClaimedAt != nil {
		t.Error("NewTask() ClaimedAt should be nil")
	}

	// Check timestamps are set and reasonable
	now := time.Now()
	if task.CreatedAt.IsZero() {
		t.Error("NewTask() CreatedAt should not be zero")
	}
	if task.UpdatedAt.IsZero() {
		t.Error("NewTask() UpdatedAt should not be zero")
	}
	// Timestamps should be within 1 second of now
	if now.Sub(task.CreatedAt) > time.Second {
		t.Error("NewTask() CreatedAt should be recent")
	}
	if now.Sub(task.UpdatedAt) > time.Second {
		t.Error("NewTask() UpdatedAt should be recent")
	}
}

func TestNewTaskWithPriority(t *testing.T) {
	title := "Test Task"
	priority := 1
	task := NewTaskWithPriority(title, priority)

	if task.Title != title {
		t.Errorf("NewTaskWithPriority() Title = %v, want %v", task.Title, title)
	}
	if task.Priority != priority {
		t.Errorf("NewTaskWithPriority() Priority = %v, want %v", task.Priority, priority)
	}
}

func TestTask_SetDescription(t *testing.T) {
	task := NewTask("Test")
	desc := "A description"
	task.SetDescription(desc)

	if task.Description == nil {
		t.Fatal("SetDescription() should set Description")
	}
	if *task.Description != desc {
		t.Errorf("SetDescription() Description = %v, want %v", *task.Description, desc)
	}
}

func TestTask_SetParentID(t *testing.T) {
	task := NewTask("Test")
	parentID := "ar-1234"
	task.SetParentID(parentID)

	if task.ParentID == nil {
		t.Fatal("SetParentID() should set ParentID")
	}
	if *task.ParentID != parentID {
		t.Errorf("SetParentID() ParentID = %v, want %v", *task.ParentID, parentID)
	}
}
