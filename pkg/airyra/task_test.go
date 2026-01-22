package airyra

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestCreateTask(t *testing.T) {
	now := time.Now()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/projects/test-project/tasks" {
			t.Errorf("expected path /v1/projects/test-project/tasks, got %s", r.URL.Path)
		}
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("expected Content-Type application/json, got %s", r.Header.Get("Content-Type"))
		}
		if r.Header.Get("X-Airyra-Agent") != "test-agent" {
			t.Errorf("expected X-Airyra-Agent test-agent, got %s", r.Header.Get("X-Airyra-Agent"))
		}

		var req createTaskRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Errorf("failed to decode request body: %v", err)
		}
		if req.Title != "Test Task" {
			t.Errorf("expected title 'Test Task', got %s", req.Title)
		}
		if req.Description == nil || *req.Description != "Test description" {
			t.Errorf("expected description 'Test description', got %v", req.Description)
		}
		if req.Priority == nil || *req.Priority != PriorityHigh {
			t.Errorf("expected priority %d, got %v", PriorityHigh, req.Priority)
		}

		w.WriteHeader(http.StatusCreated)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(Task{
			ID:        "task-123",
			Title:     req.Title,
			Status:    StatusOpen,
			Priority:  *req.Priority,
			CreatedAt: now,
			UpdatedAt: now,
		})
	}))
	defer server.Close()

	client := newTestClient(t, server)
	task, err := client.CreateTask(context.Background(), "Test Task",
		WithDescription("Test description"),
		WithPriority(PriorityHigh),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if task.ID != "task-123" {
		t.Errorf("expected task ID task-123, got %s", task.ID)
	}
	if task.Title != "Test Task" {
		t.Errorf("expected title 'Test Task', got %s", task.Title)
	}
	if task.Status != StatusOpen {
		t.Errorf("expected status %s, got %s", StatusOpen, task.Status)
	}
}

func TestGetTask(t *testing.T) {
	now := time.Now()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/projects/test-project/tasks/task-123" {
			t.Errorf("expected path /v1/projects/test-project/tasks/task-123, got %s", r.URL.Path)
		}
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(Task{
			ID:        "task-123",
			Title:     "Test Task",
			Status:    StatusOpen,
			Priority:  PriorityNormal,
			CreatedAt: now,
			UpdatedAt: now,
		})
	}))
	defer server.Close()

	client := newTestClient(t, server)
	task, err := client.GetTask(context.Background(), "task-123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if task.ID != "task-123" {
		t.Errorf("expected task ID task-123, got %s", task.ID)
	}
}

func TestGetTaskNotFound(t *testing.T) {
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
	_, err := client.GetTask(context.Background(), "task-123")
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if !IsTaskNotFound(err) {
		t.Errorf("expected IsTaskNotFound to be true, got false")
	}
}

func TestListTasks(t *testing.T) {
	now := time.Now()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}

		// Check query parameters
		if r.URL.Query().Get("status") != "open" {
			t.Errorf("expected status=open, got %s", r.URL.Query().Get("status"))
		}
		if r.URL.Query().Get("page") != "1" {
			t.Errorf("expected page=1, got %s", r.URL.Query().Get("page"))
		}
		if r.URL.Query().Get("per_page") != "10" {
			t.Errorf("expected per_page=10, got %s", r.URL.Query().Get("per_page"))
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(paginatedTaskResponse{
			Data: []*Task{
				{ID: "task-1", Title: "Task 1", Status: StatusOpen, Priority: PriorityNormal, CreatedAt: now, UpdatedAt: now},
				{ID: "task-2", Title: "Task 2", Status: StatusOpen, Priority: PriorityNormal, CreatedAt: now, UpdatedAt: now},
			},
			Pagination: paginationResponse{
				Page:       1,
				PerPage:    10,
				Total:      2,
				TotalPages: 1,
			},
		})
	}))
	defer server.Close()

	client := newTestClient(t, server)
	tasks, err := client.ListTasks(context.Background(),
		WithStatus(StatusOpen),
		WithPage(1),
		WithPerPage(10),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(tasks.Tasks) != 2 {
		t.Errorf("expected 2 tasks, got %d", len(tasks.Tasks))
	}
	if tasks.Total != 2 {
		t.Errorf("expected total 2, got %d", tasks.Total)
	}
}

func TestListReadyTasks(t *testing.T) {
	now := time.Now()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/projects/test-project/tasks/ready" {
			t.Errorf("expected path /v1/projects/test-project/tasks/ready, got %s", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(paginatedTaskResponse{
			Data: []*Task{
				{ID: "task-1", Title: "Ready Task", Status: StatusOpen, Priority: PriorityNormal, CreatedAt: now, UpdatedAt: now},
			},
			Pagination: paginationResponse{
				Page:       1,
				PerPage:    20,
				Total:      1,
				TotalPages: 1,
			},
		})
	}))
	defer server.Close()

	client := newTestClient(t, server)
	tasks, err := client.ListReadyTasks(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(tasks.Tasks) != 1 {
		t.Errorf("expected 1 task, got %d", len(tasks.Tasks))
	}
}

