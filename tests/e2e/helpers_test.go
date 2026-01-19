package e2e

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/airyra/airyra/internal/api"
	"github.com/airyra/airyra/internal/client"
	"github.com/airyra/airyra/internal/config"
	"github.com/airyra/airyra/internal/domain"
	"github.com/airyra/airyra/internal/store"
)

// E2ETestSuite provides test infrastructure for end-to-end tests.
type E2ETestSuite struct {
	t          *testing.T
	server     *httptest.Server
	dbManager  *store.Manager
	tempDir    string
	projectDir string
	host       string
	port       string
}

// setupE2E creates a new E2E test suite with a running server and clean state.
func setupE2E(t *testing.T) *E2ETestSuite {
	t.Helper()

	// Create temp directory for test databases
	tempDir, err := os.MkdirTemp("", "airyra-e2e-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	// Create database manager using temp directory
	dbPath := filepath.Join(tempDir, "projects")
	dbManager, err := store.NewManager(dbPath)
	if err != nil {
		os.RemoveAll(tempDir)
		t.Fatalf("Failed to create database manager: %v", err)
	}

	// Create HTTP test server with the API router
	router := api.NewRouter(dbManager)
	server := httptest.NewServer(router)

	// Parse server URL to get host and port
	// URL format is http://127.0.0.1:PORT
	serverURL := server.URL
	parts := strings.Split(strings.TrimPrefix(serverURL, "http://"), ":")
	host := parts[0]
	port := parts[1]

	// Create project directory for testing CLI
	projectDir := filepath.Join(tempDir, "project")
	if err := os.MkdirAll(projectDir, 0755); err != nil {
		server.Close()
		dbManager.Close()
		os.RemoveAll(tempDir)
		t.Fatalf("Failed to create project dir: %v", err)
	}

	return &E2ETestSuite{
		t:          t,
		server:     server,
		dbManager:  dbManager,
		tempDir:    tempDir,
		projectDir: projectDir,
		host:       host,
		port:       port,
	}
}

// cleanup tears down the test suite and cleans up resources.
func (s *E2ETestSuite) cleanup() {
	if s.server != nil {
		s.server.Close()
	}
	if s.dbManager != nil {
		s.dbManager.Close()
	}
	if s.tempDir != "" {
		os.RemoveAll(s.tempDir)
	}
}

// createProject creates a project configuration file and returns the project directory.
func (s *E2ETestSuite) createProject(name string) string {
	s.t.Helper()

	// Create a new project directory
	projectDir := filepath.Join(s.tempDir, "projects", name)
	if err := os.MkdirAll(projectDir, 0755); err != nil {
		s.t.Fatalf("Failed to create project directory: %v", err)
	}

	// Create airyra.toml config file
	configContent := fmt.Sprintf(`project = %q

[server]
host = %q
port = %s
`, name, s.host, s.port)

	configPath := filepath.Join(projectDir, config.ConfigFileName)
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		s.t.Fatalf("Failed to write config file: %v", err)
	}

	return projectDir
}

// runCLI executes the ar CLI command with the given arguments and returns stdout, stderr, and exit code.
// The command is run in the context of a project directory.
func (s *E2ETestSuite) runCLI(args ...string) (stdout, stderr string, exitCode int) {
	return s.runCLIInDir(s.projectDir, args...)
}

// runCLIInDir executes the ar CLI command in the specified directory.
func (s *E2ETestSuite) runCLIInDir(dir string, args ...string) (stdout, stderr string, exitCode int) {
	s.t.Helper()

	// Build the CLI binary if needed
	binPath := s.buildCLI()

	// Execute the command
	cmd := exec.Command(binPath, args...)
	cmd.Dir = dir

	var stdoutBuf, stderrBuf bytes.Buffer
	cmd.Stdout = &stdoutBuf
	cmd.Stderr = &stderrBuf

	err := cmd.Run()
	stdout = stdoutBuf.String()
	stderr = stderrBuf.String()

	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			s.t.Fatalf("Failed to execute CLI: %v", err)
		}
	}

	return stdout, stderr, exitCode
}

// buildCLI compiles the ar CLI binary and returns its path.
// It caches the binary to avoid rebuilding for each test.
func (s *E2ETestSuite) buildCLI() string {
	s.t.Helper()

	binPath := filepath.Join(s.tempDir, "ar")

	// Check if binary already exists
	if _, err := os.Stat(binPath); err == nil {
		return binPath
	}

	// Find project root by looking for go.mod
	projectRoot := findProjectRoot()

	// Build the CLI from the project root
	cmd := exec.Command("go", "build", "-o", binPath, "./cmd/ar")
	cmd.Dir = projectRoot

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		s.t.Fatalf("Failed to build CLI: %v\nstderr: %s", err, stderr.String())
	}

	return binPath
}

// findProjectRoot finds the project root by looking for go.mod
func findProjectRoot() string {
	// Start from the current working directory
	dir, err := os.Getwd()
	if err != nil {
		// Fallback: use relative path from test file location
		return filepath.Join("..", "..")
	}

	// Walk up until we find go.mod
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			// Reached root without finding go.mod
			return filepath.Join("..", "..")
		}
		dir = parent
	}
}

