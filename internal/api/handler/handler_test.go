package handler_test

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/go-chi/chi/v5"

	"github.com/airyra/airyra/internal/api"
	"github.com/airyra/airyra/internal/api/middleware"
	"github.com/airyra/airyra/internal/api/response"
	"github.com/airyra/airyra/internal/store"
)

// testSetup provides common test infrastructure
type testSetup struct {
	manager *store.Manager
	router  *chi.Mux
	tmpDir  string
}

func newTestSetup(t *testing.T) *testSetup {
	t.Helper()

	tmpDir, err := os.MkdirTemp("", "airyra-handler-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}

	manager, err := store.NewManager(tmpDir)
	if err != nil {
		os.RemoveAll(tmpDir)
		t.Fatalf("failed to create manager: %v", err)
	}

	router := api.NewRouter(manager)

	return &testSetup{
		manager: manager,
		router:  router,
		tmpDir:  tmpDir,
	}
}

func (s *testSetup) cleanup() {
	s.manager.Close()
	os.RemoveAll(s.tmpDir)
}

func (s *testSetup) doRequest(method, path string, body interface{}, headers map[string]string) *httptest.ResponseRecorder {
	var reqBody bytes.Buffer
	if body != nil {
		json.NewEncoder(&reqBody).Encode(body)
	}

	req := httptest.NewRequest(method, path, &reqBody)
	req.Header.Set("Content-Type", "application/json")
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	rr := httptest.NewRecorder()
	s.router.ServeHTTP(rr, req)
	return rr
}

// ========================
// System Tests
// ========================

func TestHealth_ReturnsOK(t *testing.T) {
	setup := newTestSetup(t)
	defer setup.cleanup()

	rr := setup.doRequest("GET", "/v1/health", nil, nil)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rr.Code)
	}

	var resp map[string]string
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp["status"] != "ok" {
		t.Errorf("expected status 'ok', got %q", resp["status"])
	}
}

func TestListProjects_Empty(t *testing.T) {
	setup := newTestSetup(t)
	defer setup.cleanup()

	rr := setup.doRequest("GET", "/v1/projects", nil, nil)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rr.Code)
	}

	var projects []string
	if err := json.NewDecoder(rr.Body).Decode(&projects); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if len(projects) != 0 {
		t.Errorf("expected empty projects list, got %v", projects)
	}
}

func TestListProjects_WithProjects(t *testing.T) {
	setup := newTestSetup(t)
	defer setup.cleanup()

	// Create some projects by creating databases
	_, err := setup.manager.GetDB("project-alpha")
	if err != nil {
		t.Fatalf("failed to create project: %v", err)
	}
	_, err = setup.manager.GetDB("project-beta")
	if err != nil {
		t.Fatalf("failed to create project: %v", err)
	}

	rr := setup.doRequest("GET", "/v1/projects", nil, nil)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rr.Code)
	}

	var projects []string
	if err := json.NewDecoder(rr.Body).Decode(&projects); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if len(projects) != 2 {
		t.Errorf("expected 2 projects, got %d: %v", len(projects), projects)
	}
}

// ========================
// Task CRUD Tests
// ========================

func TestCreateTask_Success(t *testing.T) {
	setup := newTestSetup(t)
	defer setup.cleanup()

	body := map[string]interface{}{
		"title": "Test Task",
	}

	rr := setup.doRequest("POST", "/v1/projects/testproj/tasks", body, nil)

	if rr.Code != http.StatusCreated {
		t.Errorf("expected status 201, got %d: %s", rr.Code, rr.Body.String())
	}

	var task map[string]interface{}
	if err := json.NewDecoder(rr.Body).Decode(&task); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if task["id"] == nil || task["id"] == "" {
		t.Error("expected task to have an ID")
	}
	if task["title"] != "Test Task" {
		t.Errorf("expected title 'Test Task', got %v", task["title"])
	}
	if task["status"] != "open" {
		t.Errorf("expected status 'open', got %v", task["status"])
	}
}

func TestCreateTask_MissingTitle(t *testing.T) {
	setup := newTestSetup(t)
	defer setup.cleanup()

	body := map[string]interface{}{
		"description": "A task without a title",
	}

	rr := setup.doRequest("POST", "/v1/projects/testproj/tasks", body, nil)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", rr.Code)
	}

	var resp response.ErrorResponse
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp.Error.Code != "VALIDATION_FAILED" {
		t.Errorf("expected code 'VALIDATION_FAILED', got %q", resp.Error.Code)
	}
}

