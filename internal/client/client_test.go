package client

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/airyra/airyra/internal/domain"
)

// =============================================================================
// Request Building Tests
// =============================================================================

func TestClient_BaseURL(t *testing.T) {
	tests := []struct {
		name     string
		host     string
		port     int
		expected string
	}{
		{
			name:     "localhost with default port",
			host:     "localhost",
			port:     8080,
			expected: "http://localhost:8080",
		},
		{
			name:     "custom host and port",
			host:     "192.168.1.100",
			port:     9090,
			expected: "http://192.168.1.100:9090",
		},
		{
			name:     "hostname with subdomain",
			host:     "api.example.com",
			port:     443,
			expected: "http://api.example.com:443",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := NewClient(tt.host, tt.port, "test-project", "test-agent")
			if c.baseURL != tt.expected {
				t.Errorf("expected baseURL %q, got %q", tt.expected, c.baseURL)
			}
		})
	}
}

func TestClient_AgentHeader(t *testing.T) {
	var receivedHeader string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedHeader = r.Header.Get("X-Airyra-Agent")
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	}))
	defer server.Close()

	c := newTestClient(server, "test-project", "my-agent-id")

	ctx := context.Background()
	_ = c.Health(ctx)

	if receivedHeader != "my-agent-id" {
		t.Errorf("expected X-Airyra-Agent header %q, got %q", "my-agent-id", receivedHeader)
	}
}

func TestClient_ContentType(t *testing.T) {
	var receivedContentType string
	var receivedMethod string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedContentType = r.Header.Get("Content-Type")
		receivedMethod = r.Method
		w.Header().Set("Content-Type", "application/json")
		if r.Method == http.MethodPost || r.Method == http.MethodPatch {
			json.NewEncoder(w).Encode(&domain.Task{
				ID:        "task-123",
				Title:     "Test Task",
				Status:    domain.StatusOpen,
				Priority:  2,
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			})
		} else {
			json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
		}
	}))
	defer server.Close()

	c := newTestClient(server, "test-project", "agent")
	ctx := context.Background()

	// Test POST (CreateTask)
	_, _ = c.CreateTask(ctx, "Test", "Description", 2, "")
	if receivedMethod != http.MethodPost {
		t.Errorf("expected POST method, got %s", receivedMethod)
	}
	if receivedContentType != "application/json" {
		t.Errorf("expected Content-Type application/json for POST, got %q", receivedContentType)
	}

	// Test PATCH (UpdateTask)
	title := "Updated"
	_, _ = c.UpdateTask(ctx, "task-123", TaskUpdates{Title: &title})
	if receivedMethod != http.MethodPatch {
		t.Errorf("expected PATCH method, got %s", receivedMethod)
	}
	if receivedContentType != "application/json" {
		t.Errorf("expected Content-Type application/json for PATCH, got %q", receivedContentType)
	}

	// Test GET (Health) - should NOT have Content-Type set in request
	receivedContentType = ""
	_ = c.Health(ctx)
	if receivedMethod != http.MethodGet {
		t.Errorf("expected GET method, got %s", receivedMethod)
	}
	// GET requests typically don't need Content-Type, but we don't enforce it being empty
}

func TestClient_ProjectInPath(t *testing.T) {
	var receivedPath string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(&domain.Task{
			ID:        "task-123",
			Title:     "Test Task",
			Status:    domain.StatusOpen,
			Priority:  2,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		})
	}))
	defer server.Close()

	c := newTestClient(server, "my-awesome-project", "agent")
	ctx := context.Background()

	_, _ = c.GetTask(ctx, "task-123")

	expectedPath := "/v1/projects/my-awesome-project/tasks/task-123"
	if receivedPath != expectedPath {
		t.Errorf("expected path %q, got %q", expectedPath, receivedPath)
	}
}

// =============================================================================
// Health Tests
// =============================================================================

