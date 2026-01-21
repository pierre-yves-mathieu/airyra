package main

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/airyra/airyra/internal/client"
	"github.com/airyra/airyra/internal/domain"
)

func TestPrintTask_TableFormat(t *testing.T) {
	var buf bytes.Buffer
	task := &domain.Task{
		ID:        "abc123",
		Title:     "Test Task",
		Status:    domain.StatusOpen,
		Priority:  2,
		CreatedAt: time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC),
		UpdatedAt: time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC),
	}

	printTask(&buf, task, false)

	output := buf.String()
	if !strings.Contains(output, "abc123") {
		t.Error("Output should contain task ID")
	}
	if !strings.Contains(output, "Test Task") {
		t.Error("Output should contain task title")
	}
	if !strings.Contains(output, "open") {
		t.Error("Output should contain task status")
	}
}

func TestPrintTask_JSONFormat(t *testing.T) {
	var buf bytes.Buffer
	task := &domain.Task{
		ID:        "abc123",
		Title:     "Test Task",
		Status:    domain.StatusOpen,
		Priority:  2,
		CreatedAt: time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC),
		UpdatedAt: time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC),
	}

	printTask(&buf, task, true)

	output := buf.String()

	// Should be valid JSON
	var parsed domain.Task
	if err := json.Unmarshal([]byte(output), &parsed); err != nil {
		t.Errorf("Output should be valid JSON: %v", err)
	}
	if parsed.ID != "abc123" {
		t.Errorf("Parsed ID = %s, expected abc123", parsed.ID)
	}
}