// createTask creates a task using the HTTP API directly and returns the task ID.
func (s *E2ETestSuite) createTask(projectName, title string) string {
	s.t.Helper()

	c := s.getClient(projectName, "test-agent")
	task, err := c.CreateTask(context.Background(), title, "", 2, "")
	if err != nil {
		s.t.Fatalf("Failed to create task: %v", err)
	}

	return task.ID
}

// createTaskWithPriority creates a task with a specific priority.
func (s *E2ETestSuite) createTaskWithPriority(projectName, title string, priority int) string {
	s.t.Helper()

	c := s.getClient(projectName, "test-agent")
	task, err := c.CreateTask(context.Background(), title, "", priority, "")
	if err != nil {
		s.t.Fatalf("Failed to create task: %v", err)
	}

	return task.ID
}

// getClient creates an API client for the test server.
func (s *E2ETestSuite) getClient(projectName, agentID string) *client.Client {
	s.t.Helper()

	// Parse port as int
	var port int
	fmt.Sscanf(s.port, "%d", &port)

	return client.NewClient(s.host, port, projectName, agentID)
}

// getTask retrieves a task by ID using the HTTP API.
func (s *E2ETestSuite) getTask(projectName, taskID string) *domain.Task {
	s.t.Helper()

	c := s.getClient(projectName, "test-agent")
	task, err := c.GetTask(context.Background(), taskID)
	if err != nil {
		s.t.Fatalf("Failed to get task: %v", err)
	}

	return task
}

// claimTask claims a task using the HTTP API.
func (s *E2ETestSuite) claimTask(projectName, taskID, agentID string) (*domain.Task, error) {
	c := s.getClient(projectName, agentID)
	return c.ClaimTask(context.Background(), taskID)
}

// completeTask marks a task as done using the HTTP API.
func (s *E2ETestSuite) completeTask(projectName, taskID, agentID string) (*domain.Task, error) {
	c := s.getClient(projectName, agentID)
	return c.CompleteTask(context.Background(), taskID)
}

// releaseTask releases a claimed task using the HTTP API.
func (s *E2ETestSuite) releaseTask(projectName, taskID, agentID string, force bool) (*domain.Task, error) {
	c := s.getClient(projectName, agentID)
	return c.ReleaseTask(context.Background(), taskID, force)
}

// addDependency adds a dependency between two tasks.
func (s *E2ETestSuite) addDependency(projectName, childID, parentID string) error {
	c := s.getClient(projectName, "test-agent")
	return c.AddDependency(context.Background(), childID, parentID)
}

// listReadyTasks returns tasks that are ready to be worked on.
func (s *E2ETestSuite) listReadyTasks(projectName string) []*domain.Task {
	s.t.Helper()

	c := s.getClient(projectName, "test-agent")
	result, err := c.ListReadyTasks(context.Background(), 1, 100)
	if err != nil {
		s.t.Fatalf("Failed to list ready tasks: %v", err)
	}

	return result.Data
}

// listTasks returns all tasks for a project.
func (s *E2ETestSuite) listTasks(projectName string) []*domain.Task {
	s.t.Helper()

	c := s.getClient(projectName, "test-agent")
	result, err := c.ListTasks(context.Background(), "", 1, 100)
	if err != nil {
		s.t.Fatalf("Failed to list tasks: %v", err)
	}

	return result.Data
}

// getTaskHistory retrieves audit history for a task.
func (s *E2ETestSuite) getTaskHistory(projectName, taskID string) []domain.AuditEntry {
	s.t.Helper()

	c := s.getClient(projectName, "test-agent")
	entries, err := c.GetTaskHistory(context.Background(), taskID)
	if err != nil {
		s.t.Fatalf("Failed to get task history: %v", err)
	}

	return entries
}

// waitForCondition waits until a condition is met or times out.
func (s *E2ETestSuite) waitForCondition(timeout time.Duration, condition func() bool) bool {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if condition() {
			return true
		}
		time.Sleep(10 * time.Millisecond)
	}
	return false
}

// assertTaskStatus verifies that a task has the expected status.
func (s *E2ETestSuite) assertTaskStatus(projectName, taskID string, expected domain.TaskStatus) {
	s.t.Helper()

	task := s.getTask(projectName, taskID)
	if task.Status != expected {
		s.t.Errorf("Task %s status = %s, want %s", taskID, task.Status, expected)
	}
}

// parseJSONOutput parses JSON output from the CLI into a map.
func parseJSONOutput(output string) (map[string]interface{}, error) {
	var result map[string]interface{}
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		return nil, err
	}
	return result, nil
}

// parseJSONArray parses JSON array output from the CLI.
func parseJSONArray(output string) ([]interface{}, error) {
	var result []interface{}
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		return nil, err
	}
	return result, nil
}

// containsString checks if a string contains a substring.
func containsString(s, substr string) bool {
	return strings.Contains(s, substr)
}

// extractTaskIDFromOutput extracts a task ID from CLI output.
// Assumes output contains "ID: ar-xxxxx" format.
func extractTaskIDFromOutput(output string) string {
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		if strings.HasPrefix(strings.TrimSpace(line), "ID:") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				return strings.TrimSpace(parts[1])
			}
		}
	}
	return ""
}

// extractTaskIDFromJSON extracts task ID from JSON output.
func extractTaskIDFromJSON(output string) string {
	var task map[string]interface{}
	if err := json.Unmarshal([]byte(output), &task); err != nil {
		return ""
	}
	if id, ok := task["id"].(string); ok {
		return id
	}
	return ""
}