func TestHealth_ServerRunning(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/health" {
			t.Errorf("expected path /v1/health, got %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	}))
	defer server.Close()

	c := newTestClient(server, "test", "agent")
	ctx := context.Background()

	err := c.Health(ctx)
	if err != nil {
		t.Errorf("expected nil error, got %v", err)
	}
}

func TestHealth_ServerDown(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
		json.NewEncoder(w).Encode(map[string]string{"status": "unhealthy"})
	}))
	defer server.Close()

	c := newTestClient(server, "test", "agent")
	ctx := context.Background()

	err := c.Health(ctx)
	if err == nil {
		t.Error("expected error, got nil")
	}
	if !errors.Is(err, ErrServerUnhealthy) {
		t.Errorf("expected ErrServerUnhealthy, got %v", err)
	}
}

func TestHealth_ConnectionRefused(t *testing.T) {
	// Create a client pointing to a port that's definitely not listening
	c := NewClient("localhost", 59999, "test", "agent")
	ctx := context.Background()

	err := c.Health(ctx)
	if err == nil {
		t.Error("expected error, got nil")
	}
	if !errors.Is(err, ErrServerNotRunning) {
		t.Errorf("expected ErrServerNotRunning, got %v", err)
	}
}

// =============================================================================
// Task CRUD Tests
// =============================================================================

func TestCreateTask_Success(t *testing.T) {
	expectedTask := &domain.Task{
		ID:        "task-abc123",
		Title:     "New Task",
		Status:    domain.StatusOpen,
		Priority:  2,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/v1/projects/test-project/tasks" {
			t.Errorf("expected path /v1/projects/test-project/tasks, got %s", r.URL.Path)
		}

		var req createTaskRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Errorf("failed to decode request body: %v", err)
		}
		if req.Title != "New Task" {
			t.Errorf("expected title 'New Task', got %q", req.Title)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(expectedTask)
	}))
	defer server.Close()

	c := newTestClient(server, "test-project", "agent")
	ctx := context.Background()

	task, err := c.CreateTask(ctx, "New Task", "", 2, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if task.ID != expectedTask.ID {
		t.Errorf("expected task ID %q, got %q", expectedTask.ID, task.ID)
	}
	if task.Title != expectedTask.Title {
		t.Errorf("expected task title %q, got %q", expectedTask.Title, task.Title)
	}
}

func TestCreateTask_WithOptionalFields(t *testing.T) {
	var receivedReq createTaskRequest

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewDecoder(r.Body).Decode(&receivedReq)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(&domain.Task{
			ID:        "task-123",
			Title:     "Task",
			Status:    domain.StatusOpen,
			Priority:  1,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		})
	}))
	defer server.Close()

	c := newTestClient(server, "test-project", "agent")
	ctx := context.Background()

	_, err := c.CreateTask(ctx, "Task", "A description", 1, "parent-123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if receivedReq.Title != "Task" {
		t.Errorf("expected title 'Task', got %q", receivedReq.Title)
	}
	if receivedReq.Description == nil || *receivedReq.Description != "A description" {
		t.Errorf("expected description 'A description', got %v", receivedReq.Description)
	}
	if receivedReq.Priority == nil || *receivedReq.Priority != 1 {
		t.Errorf("expected priority 1, got %v", receivedReq.Priority)
	}
	if receivedReq.ParentID == nil || *receivedReq.ParentID != "parent-123" {
		t.Errorf("expected parentID 'parent-123', got %v", receivedReq.ParentID)
	}
}

func TestCreateTask_ValidationError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error": map[string]interface{}{
				"code":    "VALIDATION_FAILED",
				"message": "Validation failed",
				"context": map[string]interface{}{
					"details": []string{"title is required"},
				},
			},
		})
	}))
	defer server.Close()

	c := newTestClient(server, "test-project", "agent")
	ctx := context.Background()

	_, err := c.CreateTask(ctx, "", "", 2, "")
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	var domainErr *domain.DomainError
	if !errors.As(err, &domainErr) {
		t.Fatalf("expected DomainError, got %T", err)
	}
	if domainErr.Code != domain.ErrCodeValidationFailed {
		t.Errorf("expected code VALIDATION_FAILED, got %s", domainErr.Code)
	}
}