func TestGetTask_Success(t *testing.T) {
	setup := newTestSetup(t)
	defer setup.cleanup()

	// Create a task first
	createBody := map[string]interface{}{"title": "Test Task"}
	createRR := setup.doRequest("POST", "/v1/projects/testproj/tasks", createBody, nil)
	var created map[string]interface{}
	json.NewDecoder(createRR.Body).Decode(&created)
	taskID := created["id"].(string)

	// Get the task
	rr := setup.doRequest("GET", fmt.Sprintf("/v1/projects/testproj/tasks/%s", taskID), nil, nil)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var task map[string]interface{}
	if err := json.NewDecoder(rr.Body).Decode(&task); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if task["id"] != taskID {
		t.Errorf("expected id %q, got %v", taskID, task["id"])
	}
}

func TestGetTask_NotFound(t *testing.T) {
	setup := newTestSetup(t)
	defer setup.cleanup()

	// Initialize the project first
	setup.manager.GetDB("testproj")

	rr := setup.doRequest("GET", "/v1/projects/testproj/tasks/ar-nonexistent", nil, nil)

	if rr.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d: %s", rr.Code, rr.Body.String())
	}

	var resp response.ErrorResponse
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp.Error.Code != "TASK_NOT_FOUND" {
		t.Errorf("expected code 'TASK_NOT_FOUND', got %q", resp.Error.Code)
	}
}

func TestListTasks_Empty(t *testing.T) {
	setup := newTestSetup(t)
	defer setup.cleanup()

	// Initialize the project
	setup.manager.GetDB("testproj")

	rr := setup.doRequest("GET", "/v1/projects/testproj/tasks", nil, nil)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var resp response.PaginatedResponse
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	tasks := resp.Data.([]interface{})
	if len(tasks) != 0 {
		t.Errorf("expected empty task list, got %v", tasks)
	}
}

func TestListTasks_WithTasks(t *testing.T) {
	setup := newTestSetup(t)
	defer setup.cleanup()

	// Create some tasks
	for i := 1; i <= 3; i++ {
		body := map[string]interface{}{"title": fmt.Sprintf("Task %d", i)}
		setup.doRequest("POST", "/v1/projects/testproj/tasks", body, nil)
	}

	rr := setup.doRequest("GET", "/v1/projects/testproj/tasks", nil, nil)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rr.Code)
	}

	var resp response.PaginatedResponse
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	tasks := resp.Data.([]interface{})
	if len(tasks) != 3 {
		t.Errorf("expected 3 tasks, got %d", len(tasks))
	}

	// Check pagination metadata
	if resp.Pagination.Total != 3 {
		t.Errorf("expected total 3, got %d", resp.Pagination.Total)
	}
}

func TestListTasks_FilterByStatus(t *testing.T) {
	setup := newTestSetup(t)
	defer setup.cleanup()

	// Create tasks
	body1 := map[string]interface{}{"title": "Open Task"}
	setup.doRequest("POST", "/v1/projects/testproj/tasks", body1, nil)

	body2 := map[string]interface{}{"title": "Another Task"}
	rr2 := setup.doRequest("POST", "/v1/projects/testproj/tasks", body2, nil)
	var task2 map[string]interface{}
	json.NewDecoder(rr2.Body).Decode(&task2)

	// Claim the second task to make it in_progress
	headers := map[string]string{middleware.AgentHeader: "test-agent"}
	setup.doRequest("POST", fmt.Sprintf("/v1/projects/testproj/tasks/%s/claim", task2["id"]), nil, headers)

	// List only open tasks
	rr := setup.doRequest("GET", "/v1/projects/testproj/tasks?status=open", nil, nil)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rr.Code)
	}

	var resp response.PaginatedResponse
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	tasks := resp.Data.([]interface{})
	if len(tasks) != 1 {
		t.Errorf("expected 1 open task, got %d", len(tasks))
	}
}

