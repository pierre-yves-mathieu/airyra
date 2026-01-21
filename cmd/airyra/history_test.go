package main

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/airyra/airyra/internal/client"
	"github.com/airyra/airyra/internal/domain"
)

func TestHistoryCmd_Exists(t *testing.T) {
	if historyCmd == nil {
		t.Error("historyCmd should not be nil")
	}
}

func TestHistoryCmd_Use(t *testing.T) {
	if historyCmd.Use != "history <id>" {
		t.Errorf("historyCmd.Use = %s, expected 'history <id>'", historyCmd.Use)
	}
}

func TestLogCmd_Exists(t *testing.T) {
	if logCmd == nil {
		t.Error("logCmd should not be nil")
	}
}

func TestLogCmd_Use(t *testing.T) {
	if logCmd.Use != "log" {
		t.Errorf("logCmd.Use = %s, expected 'log'", logCmd.Use)
	}
}

func TestHistory_Success(t *testing.T) {
	field := "status"
	oldVal := "open"
	newVal := "in_progress"

	server := newMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v1/projects/testproject/tasks/abc123/history" && r.Method == "GET" {
			entries := []domain.AuditEntry{
				{
					ID:        1,
					TaskID:    "abc123",
					Action:    domain.ActionCreate,
					ChangedAt: time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC),
					ChangedBy: "user@host:/path",
				},
				{
					ID:        2,
					TaskID:    "abc123",
					Action:    domain.ActionUpdate,
					Field:     &field,
					OldValue:  &oldVal,
					NewValue:  &newVal,
					ChangedAt: time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC),
					ChangedBy: "user@host:/path",
				},
			}
			json.NewEncoder(w).Encode(entries)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	})
	defer server.Close()

	host, port := parseURL(server.URL)
	c := client.NewClient(host, port, "testproject", "test@host:/path")

	entries, err := c.GetTaskHistory(context.Background(), "abc123")
	if err != nil {
		t.Fatalf("GetTaskHistory failed: %v", err)
	}

	if len(entries) != 2 {
		t.Errorf("Expected 2 entries, got %d", len(entries))
	}

	if entries[0].Action != domain.ActionCreate {
		t.Errorf("Expected first action to be create, got %s", entries[0].Action)
	}
	if entries[1].Action != domain.ActionUpdate {
		t.Errorf("Expected second action to be update, got %s", entries[1].Action)
	}
}

func TestHistory_Empty(t *testing.T) {
	server := newMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v1/projects/testproject/tasks/abc123/history" && r.Method == "GET" {
			json.NewEncoder(w).Encode([]domain.AuditEntry{})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	})
	defer server.Close()

	host, port := parseURL(server.URL)
	c := client.NewClient(host, port, "testproject", "test@host:/path")

	entries, err := c.GetTaskHistory(context.Background(), "abc123")
	if err != nil {
		t.Fatalf("GetTaskHistory failed: %v", err)
	}

	if len(entries) != 0 {
		t.Errorf("Expected 0 entries, got %d", len(entries))
	}
}

func TestHistory_TaskNotFound(t *testing.T) {
	server := newMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error": map[string]interface{}{
				"code":    "TASK_NOT_FOUND",
				"message": "Task notfound not found",
				"context": map[string]interface{}{"id": "notfound"},
			},
		})
	})
	defer server.Close()

	host, port := parseURL(server.URL)
	c := client.NewClient(host, port, "testproject", "test@host:/path")

	_, err := c.GetTaskHistory(context.Background(), "notfound")
	if err == nil {
		t.Error("GetTaskHistory should fail for non-existent task")
	}

	domainErr, ok := err.(*domain.DomainError)
	if !ok {
		t.Errorf("Expected DomainError, got %T", err)
		return
	}

	if domainErr.Code != domain.ErrCodeTaskNotFound {
		t.Errorf("Expected TASK_NOT_FOUND error code, got %s", domainErr.Code)
	}
}