func TestGetTask_Success(t *testing.T) {
	expectedTask := &domain.Task{
		ID:        "task-xyz",
		Title:     "Existing Task",
		Status:    domain.StatusInProgress,
		Priority:  1,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/v1/projects/test-project/tasks/task-xyz" {
			t.Errorf("expected path /v1/projects/test-project/tasks/task-xyz, got %s", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(expectedTask)
	}))
	defer server.Close()

	c := newTestClient(server, "test-project", "agent")
	ctx := context.Background()

	task, err := c.GetTask(ctx, "task-xyz")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if task.ID != expectedTask.ID {
		t.Errorf("expected task ID %q, got %q", expectedTask.ID, task.ID)
	}
}

func TestGetTask_NotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error": map[string]interface{}{
				"code":    "TASK_NOT_FOUND",
				"message": "Task nonexistent not found",
				"context": map[string]interface{}{
					"id": "nonexistent",
				},
			},
		})
	}))
	defer server.Close()

	c := newTestClient(server, "test-project", "agent")
	ctx := context.Background()

	_, err := c.GetTask(ctx, "nonexistent")
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	var domainErr *domain.DomainError
	if !errors.As(err, &domainErr) {
		t.Fatalf("expected DomainError, got %T", err)
	}
	if domainErr.Code != domain.ErrCodeTaskNotFound {
		t.Errorf("expected code TASK_NOT_FOUND, got %s", domainErr.Code)
	}
}

func TestListTasks_Success(t *testing.T) {
	now := time.Now()
	tasks := []*domain.Task{
		{ID: "task-1", Title: "Task 1", Status: domain.StatusOpen, Priority: 2, CreatedAt: now, UpdatedAt: now},
		{ID: "task-2", Title: "Task 2", Status: domain.StatusOpen, Priority: 1, CreatedAt: now, UpdatedAt: now},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/v1/projects/test-project/tasks" {
			t.Errorf("expected path /v1/projects/test-project/tasks, got %s", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"data": tasks,
			"pagination": map[string]interface{}{
				"page":        1,
				"per_page":    50,
				"total":       2,
				"total_pages": 1,
			},
		})
	}))
	defer server.Close()

	c := newTestClient(server, "test-project", "agent")
	ctx := context.Background()

	result, err := c.ListTasks(ctx, "", 1, 50)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Data) != 2 {
		t.Errorf("expected 2 tasks, got %d", len(result.Data))
	}
	if result.Pagination.Total != 2 {
		t.Errorf("expected total 2, got %d", result.Pagination.Total)
	}
}

func TestListTasks_WithStatusFilter(t *testing.T) {
	var receivedQuery url.Values

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedQuery = r.URL.Query()
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"data": []*domain.Task{},
			"pagination": map[string]interface{}{
				"page":        1,
				"per_page":    50,
				"total":       0,
				"total_pages": 0,
			},
		})
	}))
	defer server.Close()

	c := newTestClient(server, "test-project", "agent")
	ctx := context.Background()

	_, err := c.ListTasks(ctx, "in_progress", 1, 50)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if receivedQuery.Get("status") != "in_progress" {
		t.Errorf("expected status=in_progress, got %q", receivedQuery.Get("status"))
	}
}

func TestListTasks_Pagination(t *testing.T) {
	var receivedQuery url.Values

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedQuery = r.URL.Query()
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"data": []*domain.Task{},
			"pagination": map[string]interface{}{
				"page":        2,
				"per_page":    10,
				"total":       25,
				"total_pages": 3,
			},
		})
	}))
	defer server.Close()

	c := newTestClient(server, "test-project", "agent")
	ctx := context.Background()

	result, err := c.ListTasks(ctx, "", 2, 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if receivedQuery.Get("page") != "2" {
		t.Errorf("expected page=2, got %q", receivedQuery.Get("page"))
	}
	if receivedQuery.Get("per_page") != "10" {
		t.Errorf("expected per_page=10, got %q", receivedQuery.Get("per_page"))
	}
	if result.Pagination.Page != 2 {
		t.Errorf("expected page 2, got %d", result.Pagination.Page)
	}
	if result.Pagination.TotalPages != 3 {
		t.Errorf("expected total_pages 3, got %d", result.Pagination.TotalPages)
	}
}