func TestListReadyTasks(t *testing.T) {
	setup := newTestSetup(t)
	defer setup.cleanup()

	// Create a task - without dependencies, it should be ready
	body := map[string]interface{}{"title": "Ready Task"}
	setup.doRequest("POST", "/v1/projects/testproj/tasks", body, nil)

	rr := setup.doRequest("GET", "/v1/projects/testproj/tasks/ready", nil, nil)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var resp response.PaginatedResponse
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	tasks := resp.Data.([]interface{})
	if len(tasks) != 1 {
		t.Errorf("expected 1 ready task, got %d", len(tasks))
	}
}

func TestUpdateTask_Success(t *testing.T) {
	setup := newTestSetup(t)
	defer setup.cleanup()

	// Create a task first
	createBody := map[string]interface{}{"title": "Original Title"}
	createRR := setup.doRequest("POST", "/v1/projects/testproj/tasks", createBody, nil)
	var created map[string]interface{}
	json.NewDecoder(createRR.Body).Decode(&created)
	taskID := created["id"].(string)

	// Update the task
	updateBody := map[string]interface{}{"title": "Updated Title"}
	rr := setup.doRequest("PATCH", fmt.Sprintf("/v1/projects/testproj/tasks/%s", taskID), updateBody, nil)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var task map[string]interface{}
	if err := json.NewDecoder(rr.Body).Decode(&task); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if task["title"] != "Updated Title" {
		t.Errorf("expected title 'Updated Title', got %v", task["title"])
	}
}

func TestDeleteTask_Success(t *testing.T) {
	setup := newTestSetup(t)
	defer setup.cleanup()

	// Create a task first
	createBody := map[string]interface{}{"title": "Task to delete"}
	createRR := setup.doRequest("POST", "/v1/projects/testproj/tasks", createBody, nil)
	var created map[string]interface{}
	json.NewDecoder(createRR.Body).Decode(&created)
	taskID := created["id"].(string)

	// Delete the task
	rr := setup.doRequest("DELETE", fmt.Sprintf("/v1/projects/testproj/tasks/%s", taskID), nil, nil)

	if rr.Code != http.StatusNoContent {
		t.Errorf("expected status 204, got %d: %s", rr.Code, rr.Body.String())
	}

	// Verify it's gone
	getRR := setup.doRequest("GET", fmt.Sprintf("/v1/projects/testproj/tasks/%s", taskID), nil, nil)
	if getRR.Code != http.StatusNotFound {
		t.Errorf("expected task to be deleted (404), got %d", getRR.Code)
	}
}

// ========================
// Status Transition Tests
// ========================

func TestClaimTask_Success(t *testing.T) {
	setup := newTestSetup(t)
	defer setup.cleanup()

	// Create a task
	createBody := map[string]interface{}{"title": "Task to claim"}
	createRR := setup.doRequest("POST", "/v1/projects/testproj/tasks", createBody, nil)
	var created map[string]interface{}
	json.NewDecoder(createRR.Body).Decode(&created)
	taskID := created["id"].(string)

	// Claim the task
	headers := map[string]string{middleware.AgentHeader: "agent-123"}
	rr := setup.doRequest("POST", fmt.Sprintf("/v1/projects/testproj/tasks/%s/claim", taskID), nil, headers)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var task map[string]interface{}
	if err := json.NewDecoder(rr.Body).Decode(&task); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if task["status"] != "in_progress" {
		t.Errorf("expected status 'in_progress', got %v", task["status"])
	}
	if task["claimed_by"] != "agent-123" {
		t.Errorf("expected claimed_by 'agent-123', got %v", task["claimed_by"])
	}
}

func TestClaimTask_AlreadyClaimed(t *testing.T) {
	setup := newTestSetup(t)
	defer setup.cleanup()

	// Create a task
	createBody := map[string]interface{}{"title": "Task to claim"}
	createRR := setup.doRequest("POST", "/v1/projects/testproj/tasks", createBody, nil)
	var created map[string]interface{}
	json.NewDecoder(createRR.Body).Decode(&created)
	taskID := created["id"].(string)

	// First agent claims
	headers1 := map[string]string{middleware.AgentHeader: "agent-1"}
	setup.doRequest("POST", fmt.Sprintf("/v1/projects/testproj/tasks/%s/claim", taskID), nil, headers1)

	// Second agent tries to claim
	headers2 := map[string]string{middleware.AgentHeader: "agent-2"}
	rr := setup.doRequest("POST", fmt.Sprintf("/v1/projects/testproj/tasks/%s/claim", taskID), nil, headers2)

	if rr.Code != http.StatusConflict {
		t.Errorf("expected status 409, got %d: %s", rr.Code, rr.Body.String())
	}

	var resp response.ErrorResponse
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp.Error.Code != "ALREADY_CLAIMED" {
		t.Errorf("expected code 'ALREADY_CLAIMED', got %q", resp.Error.Code)
	}
}

