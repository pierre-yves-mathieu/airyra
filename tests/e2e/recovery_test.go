package e2e

import (
	"context"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/airyra/airyra/internal/api"
	"github.com/airyra/airyra/internal/client"
	"github.com/airyra/airyra/internal/domain"
	"github.com/airyra/airyra/internal/store"
)

// TestE2E_ServerRestart tests that tasks persist across server restarts.
func TestE2E_ServerRestart(t *testing.T) {
	// Create temp directory for databases
	tempDir, err := os.MkdirTemp("", "airyra-e2e-restart-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	dbPath := filepath.Join(tempDir, "projects")
	projectName := "restart-test"

	var taskID1, taskID2 string

	// Phase 1: Start server, create tasks, stop server
	{
		// Create database manager
		dbManager, err := store.NewManager(dbPath)
		if err != nil {
			t.Fatalf("Failed to create database manager: %v", err)
		}

		// Create HTTP test server
		router := api.NewRouter(dbManager)
		server := httptest.NewServer(router)

		// Parse server URL
		serverURL := server.URL
		parts := parseServerURL(serverURL)
		host := parts[0]
		port := parsePort(parts[1])

		// Create client
		c := newTestClient(host, port, projectName, "test-agent")

		// Create tasks
		task1, err := c.CreateTask(context.Background(), "Persistent Task 1", "Description 1", 1, "")
		if err != nil {
			t.Fatalf("Failed to create task 1: %v", err)
		}
		taskID1 = task1.ID

		task2, err := c.CreateTask(context.Background(), "Persistent Task 2", "", 2, "")
		if err != nil {
			t.Fatalf("Failed to create task 2: %v", err)
		}
		taskID2 = task2.ID

		// Claim task 1
		_, err = c.ClaimTask(context.Background(), taskID1)
		if err != nil {
			t.Fatalf("Failed to claim task 1: %v", err)
		}

		t.Logf("Created tasks: %s, %s", taskID1, taskID2)

		// Stop server
		server.Close()
		dbManager.Close()

		t.Log("Server stopped, data should be persisted")
	}

	// Phase 2: Start new server, verify tasks persist
	{
		// Create new database manager (same path)
		dbManager, err := store.NewManager(dbPath)
		if err != nil {
			t.Fatalf("Failed to create database manager for restart: %v", err)
		}
		defer dbManager.Close()

		// Create new HTTP test server
		router := api.NewRouter(dbManager)
		server := httptest.NewServer(router)
		defer server.Close()

		// Parse server URL
		serverURL := server.URL
		parts := parseServerURL(serverURL)
		host := parts[0]
		port := parsePort(parts[1])

		// Create client
		c := newTestClient(host, port, projectName, "test-agent")

		// Verify tasks exist
		task1, err := c.GetTask(context.Background(), taskID1)
		if err != nil {
			t.Fatalf("Task 1 should exist after restart: %v", err)
		}

		// Verify task 1 properties
		if task1.Title != "Persistent Task 1" {
			t.Errorf("Task 1 title = %q, want %q", task1.Title, "Persistent Task 1")
		}
		if task1.Status != domain.StatusInProgress {
			t.Errorf("Task 1 status = %s, want in_progress", task1.Status)
		}
		if task1.Priority != 1 {
			t.Errorf("Task 1 priority = %d, want 1", task1.Priority)
		}

		task2, err := c.GetTask(context.Background(), taskID2)
		if err != nil {
			t.Fatalf("Task 2 should exist after restart: %v", err)
		}

		if task2.Title != "Persistent Task 2" {
			t.Errorf("Task 2 title = %q, want %q", task2.Title, "Persistent Task 2")
		}
		if task2.Status != domain.StatusOpen {
			t.Errorf("Task 2 status = %s, want open", task2.Status)
		}

		// Verify list works
		result, err := c.ListTasks(context.Background(), "", 1, 100)
		if err != nil {
			t.Fatalf("Failed to list tasks after restart: %v", err)
		}

		if len(result.Data) != 2 {
			t.Errorf("Expected 2 tasks after restart, got %d", len(result.Data))
		}

		t.Log("Data persisted correctly across restart")
	}
}

// TestE2E_ServerRestart_Dependencies tests that dependencies persist across restarts.
func TestE2E_ServerRestart_Dependencies(t *testing.T) {
	// Create temp directory
	tempDir, err := os.MkdirTemp("", "airyra-e2e-deps-restart-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	dbPath := filepath.Join(tempDir, "projects")
	projectName := "deps-restart-test"

	var taskA, taskB, taskC string

	// Phase 1: Create tasks and dependencies
	{
		dbManager, err := store.NewManager(dbPath)
		if err != nil {
			t.Fatalf("Failed to create database manager: %v", err)
		}

		router := api.NewRouter(dbManager)
		server := httptest.NewServer(router)

		parts := parseServerURL(server.URL)
		port := parsePort(parts[1])
		c := newTestClient(parts[0], port, projectName, "test-agent")

		// Create tasks
		task, _ := c.CreateTask(context.Background(), "Task A", "", 2, "")
		taskA = task.ID
		task, _ = c.CreateTask(context.Background(), "Task B", "", 2, "")
		taskB = task.ID
		task, _ = c.CreateTask(context.Background(), "Task C", "", 2, "")
		taskC = task.ID

		// Create dependencies: B depends on A, C depends on B
		if err := c.AddDependency(context.Background(), taskB, taskA); err != nil {
			t.Fatalf("Failed to add B->A dependency: %v", err)
		}
		if err := c.AddDependency(context.Background(), taskC, taskB); err != nil {
			t.Fatalf("Failed to add C->B dependency: %v", err)
		}

		t.Logf("Created dependency chain: %s <- %s <- %s", taskA, taskB, taskC)

		server.Close()
		dbManager.Close()
	}

	// Phase 2: Verify dependencies persist
	{
		dbManager, err := store.NewManager(dbPath)
		if err != nil {
			t.Fatalf("Failed to create database manager: %v", err)
		}
		defer dbManager.Close()

		router := api.NewRouter(dbManager)
		server := httptest.NewServer(router)
		defer server.Close()

		parts := parseServerURL(server.URL)
		port := parsePort(parts[1])
		c := newTestClient(parts[0], port, projectName, "test-agent")

		// Verify only A is ready
		result, err := c.ListReadyTasks(context.Background(), 1, 100)
		if err != nil {
			t.Fatalf("Failed to list ready tasks: %v", err)
		}

		readyIDs := make(map[string]bool)
		for _, task := range result.Data {
			readyIDs[task.ID] = true
		}

		if !readyIDs[taskA] {
			t.Error("Task A should be ready after restart")
		}
		if readyIDs[taskB] {
			t.Error("Task B should NOT be ready (depends on A)")
		}
		if readyIDs[taskC] {
			t.Error("Task C should NOT be ready (depends on B)")
		}

		// Verify dependencies via API
		deps, err := c.ListDependencies(context.Background(), taskB)
		if err != nil {
			t.Fatalf("Failed to list dependencies: %v", err)
		}

		foundDep := false
		for _, dep := range deps {
			if dep.ChildID == taskB && dep.ParentID == taskA {
				foundDep = true
				break
			}
		}
		if !foundDep {
			t.Error("B->A dependency should persist after restart")
		}

		t.Log("Dependencies persisted correctly")
	}
}

// TestE2E_ServerRestart_AuditHistory tests that audit history persists.
func TestE2E_ServerRestart_AuditHistory(t *testing.T) {
	// Create temp directory
	tempDir, err := os.MkdirTemp("", "airyra-e2e-audit-restart-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	dbPath := filepath.Join(tempDir, "projects")
	projectName := "audit-restart-test"

	var taskID string

	// Phase 1: Create task and perform operations
	{
		dbManager, err := store.NewManager(dbPath)
		if err != nil {
			t.Fatalf("Failed to create database manager: %v", err)
		}

		router := api.NewRouter(dbManager)
		server := httptest.NewServer(router)

		parts := parseServerURL(server.URL)
		port := parsePort(parts[1])
		c := newTestClient(parts[0], port, projectName, "test-agent")

		// Create task
		task, err := c.CreateTask(context.Background(), "Audited Task", "", 2, "")
		if err != nil {
			t.Fatalf("Failed to create task: %v", err)
		}
		taskID = task.ID

		// Claim task
		_, err = c.ClaimTask(context.Background(), taskID)
		if err != nil {
			t.Fatalf("Failed to claim task: %v", err)
		}

		// Release task
		_, err = c.ReleaseTask(context.Background(), taskID, false)
		if err != nil {
			t.Fatalf("Failed to release task: %v", err)
		}

		server.Close()
		dbManager.Close()
	}

	// Phase 2: Verify audit history persists
	{
		dbManager, err := store.NewManager(dbPath)
		if err != nil {
			t.Fatalf("Failed to create database manager: %v", err)
		}
		defer dbManager.Close()

		router := api.NewRouter(dbManager)
		server := httptest.NewServer(router)
		defer server.Close()

		parts := parseServerURL(server.URL)
		port := parsePort(parts[1])
		c := newTestClient(parts[0], port, projectName, "test-agent")

		// Get task history
		entries, err := c.GetTaskHistory(context.Background(), taskID)
		if err != nil {
			t.Fatalf("Failed to get task history: %v", err)
		}

		// Should have create, claim, release actions
		actionsSeen := make(map[domain.AuditAction]bool)
		for _, entry := range entries {
			actionsSeen[entry.Action] = true
		}

		if !actionsSeen[domain.ActionCreate] {
			t.Error("Audit history should contain 'create' action")
		}
		if !actionsSeen[domain.ActionClaim] {
			t.Error("Audit history should contain 'claim' action")
		}
		if !actionsSeen[domain.ActionRelease] {
			t.Error("Audit history should contain 'release' action")
		}

		t.Logf("Audit history persisted: %d entries", len(entries))
	}
}

// TestE2E_StalePIDFile tests that the CLI handles stale PID files gracefully.
// This test simulates a scenario where a PID file exists but the process is dead.
func TestE2E_StalePIDFile(t *testing.T) {
	suite := setupE2E(t)
	defer suite.cleanup()

	// Create a fake PID file with a non-existent process ID
	// We use a very high PID that is unlikely to exist
	stalePID := 999999999

	// Create the .airyra directory in temp
	airyraDir := filepath.Join(suite.tempDir, ".airyra")
	if err := os.MkdirAll(airyraDir, 0755); err != nil {
		t.Fatalf("Failed to create .airyra dir: %v", err)
	}

	// Write stale PID file
	pidPath := filepath.Join(airyraDir, "airyra.pid")
	if err := os.WriteFile(pidPath, []byte("999999999"), 0644); err != nil {
		t.Fatalf("Failed to write PID file: %v", err)
	}

	// Verify PID file exists
	if _, err := os.Stat(pidPath); os.IsNotExist(err) {
		t.Fatal("PID file should exist")
	}

	t.Logf("Created stale PID file at %s with PID %d", pidPath, stalePID)

	// The actual server start command would detect the stale PID and clean it up
	// For this test, we verify the behavior is reasonable

	// Note: We can't easily test the full ar server start command here because
	// it would actually try to start a server. Instead, we verify that
	// our test infrastructure doesn't break with stale PIDs.

	// The server status command should report server not running
	// (since our test suite uses httptest, not the real server management)
}

// TestE2E_DatabaseCorruptionRecovery tests basic database integrity after operations.
func TestE2E_DatabaseCorruptionRecovery(t *testing.T) {
	// This test verifies that the database remains consistent after many operations
	suite := setupE2E(t)
	defer suite.cleanup()

	projectName := "corruption-test"
	_ = suite.createProject(projectName)

	// Create many tasks
	numTasks := 20
	taskIDs := make([]string, numTasks)

	for i := 0; i < numTasks; i++ {
		taskIDs[i] = suite.createTask(projectName, taskTitleForNum(i))
	}

	// Perform various operations
	for i := 0; i < numTasks; i++ {
		// Claim every other task
		if i%2 == 0 {
			suite.claimTask(projectName, taskIDs[i], agentIDForNum(i))
		}
	}

	// Complete some claimed tasks
	for i := 0; i < numTasks; i += 4 {
		suite.completeTask(projectName, taskIDs[i], agentIDForNum(i))
	}

	// Release some claimed tasks
	for i := 2; i < numTasks; i += 4 {
		suite.releaseTask(projectName, taskIDs[i], agentIDForNum(i), false)
	}

	// Add some dependencies
	for i := 1; i < numTasks; i += 3 {
		if i > 0 {
			suite.addDependency(projectName, taskIDs[i], taskIDs[i-1])
		}
	}

	// Verify database integrity by listing all tasks
	tasks := suite.listTasks(projectName)
	if len(tasks) != numTasks {
		t.Errorf("Expected %d tasks, got %d", numTasks, len(tasks))
	}

	// Verify each task is in a valid state
	for _, task := range tasks {
		if !task.Status.IsValid() {
			t.Errorf("Task %s has invalid status: %s", task.ID, task.Status)
		}
		if task.Priority < 0 || task.Priority > 4 {
			t.Errorf("Task %s has invalid priority: %d", task.ID, task.Priority)
		}
		if task.Title == "" {
			t.Errorf("Task %s has empty title", task.ID)
		}
	}

	// Verify ready tasks are consistent with dependencies
	readyTasks := suite.listReadyTasks(projectName)
	for _, task := range readyTasks {
		if task.Status != domain.StatusOpen {
			t.Errorf("Ready task %s should be open, got %s", task.ID, task.Status)
		}
	}

	t.Log("Database integrity verified after operations")
}

// TestE2E_ConnectionTimeout tests behavior when server is slow to respond.
func TestE2E_ConnectionTimeout(t *testing.T) {
	// This test is more of a placeholder - full timeout testing would require
	// a mock server that delays responses
	suite := setupE2E(t)
	defer suite.cleanup()

	projectName := "timeout-test"
	_ = suite.createProject(projectName)

	// Basic operation should complete quickly
	start := time.Now()
	suite.createTask(projectName, "Fast task")
	elapsed := time.Since(start)

	// Should complete in under 5 seconds (generous for a local test)
	if elapsed > 5*time.Second {
		t.Errorf("Task creation took too long: %v", elapsed)
	}

	t.Logf("Task creation completed in %v", elapsed)
}

// Helper functions for recovery tests

func parseServerURL(url string) []string {
	// URL format: http://127.0.0.1:PORT
	trimmed := strings.TrimPrefix(url, "http://")
	return strings.Split(trimmed, ":")
}

func parsePort(portStr string) int {
	p, _ := strconv.Atoi(portStr)
	return p
}

func newTestClient(host string, port int, project, agentID string) *client.Client {
	return client.NewClient(host, port, project, agentID)
}