func TestListReadyTasks_Success(t *testing.T) {
	now := time.Now()
	tasks := []*domain.Task{
		{ID: "ready-1", Title: "Ready Task", Status: domain.StatusOpen, Priority: 1, CreatedAt: now, UpdatedAt: now},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/projects/test-project/tasks/ready" {
			t.Errorf("expected path /v1/projects/test-project/tasks/ready, got %s", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"data": tasks,
			"pagination": map[string]interface{}{
				"page":        1,
				"per_page":    50,
				"total":       1,
				"total_pages": 1,
			},
		})
	}))
	defer server.Close()

	c := newTestClient(server, "test-project", "agent")
	ctx := context.Background()

	result, err := c.ListReadyTasks(ctx, 1, 50)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Data) != 1 {
		t.Errorf("expected 1 task, got %d", len(result.Data))
	}
}

func TestUpdateTask_Success(t *testing.T) {
	var receivedReq updateTaskRequest

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch {
			t.Errorf("expected PATCH, got %s", r.Method)
		}
		json.NewDecoder(r.Body).Decode(&receivedReq)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(&domain.Task{
			ID:        "task-123",
			Title:     "Updated Title",
			Status:    domain.StatusOpen,
			Priority:  0,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		})
	}))
	defer server.Close()

	c := newTestClient(server, "test-project", "agent")
	ctx := context.Background()

	title := "Updated Title"
	priority := 0
	task, err := c.UpdateTask(ctx, "task-123", TaskUpdates{
		Title:    &title,
		Priority: &priority,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if task.Title != "Updated Title" {
		t.Errorf("expected title 'Updated Title', got %q", task.Title)
	}
	if receivedReq.Title == nil || *receivedReq.Title != "Updated Title" {
		t.Errorf("expected request title 'Updated Title', got %v", receivedReq.Title)
	}
	if receivedReq.Priority == nil || *receivedReq.Priority != 0 {
		t.Errorf("expected request priority 0, got %v", receivedReq.Priority)
	}
}

func TestDeleteTask_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("expected DELETE, got %s", r.Method)
		}
		if r.URL.Path != "/v1/projects/test-project/tasks/task-to-delete" {
			t.Errorf("expected path /v1/projects/test-project/tasks/task-to-delete, got %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	c := newTestClient(server, "test-project", "agent")
	ctx := context.Background()

	err := c.DeleteTask(ctx, "task-to-delete")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

// =============================================================================
// Status Transition Tests
// =============================================================================

func TestClaimTask_Success(t *testing.T) {
	agentID := "claiming-agent"
	now := time.Now()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/v1/projects/test-project/tasks/task-123/claim" {
			t.Errorf("expected path /v1/projects/test-project/tasks/task-123/claim, got %s", r.URL.Path)
		}
		if r.Header.Get("X-Airyra-Agent") != agentID {
			t.Errorf("expected agent header %q, got %q", agentID, r.Header.Get("X-Airyra-Agent"))
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(&domain.Task{
			ID:        "task-123",
			Title:     "Task",
			Status:    domain.StatusInProgress,
			Priority:  2,
			ClaimedBy: &agentID,
			ClaimedAt: &now,
			CreatedAt: now,
			UpdatedAt: now,
		})
	}))
	defer server.Close()

	c := newTestClient(server, "test-project", agentID)
	ctx := context.Background()

	task, err := c.ClaimTask(ctx, "task-123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if task.Status != domain.StatusInProgress {
		t.Errorf("expected status in_progress, got %s", task.Status)
	}
	if task.ClaimedBy == nil || *task.ClaimedBy != agentID {
		t.Errorf("expected claimed_by %q, got %v", agentID, task.ClaimedBy)
	}
}

func TestClaimTask_AlreadyClaimed(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusConflict)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error": map[string]interface{}{
				"code":    "ALREADY_CLAIMED",
				"message": "Task already claimed by another agent",
				"context": map[string]interface{}{
					"claimed_by": "other-agent",
					"claimed_at": "2024-01-01T00:00:00Z",
				},
			},
		})
	}))
	defer server.Close()

	c := newTestClient(server, "test-project", "my-agent")
	ctx := context.Background()

	_, err := c.ClaimTask(ctx, "task-123")
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	var domainErr *domain.DomainError
	if !errors.As(err, &domainErr) {
		t.Fatalf("expected DomainError, got %T", err)
	}
	if domainErr.Code != domain.ErrCodeAlreadyClaimed {
		t.Errorf("expected code ALREADY_CLAIMED, got %s", domainErr.Code)
	}
}