func TestCompleteTask_Success(t *testing.T) {
	setup := newTestSetup(t)
	defer setup.cleanup()

	// Create and claim a task
	createBody := map[string]interface{}{"title": "Task to complete"}
	createRR := setup.doRequest("POST", "/v1/projects/testproj/tasks", createBody, nil)
	var created map[string]interface{}
	json.NewDecoder(createRR.Body).Decode(&created)
	taskID := created["id"].(string)

	headers := map[string]string{middleware.AgentHeader: "agent-123"}
	setup.doRequest("POST", fmt.Sprintf("/v1/projects/testproj/tasks/%s/claim", taskID), nil, headers)

	// Complete the task
	rr := setup.doRequest("POST", fmt.Sprintf("/v1/projects/testproj/tasks/%s/done", taskID), nil, headers)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var task map[string]interface{}
	if err := json.NewDecoder(rr.Body).Decode(&task); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if task["status"] != "done" {
		t.Errorf("expected status 'done', got %v", task["status"])
	}
}

func TestCompleteTask_NotOwner(t *testing.T) {
	setup := newTestSetup(t)
	defer setup.cleanup()

	// Create and claim a task with one agent
	createBody := map[string]interface{}{"title": "Task to complete"}
	createRR := setup.doRequest("POST", "/v1/projects/testproj/tasks", createBody, nil)
	var created map[string]interface{}
	json.NewDecoder(createRR.Body).Decode(&created)
	taskID := created["id"].(string)

	headers1 := map[string]string{middleware.AgentHeader: "agent-1"}
	setup.doRequest("POST", fmt.Sprintf("/v1/projects/testproj/tasks/%s/claim", taskID), nil, headers1)

	// Try to complete with another agent
	headers2 := map[string]string{middleware.AgentHeader: "agent-2"}
	rr := setup.doRequest("POST", fmt.Sprintf("/v1/projects/testproj/tasks/%s/done", taskID), nil, headers2)

	if rr.Code != http.StatusForbidden {
		t.Errorf("expected status 403, got %d: %s", rr.Code, rr.Body.String())
	}

	var resp response.ErrorResponse
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp.Error.Code != "NOT_OWNER" {
		t.Errorf("expected code 'NOT_OWNER', got %q", resp.Error.Code)
	}
}

func TestReleaseTask_Success(t *testing.T) {
	setup := newTestSetup(t)
	defer setup.cleanup()

	// Create and claim a task
	createBody := map[string]interface{}{"title": "Task to release"}
	createRR := setup.doRequest("POST", "/v1/projects/testproj/tasks", createBody, nil)
	var created map[string]interface{}
	json.NewDecoder(createRR.Body).Decode(&created)
	taskID := created["id"].(string)

	headers := map[string]string{middleware.AgentHeader: "agent-123"}
	setup.doRequest("POST", fmt.Sprintf("/v1/projects/testproj/tasks/%s/claim", taskID), nil, headers)

	// Release the task
	rr := setup.doRequest("POST", fmt.Sprintf("/v1/projects/testproj/tasks/%s/release", taskID), nil, headers)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var task map[string]interface{}
	if err := json.NewDecoder(rr.Body).Decode(&task); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if task["status"] != "open" {
		t.Errorf("expected status 'open', got %v", task["status"])
	}
	if task["claimed_by"] != nil {
		t.Errorf("expected claimed_by to be nil, got %v", task["claimed_by"])
	}
}

