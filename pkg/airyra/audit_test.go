package airyra

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestGetTaskHistory(t *testing.T) {
	now := time.Now()
	field := "status"
	oldValue := "open"
	newValue := "in_progress"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/projects/test-project/tasks/task-123/history" {
			t.Errorf("expected path /v1/projects/test-project/tasks/task-123/history, got %s", r.URL.Path)
		}
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]AuditEntry{
			{
				ID:        1,
				TaskID:    "task-123",
				Action:    ActionCreate,
				ChangedAt: now,
				ChangedBy: "agent-1",
			},
			{
				ID:        2,
				TaskID:    "task-123",
				Action:    ActionUpdate,
				Field:     &field,
				OldValue:  &oldValue,
				NewValue:  &newValue,
				ChangedAt: now,
				ChangedBy: "agent-1",
			},
		})
	}))
	defer server.Close()

	client := newTestClient(t, server)
	history, err := client.GetTaskHistory(context.Background(), "task-123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(history) != 2 {
		t.Errorf("expected 2 entries, got %d", len(history))
	}

	if history[0].Action != ActionCreate {
		t.Errorf("expected first entry action to be %s, got %s", ActionCreate, history[0].Action)
	}
	if history[0].ChangedBy != "agent-1" {
		t.Errorf("expected first entry changed_by agent-1, got %s", history[0].ChangedBy)
	}

	if history[1].Action != ActionUpdate {
		t.Errorf("expected second entry action to be %s, got %s", ActionUpdate, history[1].Action)
	}
	if history[1].Field == nil || *history[1].Field != "status" {
		t.Errorf("expected second entry field to be 'status', got %v", history[1].Field)
	}
	if history[1].OldValue == nil || *history[1].OldValue != "open" {
		t.Errorf("expected second entry old_value to be 'open', got %v", history[1].OldValue)
	}
	if history[1].NewValue == nil || *history[1].NewValue != "in_progress" {
		t.Errorf("expected second entry new_value to be 'in_progress', got %v", history[1].NewValue)
	}
}

func TestGetTaskHistoryTaskNotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(apiErrorResponse{
			Error: apiError{
				Code:    string(ErrCodeTaskNotFound),
				Message: "Task not found",
				Context: map[string]interface{}{"id": "task-123"},
			},
		})
	}))
	defer server.Close()

	client := newTestClient(t, server)
	_, err := client.GetTaskHistory(context.Background(), "task-123")
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if !IsTaskNotFound(err) {
		t.Errorf("expected IsTaskNotFound to be true, got false")
	}
}