func TestCompleteTask_Success(t *testing.T) {
	now := time.Now()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/projects/test-project/tasks/task-123/done" {
			t.Errorf("expected path /v1/projects/test-project/tasks/task-123/done, got %s", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(&domain.Task{
			ID:        "task-123",
			Title:     "Task",
			Status:    domain.StatusDone,
			Priority:  2,
			CreatedAt: now,
			UpdatedAt: now,
		})
	}))
	defer server.Close()

	c := newTestClient(server, "test-project", "agent")
	ctx := context.Background()

	task, err := c.CompleteTask(ctx, "task-123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if task.Status != domain.StatusDone {
		t.Errorf("expected status done, got %s", task.Status)
	}
}

func TestCompleteTask_NotOwner(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusForbidden)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error": map[string]interface{}{
				"code":    "NOT_OWNER",
				"message": "Task is claimed by another agent",
				"context": map[string]interface{}{
					"claimed_by": "other-agent",
				},
			},
		})
	}))
	defer server.Close()

	c := newTestClient(server, "test-project", "my-agent")
	ctx := context.Background()

	_, err := c.CompleteTask(ctx, "task-123")
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	var domainErr *domain.DomainError
	if !errors.As(err, &domainErr) {
		t.Fatalf("expected DomainError, got %T", err)
	}
	if domainErr.Code != domain.ErrCodeNotOwner {
		t.Errorf("expected code NOT_OWNER, got %s", domainErr.Code)
	}
}

func TestReleaseTask_Success(t *testing.T) {
	now := time.Now()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/projects/test-project/tasks/task-123/release" {
			t.Errorf("expected path /v1/projects/test-project/tasks/task-123/release, got %s", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(&domain.Task{
			ID:        "task-123",
			Title:     "Task",
			Status:    domain.StatusOpen,
			Priority:  2,
			CreatedAt: now,
			UpdatedAt: now,
		})
	}))
	defer server.Close()

	c := newTestClient(server, "test-project", "agent")
	ctx := context.Background()

	task, err := c.ReleaseTask(ctx, "task-123", false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if task.Status != domain.StatusOpen {
		t.Errorf("expected status open, got %s", task.Status)
	}
}

func TestReleaseTask_WithForce(t *testing.T) {
	var receivedQuery url.Values

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedQuery = r.URL.Query()
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(&domain.Task{
			ID:        "task-123",
			Title:     "Task",
			Status:    domain.StatusOpen,
			Priority:  2,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		})
	}))
	defer server.Close()

	c := newTestClient(server, "test-project", "agent")
	ctx := context.Background()

	_, err := c.ReleaseTask(ctx, "task-123", true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if receivedQuery.Get("force") != "true" {
		t.Errorf("expected force=true, got %q", receivedQuery.Get("force"))
	}
}

func TestBlockTask_Success(t *testing.T) {
	now := time.Now()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/projects/test-project/tasks/task-123/block" {
			t.Errorf("expected path /v1/projects/test-project/tasks/task-123/block, got %s", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(&domain.Task{
			ID:        "task-123",
			Title:     "Task",
			Status:    domain.StatusBlocked,
			Priority:  2,
			CreatedAt: now,
			UpdatedAt: now,
		})
	}))
	defer server.Close()

	c := newTestClient(server, "test-project", "agent")
	ctx := context.Background()

	task, err := c.BlockTask(ctx, "task-123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if task.Status != domain.StatusBlocked {
		t.Errorf("expected status blocked, got %s", task.Status)
	}
}

func TestUnblockTask_Success(t *testing.T) {
	now := time.Now()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/projects/test-project/tasks/task-123/unblock" {
			t.Errorf("expected path /v1/projects/test-project/tasks/task-123/unblock, got %s", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(&domain.Task{
			ID:        "task-123",
			Title:     "Task",
			Status:    domain.StatusOpen,
			Priority:  2,
			CreatedAt: now,
			UpdatedAt: now,
		})
	}))
	defer server.Close()

	c := newTestClient(server, "test-project", "agent")
	ctx := context.Background()

	task, err := c.UnblockTask(ctx, "task-123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if task.Status != domain.StatusOpen {
		t.Errorf("expected status open, got %s", task.Status)
	}
}