func TestBlockTask_Success(t *testing.T) {
	setup := newTestSetup(t)
	defer setup.cleanup()

	// Create a task
	createBody := map[string]interface{}{"title": "Task to block"}
	createRR := setup.doRequest("POST", "/v1/projects/testproj/tasks", createBody, nil)
	var created map[string]interface{}
	json.NewDecoder(createRR.Body).Decode(&created)
	taskID := created["id"].(string)

	// Block the task
	rr := setup.doRequest("POST", fmt.Sprintf("/v1/projects/testproj/tasks/%s/block", taskID), nil, nil)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var task map[string]interface{}
	if err := json.NewDecoder(rr.Body).Decode(&task); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if task["status"] != "blocked" {
		t.Errorf("expected status 'blocked', got %v", task["status"])
	}
}

func TestUnblockTask_Success(t *testing.T) {
	setup := newTestSetup(t)
	defer setup.cleanup()

	// Create and block a task
	createBody := map[string]interface{}{"title": "Task to unblock"}
	createRR := setup.doRequest("POST", "/v1/projects/testproj/tasks", createBody, nil)
	var created map[string]interface{}
	json.NewDecoder(createRR.Body).Decode(&created)
	taskID := created["id"].(string)

	setup.doRequest("POST", fmt.Sprintf("/v1/projects/testproj/tasks/%s/block", taskID), nil, nil)

	// Unblock the task
	rr := setup.doRequest("POST", fmt.Sprintf("/v1/projects/testproj/tasks/%s/unblock", taskID), nil, nil)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var task map[string]interface{}
	if err := json.NewDecoder(rr.Body).Decode(&task); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if task["status"] != "open" {
		t.Errorf("expected status 'open', got %v", task["status"])
	}
}

// ========================
// Dependency Tests
// ========================