func TestPrintTask_WithDescription(t *testing.T) {
	var buf bytes.Buffer
	desc := "This is a description"
	task := &domain.Task{
		ID:          "abc123",
		Title:       "Test Task",
		Description: &desc,
		Status:      domain.StatusOpen,
		Priority:    2,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	printTask(&buf, task, false)

	output := buf.String()
	if !strings.Contains(output, "This is a description") {
		t.Error("Output should contain description")
	}
}

func TestPrintTask_WithClaimedBy(t *testing.T) {
	var buf bytes.Buffer
	claimedBy := "user@host:/path"
	claimedAt := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)
	task := &domain.Task{
		ID:        "abc123",
		Title:     "Test Task",
		Status:    domain.StatusInProgress,
		Priority:  2,
		ClaimedBy: &claimedBy,
		ClaimedAt: &claimedAt,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	printTask(&buf, task, false)

	output := buf.String()
	if !strings.Contains(output, "user@host:/path") {
		t.Error("Output should contain claimed_by")
	}
}

func TestPrintTaskList_TableFormat(t *testing.T) {
	var buf bytes.Buffer
	tasks := []*domain.Task{
		{
			ID:        "abc123",
			Title:     "Task 1",
			Status:    domain.StatusOpen,
			Priority:  1,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		{
			ID:        "def456",
			Title:     "Task 2",
			Status:    domain.StatusInProgress,
			Priority:  2,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
	}
	pagination := &client.Pagination{
		Page:       1,
		PerPage:    50,
		Total:      2,
		TotalPages: 1,
	}

	printTaskList(&buf, tasks, pagination, false)

	output := buf.String()
	if !strings.Contains(output, "abc123") {
		t.Error("Output should contain first task ID")
	}
	if !strings.Contains(output, "def456") {
		t.Error("Output should contain second task ID")
	}
	if !strings.Contains(output, "Task 1") {
		t.Error("Output should contain first task title")
	}
	if !strings.Contains(output, "Task 2") {
		t.Error("Output should contain second task title")
	}
}

func TestPrintTaskList_JSONFormat(t *testing.T) {
	var buf bytes.Buffer
	tasks := []*domain.Task{
		{
			ID:        "abc123",
			Title:     "Task 1",
			Status:    domain.StatusOpen,
			Priority:  1,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
	}
	pagination := &client.Pagination{
		Page:       1,
		PerPage:    50,
		Total:      1,
		TotalPages: 1,
	}

	printTaskList(&buf, tasks, pagination, true)

	output := buf.String()

	// Should be valid JSON
	var parsed struct {
		Data       []domain.Task `json:"data"`
		Pagination struct {
			Page       int `json:"page"`
			PerPage    int `json:"per_page"`
			Total      int `json:"total"`
			TotalPages int `json:"total_pages"`
		} `json:"pagination"`
	}
	if err := json.Unmarshal([]byte(output), &parsed); err != nil {
		t.Errorf("Output should be valid JSON: %v", err)
	}
	if len(parsed.Data) != 1 {
		t.Errorf("Parsed data length = %d, expected 1", len(parsed.Data))
	}
}

func TestPrintTaskList_EmptyList(t *testing.T) {
	var buf bytes.Buffer
	tasks := []*domain.Task{}
	pagination := &client.Pagination{
		Page:       1,
		PerPage:    50,
		Total:      0,
		TotalPages: 0,
	}

	printTaskList(&buf, tasks, pagination, false)

	output := buf.String()
	if !strings.Contains(output, "No tasks found") {
		t.Error("Output should indicate no tasks found")
	}
}

func TestPrintDependencies_TableFormat(t *testing.T) {
	var buf bytes.Buffer
	deps := []domain.Dependency{
		{ChildID: "abc123", ParentID: "def456"},
		{ChildID: "abc123", ParentID: "ghi789"},
	}

	printDependencies(&buf, "abc123", deps, false)

	output := buf.String()
	if !strings.Contains(output, "def456") {
		t.Error("Output should contain first parent ID")
	}
	if !strings.Contains(output, "ghi789") {
		t.Error("Output should contain second parent ID")
	}
}

func TestPrintDependencies_JSONFormat(t *testing.T) {
	var buf bytes.Buffer
	deps := []domain.Dependency{
		{ChildID: "abc123", ParentID: "def456"},
	}

	printDependencies(&buf, "abc123", deps, true)

	output := buf.String()

	// Should be valid JSON
	var parsed []domain.Dependency
	if err := json.Unmarshal([]byte(output), &parsed); err != nil {
		t.Errorf("Output should be valid JSON: %v", err)
	}
}

func TestPrintHistory_TableFormat(t *testing.T) {
	var buf bytes.Buffer
	entries := []domain.AuditEntry{
		{
			ID:        1,
			TaskID:    "abc123",
			Action:    domain.ActionCreate,
			ChangedAt: time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC),
			ChangedBy: "user@host:/path",
		},
	}

	printHistory(&buf, entries, false)

	output := buf.String()
	if !strings.Contains(output, "create") {
		t.Error("Output should contain action")
	}
	if !strings.Contains(output, "user@host:/path") {
		t.Error("Output should contain changed_by")
	}
}

func TestPrintHistory_JSONFormat(t *testing.T) {
	var buf bytes.Buffer
	entries := []domain.AuditEntry{
		{
			ID:        1,
			TaskID:    "abc123",
			Action:    domain.ActionCreate,
			ChangedAt: time.Now(),
			ChangedBy: "user@host:/path",
		},
	}

	printHistory(&buf, entries, true)

	output := buf.String()

	// Should be valid JSON
	var parsed []domain.AuditEntry
	if err := json.Unmarshal([]byte(output), &parsed); err != nil {
		t.Errorf("Output should be valid JSON: %v", err)
	}
}

func TestPrintError(t *testing.T) {
	var buf bytes.Buffer
	err := domain.NewTaskNotFoundError("abc123")

	printError(&buf, err, false)

	output := buf.String()
	if !strings.Contains(output, "Error:") {
		t.Error("Output should contain 'Error:'")
	}
}

func TestPrintError_JSON(t *testing.T) {
	var buf bytes.Buffer
	err := domain.NewTaskNotFoundError("abc123")

	printError(&buf, err, true)

	output := buf.String()

	// Should be valid JSON
	var parsed struct {
		Error struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	if jsonErr := json.Unmarshal([]byte(output), &parsed); jsonErr != nil {
		t.Errorf("Output should be valid JSON: %v", jsonErr)
	}
}

func TestPriorityString(t *testing.T) {
	tests := []struct {
		priority int
		expected string
	}{
		{0, "critical"},
		{1, "high"},
		{2, "normal"},
		{3, "low"},
		{4, "lowest"},
		{99, "99"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			result := priorityString(tt.priority)
			if result != tt.expected {
				t.Errorf("priorityString(%d) = %s, expected %s", tt.priority, result, tt.expected)
			}
		})
	}
}