// =============================================================================
// Dependency Tests
// =============================================================================

func TestAddDependency_Success(t *testing.T) {
	var receivedReq addDependencyRequest

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/v1/projects/test-project/tasks/child-task/deps" {
			t.Errorf("expected path /v1/projects/test-project/tasks/child-task/deps, got %s", r.URL.Path)
		}

		json.NewDecoder(r.Body).Decode(&receivedReq)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]string{
			"child_id":  "child-task",
			"parent_id": "parent-task",
		})
	}))
	defer server.Close()

	c := newTestClient(server, "test-project", "agent")
	ctx := context.Background()

	err := c.AddDependency(ctx, "child-task", "parent-task")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if receivedReq.ParentID != "parent-task" {
		t.Errorf("expected parent_id 'parent-task', got %q", receivedReq.ParentID)
	}
}

func TestAddDependency_CycleDetected(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error": map[string]interface{}{
				"code":    "CYCLE_DETECTED",
				"message": "Adding this dependency would create a cycle",
				"context": map[string]interface{}{
					"path": []string{"task-a", "task-b", "task-c", "task-a"},
				},
			},
		})
	}))
	defer server.Close()

	c := newTestClient(server, "test-project", "agent")
	ctx := context.Background()

	err := c.AddDependency(ctx, "task-a", "task-c")
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	var domainErr *domain.DomainError
	if !errors.As(err, &domainErr) {
		t.Fatalf("expected DomainError, got %T", err)
	}
	if domainErr.Code != domain.ErrCodeCycleDetected {
		t.Errorf("expected code CYCLE_DETECTED, got %s", domainErr.Code)
	}
}

func TestRemoveDependency_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("expected DELETE, got %s", r.Method)
		}
		if r.URL.Path != "/v1/projects/test-project/tasks/child-task/deps/parent-task" {
			t.Errorf("expected path /v1/projects/test-project/tasks/child-task/deps/parent-task, got %s", r.URL.Path)
		}

		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	c := newTestClient(server, "test-project", "agent")
	ctx := context.Background()

	err := c.RemoveDependency(ctx, "child-task", "parent-task")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestListDependencies_Success(t *testing.T) {
	deps := []domain.Dependency{
		{ChildID: "task-123", ParentID: "parent-1"},
		{ChildID: "task-123", ParentID: "parent-2"},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/v1/projects/test-project/tasks/task-123/deps" {
			t.Errorf("expected path /v1/projects/test-project/tasks/task-123/deps, got %s", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(deps)
	}))
	defer server.Close()

	c := newTestClient(server, "test-project", "agent")
	ctx := context.Background()

	result, err := c.ListDependencies(ctx, "task-123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 2 {
		t.Errorf("expected 2 dependencies, got %d", len(result))
	}
}

// =============================================================================
// Audit Tests
// =============================================================================

func TestGetTaskHistory_Success(t *testing.T) {
	now := time.Now()
	entries := []domain.AuditEntry{
		{ID: 1, TaskID: "task-123", Action: domain.ActionCreate, ChangedAt: now, ChangedBy: "agent-1"},
		{ID: 2, TaskID: "task-123", Action: domain.ActionClaim, ChangedAt: now, ChangedBy: "agent-1"},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/v1/projects/test-project/tasks/task-123/history" {
			t.Errorf("expected path /v1/projects/test-project/tasks/task-123/history, got %s", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(entries)
	}))
	defer server.Close()

	c := newTestClient(server, "test-project", "agent")
	ctx := context.Background()

	result, err := c.GetTaskHistory(ctx, "task-123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 2 {
		t.Errorf("expected 2 entries, got %d", len(result))
	}
	if result[0].Action != domain.ActionCreate {
		t.Errorf("expected first action 'create', got %s", result[0].Action)
	}
}

// =============================================================================
// System Tests
// =============================================================================

func TestListProjects_Success(t *testing.T) {
	projects := []string{"project-1", "project-2", "project-3"}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/v1/projects" {
			t.Errorf("expected path /v1/projects, got %s", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(projects)
	}))
	defer server.Close()

	c := newTestClient(server, "test-project", "agent")
	ctx := context.Background()

	result, err := c.ListProjects(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 3 {
		t.Errorf("expected 3 projects, got %d", len(result))
	}
}

