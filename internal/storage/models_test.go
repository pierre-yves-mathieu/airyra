package storage

import (
	"encoding/json"
	"testing"
	"time"
)

func TestTask_Validation(t *testing.T) {
	t.Run("valid task passes validation", func(t *testing.T) {
		task := Task{
			ID:        "ar-0001",
			Title:     "Test Task",
			Status:    StatusOpen,
			Priority:  PriorityMedium,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}
		if err := task.Validate(); err != nil {
			t.Errorf("expected valid task to pass validation, got error: %v", err)
		}
	})

	t.Run("title is required", func(t *testing.T) {
		task := Task{
			ID:        "ar-0001",
			Title:     "",
			Status:    StatusOpen,
			Priority:  PriorityMedium,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}
		err := task.Validate()
		if err == nil {
			t.Error("expected validation error for empty title, got nil")
		}
		if err != nil && err.Error() != "title is required" {
			t.Errorf("expected error 'title is required', got: %v", err)
		}
	})

	t.Run("title with only whitespace is invalid", func(t *testing.T) {
		task := Task{
			ID:        "ar-0001",
			Title:     "   ",
			Status:    StatusOpen,
			Priority:  PriorityMedium,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}
		err := task.Validate()
		if err == nil {
			t.Error("expected validation error for whitespace-only title, got nil")
		}
	})

	t.Run("priority must be in range 0-4", func(t *testing.T) {
		testCases := []struct {
			name     string
			priority int
			valid    bool
		}{
			{"priority -1 is invalid", -1, false},
			{"priority 0 is valid", 0, true},
			{"priority 1 is valid", 1, true},
			{"priority 2 is valid", 2, true},
			{"priority 3 is valid", 3, true},
			{"priority 4 is valid", 4, true},
			{"priority 5 is invalid", 5, false},
			{"priority 100 is invalid", 100, false},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				task := Task{
					ID:        "ar-0001",
					Title:     "Test Task",
					Status:    StatusOpen,
					Priority:  tc.priority,
					CreatedAt: time.Now(),
					UpdatedAt: time.Now(),
				}
				err := task.Validate()
				if tc.valid && err != nil {
					t.Errorf("expected priority %d to be valid, got error: %v", tc.priority, err)
				}
				if !tc.valid && err == nil {
					t.Errorf("expected priority %d to be invalid, got nil error", tc.priority)
				}
			})
		}
	})

	t.Run("invalid status fails validation", func(t *testing.T) {
		task := Task{
			ID:        "ar-0001",
			Title:     "Test Task",
			Status:    "invalid_status",
			Priority:  PriorityMedium,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}
		err := task.Validate()
		if err == nil {
			t.Error("expected validation error for invalid status, got nil")
		}
	})

	t.Run("all valid statuses pass validation", func(t *testing.T) {
		statuses := []string{StatusOpen, StatusInProgress, StatusBlocked, StatusDone}
		for _, status := range statuses {
			t.Run(status, func(t *testing.T) {
				task := Task{
					ID:        "ar-0001",
					Title:     "Test Task",
					Status:    status,
					Priority:  PriorityMedium,
					CreatedAt: time.Now(),
					UpdatedAt: time.Now(),
				}
				if err := task.Validate(); err != nil {
					t.Errorf("expected status %q to be valid, got error: %v", status, err)
				}
			})
		}
	})
}