func TestAddDependency_Success(t *testing.T) {
	setup := newTestSetup(t)
	defer setup.cleanup()

	// Create two tasks
	body1 := map[string]interface{}{"title": "Child Task"}
	rr1 := setup.doRequest("POST", "/v1/projects/testproj/tasks", body1, nil)
	var child map[string]interface{}
	json.NewDecoder(rr1.Body).Decode(&child)

	body2 := map[string]interface{}{"title": "Parent Task"}
	rr2 := setup.doRequest("POST", "/v1/projects/testproj/tasks", body2, nil)
	var parent map[string]interface{}
	json.NewDecoder(rr2.Body).Decode(&parent)

	// Add dependency: child depends on parent
	depBody := map[string]interface{}{"parent_id": parent["id"]}
	rr := setup.doRequest("POST", fmt.Sprintf("/v1/projects/testproj/tasks/%s/deps", child["id"]), depBody, nil)

	if rr.Code != http.StatusCreated {
		t.Errorf("expected status 201, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestAddDependency_Cycle(t *testing.T) {
	setup := newTestSetup(t)
	defer setup.cleanup()

	// Create two tasks
	body1 := map[string]interface{}{"title": "Task A"}
	rr1 := setup.doRequest("POST", "/v1/projects/testproj/tasks", body1, nil)
	var taskA map[string]interface{}
	json.NewDecoder(rr1.Body).Decode(&taskA)

	body2 := map[string]interface{}{"title": "Task B"}
	rr2 := setup.doRequest("POST", "/v1/projects/testproj/tasks", body2, nil)
	var taskB map[string]interface{}
	json.NewDecoder(rr2.Body).Decode(&taskB)

	// Add A -> B dependency
	depBody1 := map[string]interface{}{"parent_id": taskB["id"]}
	setup.doRequest("POST", fmt.Sprintf("/v1/projects/testproj/tasks/%s/deps", taskA["id"]), depBody1, nil)

	// Try to add B -> A dependency (would create cycle)
	depBody2 := map[string]interface{}{"parent_id": taskA["id"]}
	rr := setup.doRequest("POST", fmt.Sprintf("/v1/projects/testproj/tasks/%s/deps", taskB["id"]), depBody2, nil)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d: %s", rr.Code, rr.Body.String())
	}

	var resp response.ErrorResponse
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp.Error.Code != "CYCLE_DETECTED" {
		t.Errorf("expected code 'CYCLE_DETECTED', got %q", resp.Error.Code)
	}
}

func TestListDependencies(t *testing.T) {
	setup := newTestSetup(t)
	defer setup.cleanup()

	// Create tasks
	body1 := map[string]interface{}{"title": "Child Task"}
	rr1 := setup.doRequest("POST", "/v1/projects/testproj/tasks", body1, nil)
	var child map[string]interface{}
	json.NewDecoder(rr1.Body).Decode(&child)

	body2 := map[string]interface{}{"title": "Parent Task"}
	rr2 := setup.doRequest("POST", "/v1/projects/testproj/tasks", body2, nil)
	var parent map[string]interface{}
	json.NewDecoder(rr2.Body).Decode(&parent)

	// Add dependency
	depBody := map[string]interface{}{"parent_id": parent["id"]}
	setup.doRequest("POST", fmt.Sprintf("/v1/projects/testproj/tasks/%s/deps", child["id"]), depBody, nil)

	// List dependencies
	rr := setup.doRequest("GET", fmt.Sprintf("/v1/projects/testproj/tasks/%s/deps", child["id"]), nil, nil)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var deps []map[string]interface{}
	if err := json.NewDecoder(rr.Body).Decode(&deps); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if len(deps) != 1 {
		t.Errorf("expected 1 dependency, got %d", len(deps))
	}
}

func TestRemoveDependency(t *testing.T) {
	setup := newTestSetup(t)
	defer setup.cleanup()

	// Create tasks
	body1 := map[string]interface{}{"title": "Child Task"}
	rr1 := setup.doRequest("POST", "/v1/projects/testproj/tasks", body1, nil)
	var child map[string]interface{}
	json.NewDecoder(rr1.Body).Decode(&child)

	body2 := map[string]interface{}{"title": "Parent Task"}
	rr2 := setup.doRequest("POST", "/v1/projects/testproj/tasks", body2, nil)
	var parent map[string]interface{}
	json.NewDecoder(rr2.Body).Decode(&parent)

	// Add dependency
	depBody := map[string]interface{}{"parent_id": parent["id"]}
	setup.doRequest("POST", fmt.Sprintf("/v1/projects/testproj/tasks/%s/deps", child["id"]), depBody, nil)

	// Remove dependency
	rr := setup.doRequest("DELETE", fmt.Sprintf("/v1/projects/testproj/tasks/%s/deps/%s", child["id"], parent["id"]), nil, nil)

	if rr.Code != http.StatusNoContent {
		t.Errorf("expected status 204, got %d: %s", rr.Code, rr.Body.String())
	}

	// Verify it's gone
	listRR := setup.doRequest("GET", fmt.Sprintf("/v1/projects/testproj/tasks/%s/deps", child["id"]), nil, nil)
	var deps []map[string]interface{}
	json.NewDecoder(listRR.Body).Decode(&deps)
	if len(deps) != 0 {
		t.Errorf("expected 0 dependencies, got %d", len(deps))
	}
}

// ========================
// Audit Tests
// ========================

func TestTaskHistory(t *testing.T) {
	setup := newTestSetup(t)
	defer setup.cleanup()

	// Create a task
	createBody := map[string]interface{}{"title": "Task with history"}
	createRR := setup.doRequest("POST", "/v1/projects/testproj/tasks", createBody, nil)
	var created map[string]interface{}
	json.NewDecoder(createRR.Body).Decode(&created)
	taskID := created["id"].(string)

	// Get history
	rr := setup.doRequest("GET", fmt.Sprintf("/v1/projects/testproj/tasks/%s/history", taskID), nil, nil)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var entries []map[string]interface{}
	if err := json.NewDecoder(rr.Body).Decode(&entries); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	// Should have at least the create entry
	if len(entries) < 1 {
		t.Errorf("expected at least 1 audit entry, got %d", len(entries))
	}
}

func TestAuditQuery(t *testing.T) {
	setup := newTestSetup(t)
	defer setup.cleanup()

	// Create some tasks to generate audit entries
	body := map[string]interface{}{"title": "Audit test task"}
	setup.doRequest("POST", "/v1/projects/testproj/tasks", body, nil)

	// Query audit log
	rr := setup.doRequest("GET", "/v1/projects/testproj/audit", nil, nil)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var resp response.PaginatedResponse
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	entries := resp.Data.([]interface{})
	if len(entries) < 1 {
		t.Errorf("expected at least 1 audit entry, got %d", len(entries))
	}
}

// Unused imports that are needed for compilation
var _ = filepath.Base
var _ = sql.Open
var _ = context.Background