// =============================================================================
// Error Mapping Tests
// =============================================================================

func TestParseError_TaskNotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error": map[string]interface{}{
				"code":    "TASK_NOT_FOUND",
				"message": "Task abc not found",
				"context": map[string]interface{}{
					"id": "abc",
				},
			},
		})
	}))
	defer server.Close()

	c := newTestClient(server, "test-project", "agent")
	ctx := context.Background()

	_, err := c.GetTask(ctx, "abc")
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	var domainErr *domain.DomainError
	if !errors.As(err, &domainErr) {
		t.Fatalf("expected DomainError, got %T", err)
	}
	if domainErr.Code != domain.ErrCodeTaskNotFound {
		t.Errorf("expected code TASK_NOT_FOUND, got %s", domainErr.Code)
	}
}

func TestParseError_AlreadyClaimed(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusConflict)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error": map[string]interface{}{
				"code":    "ALREADY_CLAIMED",
				"message": "Task already claimed by another agent",
				"context": map[string]interface{}{
					"claimed_by": "other-agent",
					"claimed_at": "2024-01-01T00:00:00Z",
				},
			},
		})
	}))
	defer server.Close()

	c := newTestClient(server, "test-project", "agent")
	ctx := context.Background()

	_, err := c.ClaimTask(ctx, "task-123")
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	var domainErr *domain.DomainError
	if !errors.As(err, &domainErr) {
		t.Fatalf("expected DomainError, got %T", err)
	}
	if domainErr.Code != domain.ErrCodeAlreadyClaimed {
		t.Errorf("expected code ALREADY_CLAIMED, got %s", domainErr.Code)
	}
}

func TestParseError_NotOwner(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusForbidden)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error": map[string]interface{}{
				"code":    "NOT_OWNER",
				"message": "Task is claimed by another agent",
				"context": map[string]interface{}{
					"claimed_by": "other-agent",
				},
			},
		})
	}))
	defer server.Close()

	c := newTestClient(server, "test-project", "agent")
	ctx := context.Background()

	_, err := c.CompleteTask(ctx, "task-123")
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	var domainErr *domain.DomainError
	if !errors.As(err, &domainErr) {
		t.Fatalf("expected DomainError, got %T", err)
	}
	if domainErr.Code != domain.ErrCodeNotOwner {
		t.Errorf("expected code NOT_OWNER, got %s", domainErr.Code)
	}
}

func TestParseError_CycleDetected(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error": map[string]interface{}{
				"code":    "CYCLE_DETECTED",
				"message": "Adding this dependency would create a cycle",
				"context": map[string]interface{}{
					"path": []string{"a", "b", "c", "a"},
				},
			},
		})
	}))
	defer server.Close()

	c := newTestClient(server, "test-project", "agent")
	ctx := context.Background()

	err := c.AddDependency(ctx, "a", "c")
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	var domainErr *domain.DomainError
	if !errors.As(err, &domainErr) {
		t.Fatalf("expected DomainError, got %T", err)
	}
	if domainErr.Code != domain.ErrCodeCycleDetected {
		t.Errorf("expected code CYCLE_DETECTED, got %s", domainErr.Code)
	}
	// Check that path was extracted
	if domainErr.Context != nil {
		path, ok := domainErr.Context["path"].([]string)
		if ok && len(path) != 4 {
			t.Errorf("expected path length 4, got %d", len(path))
		}
	}
}

// =============================================================================
// Helper Functions
// =============================================================================

// newTestClient creates a client pointing to a test server.
func newTestClient(server *httptest.Server, project, agentID string) *Client {
	// Parse the test server URL to extract host and port
	u, _ := url.Parse(server.URL)
	host := strings.Split(u.Host, ":")[0]
	port := 80
	if p := strings.Split(u.Host, ":"); len(p) > 1 {
		fmt.Sscanf(p[1], "%d", &port)
	}

	c := NewClient(host, port, project, agentID)
	// Override the baseURL to use the test server URL directly
	c.baseURL = server.URL
	return c
}