func TestAuditEntry_JSONValues(t *testing.T) {
	t.Run("old_value serializes correctly", func(t *testing.T) {
		oldValue := map[string]interface{}{
			"priority": 2,
			"status":   "open",
		}
		jsonBytes, err := json.Marshal(oldValue)
		if err != nil {
			t.Fatalf("failed to marshal old_value: %v", err)
		}
		jsonStr := string(jsonBytes)

		entry := AuditEntry{
			ID:        1,
			TaskID:    "ar-0001",
			Action:    ActionUpdate,
			Field:     stringPtr("priority"),
			OldValue:  &jsonStr,
			NewValue:  nil,
			ChangedAt: time.Now(),
			ChangedBy: "agent-001",
		}

		// Verify we can unmarshal the stored value
		if entry.OldValue != nil {
			var decoded map[string]interface{}
			if err := json.Unmarshal([]byte(*entry.OldValue), &decoded); err != nil {
				t.Errorf("failed to unmarshal old_value: %v", err)
			}
			if decoded["priority"].(float64) != 2 {
				t.Errorf("expected priority 2, got %v", decoded["priority"])
			}
			if decoded["status"].(string) != "open" {
				t.Errorf("expected status 'open', got %v", decoded["status"])
			}
		}
	})

	t.Run("new_value serializes correctly", func(t *testing.T) {
		newValue := map[string]interface{}{
			"priority": 4,
			"status":   "in_progress",
		}
		jsonBytes, err := json.Marshal(newValue)
		if err != nil {
			t.Fatalf("failed to marshal new_value: %v", err)
		}
		jsonStr := string(jsonBytes)

		entry := AuditEntry{
			ID:        2,
			TaskID:    "ar-0001",
			Action:    ActionUpdate,
			Field:     stringPtr("priority"),
			OldValue:  nil,
			NewValue:  &jsonStr,
			ChangedAt: time.Now(),
			ChangedBy: "agent-002",
		}

		// Verify we can unmarshal the stored value
		if entry.NewValue != nil {
			var decoded map[string]interface{}
			if err := json.Unmarshal([]byte(*entry.NewValue), &decoded); err != nil {
				t.Errorf("failed to unmarshal new_value: %v", err)
			}
			if decoded["priority"].(float64) != 4 {
				t.Errorf("expected priority 4, got %v", decoded["priority"])
			}
			if decoded["status"].(string) != "in_progress" {
				t.Errorf("expected status 'in_progress', got %v", decoded["status"])
			}
		}
	})

	t.Run("both old_value and new_value can be set", func(t *testing.T) {
		oldJSON := `{"title":"Old Title"}`
		newJSON := `{"title":"New Title"}`

		entry := AuditEntry{
			ID:        3,
			TaskID:    "ar-0002",
			Action:    ActionUpdate,
			Field:     stringPtr("title"),
			OldValue:  &oldJSON,
			NewValue:  &newJSON,
			ChangedAt: time.Now(),
			ChangedBy: "agent-003",
		}

		var oldDecoded, newDecoded map[string]interface{}
		if err := json.Unmarshal([]byte(*entry.OldValue), &oldDecoded); err != nil {
			t.Errorf("failed to unmarshal old_value: %v", err)
		}
		if err := json.Unmarshal([]byte(*entry.NewValue), &newDecoded); err != nil {
			t.Errorf("failed to unmarshal new_value: %v", err)
		}

		if oldDecoded["title"].(string) != "Old Title" {
			t.Errorf("expected old title 'Old Title', got %v", oldDecoded["title"])
		}
		if newDecoded["title"].(string) != "New Title" {
			t.Errorf("expected new title 'New Title', got %v", newDecoded["title"])
		}
	})

	t.Run("nil values are handled correctly", func(t *testing.T) {
		entry := AuditEntry{
			ID:        4,
			TaskID:    "ar-0003",
			Action:    ActionCreate,
			Field:     nil,
			OldValue:  nil,
			NewValue:  nil,
			ChangedAt: time.Now(),
			ChangedBy: "agent-004",
		}

		if entry.OldValue != nil {
			t.Error("expected OldValue to be nil")
		}
		if entry.NewValue != nil {
			t.Error("expected NewValue to be nil")
		}
		if entry.Field != nil {
			t.Error("expected Field to be nil")
		}
	})
}

// Helper function to create string pointers
func stringPtr(s string) *string {
	return &s
}