func TestUpdateTask(t *testing.T) {
	now := time.Now()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/projects/test-project/tasks/task-123" {
			t.Errorf("expected path /v1/projects/test-project/tasks/task-123, got %s", r.URL.Path)
		}
		if r.Method != http.MethodPatch {
			t.Errorf("expected PATCH, got %s", r.Method)
		}

		var req updateTaskRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Errorf("failed to decode request body: %v", err)
		}
		if req.Title == nil || *req.Title != "Updated Title" {
			t.Errorf("expected title 'Updated Title', got %v", req.Title)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(Task{
			ID:        "task-123",
			Title:     *req.Title,
			Status:    StatusOpen,
			Priority:  PriorityNormal,
			CreatedAt: now,
			UpdatedAt: now,
		})
	}))
	defer server.Close()

	client := newTestClient(t, server)
	task, err := client.UpdateTask(context.Background(), "task-123",
		WithTitle("Updated Title"),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if task.Title != "Updated Title" {
		t.Errorf("expected title 'Updated Title', got %s", task.Title)
	}
}

func TestDeleteTask(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/projects/test-project/tasks/task-123" {
			t.Errorf("expected path /v1/projects/test-project/tasks/task-123, got %s", r.URL.Path)
		}
		if r.Method != http.MethodDelete {
			t.Errorf("expected DELETE, got %s", r.Method)
		}

		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	client := newTestClient(t, server)
	err := client.DeleteTask(context.Background(), "task-123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestClaimTask(t *testing.T) {
	now := time.Now()
	claimedBy := "test-agent"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/projects/test-project/tasks/task-123/claim" {
			t.Errorf("expected path /v1/projects/test-project/tasks/task-123/claim, got %s", r.URL.Path)
		}
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(Task{
			ID:        "task-123",
			Title:     "Test Task",
			Status:    StatusInProgress,
			Priority:  PriorityNormal,
			ClaimedBy: &claimedBy,
			ClaimedAt: &now,
			CreatedAt: now,
			UpdatedAt: now,
		})
	}))
	defer server.Close()

	client := newTestClient(t, server)
	task, err := client.ClaimTask(context.Background(), "task-123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if task.Status != StatusInProgress {
		t.Errorf("expected status %s, got %s", StatusInProgress, task.Status)
	}
	if task.ClaimedBy == nil || *task.ClaimedBy != "test-agent" {
		t.Errorf("expected claimed_by test-agent, got %v", task.ClaimedBy)
	}
}

func TestClaimTaskAlreadyClaimed(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusConflict)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(apiErrorResponse{
			Error: apiError{
				Code:    string(ErrCodeAlreadyClaimed),
				Message: "Task already claimed",
				Context: map[string]interface{}{
					"claimed_by": "other-agent",
					"claimed_at": "2024-01-01T00:00:00Z",
				},
			},
		})
	}))
	defer server.Close()

	client := newTestClient(t, server)
	_, err := client.ClaimTask(context.Background(), "task-123")
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if !IsAlreadyClaimed(err) {
		t.Errorf("expected IsAlreadyClaimed to be true, got false")
	}
}

func TestCompleteTask(t *testing.T) {
	now := time.Now()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/projects/test-project/tasks/task-123/done" {
			t.Errorf("expected path /v1/projects/test-project/tasks/task-123/done, got %s", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(Task{
			ID:        "task-123",
			Title:     "Test Task",
			Status:    StatusDone,
			Priority:  PriorityNormal,
			CreatedAt: now,
			UpdatedAt: now,
		})
	}))
	defer server.Close()

	client := newTestClient(t, server)
	task, err := client.CompleteTask(context.Background(), "task-123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if task.Status != StatusDone {
		t.Errorf("expected status %s, got %s", StatusDone, task.Status)
	}
}

func TestReleaseTask(t *testing.T) {
	now := time.Now()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/projects/test-project/tasks/task-123/release" {
			t.Errorf("expected path /v1/projects/test-project/tasks/task-123/release, got %s", r.URL.Path)
		}
		if r.URL.Query().Get("force") != "true" {
			t.Errorf("expected force=true, got %s", r.URL.Query().Get("force"))
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(Task{
			ID:        "task-123",
			Title:     "Test Task",
			Status:    StatusOpen,
			Priority:  PriorityNormal,
			CreatedAt: now,
			UpdatedAt: now,
		})
	}))
	defer server.Close()

	client := newTestClient(t, server)
	task, err := client.ReleaseTask(context.Background(), "task-123", true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if task.Status != StatusOpen {
		t.Errorf("expected status %s, got %s", StatusOpen, task.Status)
	}
}

func TestBlockTask(t *testing.T) {
	now := time.Now()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/projects/test-project/tasks/task-123/block" {
			t.Errorf("expected path /v1/projects/test-project/tasks/task-123/block, got %s", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(Task{
			ID:        "task-123",
			Title:     "Test Task",
			Status:    StatusBlocked,
			Priority:  PriorityNormal,
			CreatedAt: now,
			UpdatedAt: now,
		})
	}))
	defer server.Close()

	client := newTestClient(t, server)
	task, err := client.BlockTask(context.Background(), "task-123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if task.Status != StatusBlocked {
		t.Errorf("expected status %s, got %s", StatusBlocked, task.Status)
	}
}

func TestUnblockTask(t *testing.T) {
	now := time.Now()
	claimedBy := "test-agent"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/projects/test-project/tasks/task-123/unblock" {
			t.Errorf("expected path /v1/projects/test-project/tasks/task-123/unblock, got %s", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(Task{
			ID:        "task-123",
			Title:     "Test Task",
			Status:    StatusInProgress,
			Priority:  PriorityNormal,
			ClaimedBy: &claimedBy,
			CreatedAt: now,
			UpdatedAt: now,
		})
	}))
	defer server.Close()

	client := newTestClient(t, server)
	task, err := client.UnblockTask(context.Background(), "task-123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if task.Status != StatusInProgress {
		t.Errorf("expected status %s, got %s", StatusInProgress, task.Status)
	}
}

// Silence unused import warning
var _ = io.Discard
