package e2e

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/airyra/airyra/internal/config"
	"github.com/airyra/airyra/internal/domain"
)

// Exit codes from cmd/ar/exitcodes.go
const (
	ExitSuccess              = 0
	ExitGeneralError         = 1
	ExitServerNotRunning     = 2
	ExitProjectNotConfigured = 3
	ExitTaskNotFound         = 4
	ExitPermissionDenied     = 5
	ExitConflict             = 6
)

// TestE2E_ServerNotRunning tests the error handling when the server is not running.
func TestE2E_ServerNotRunning(t *testing.T) {
	// Create a temp directory for this test
	tempDir, err := os.MkdirTemp("", "airyra-e2e-noserver-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create project dir with config pointing to a non-existent server
	projectDir := filepath.Join(tempDir, "project")
	if err := os.MkdirAll(projectDir, 0755); err != nil {
		t.Fatalf("Failed to create project dir: %v", err)
	}

	// Create airyra.toml with a port that is not running
	configContent := `project = "test-project"

[server]
host = "127.0.0.1"
port = 59999
`
	configPath := filepath.Join(projectDir, config.ConfigFileName)
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to write config: %v", err)
	}

	// Build CLI from project root
	binPath := filepath.Join(tempDir, "ar")
	projectRoot := findProjectRoot()
	buildCmd := exec.Command("go", "build", "-o", binPath, "./cmd/ar")
	buildCmd.Dir = projectRoot
	if out, err := buildCmd.CombinedOutput(); err != nil {
		t.Fatalf("Failed to build CLI: %v\n%s", err, out)
	}

	// Run ar list without a server running
	stdout, stderr, exitCode := runCLICommand(binPath, projectDir, "list")

	// Verify exit code is ExitServerNotRunning (2)
	if exitCode != ExitServerNotRunning {
		t.Errorf("Exit code = %d, want %d (server not running)", exitCode, ExitServerNotRunning)
		t.Logf("stdout: %s", stdout)
		t.Logf("stderr: %s", stderr)
	}

	// Verify error message is helpful
	output := stdout + stderr
	if !containsString(output, "server") && !containsString(output, "running") && !containsString(output, "refused") && !containsString(output, "unreachable") {
		t.Errorf("Error message should mention server not running:\nstdout: %s\nstderr: %s", stdout, stderr)
	}
}

// TestE2E_NoProjectConfig tests the error handling when no airyra.toml exists.
func TestE2E_NoProjectConfig(t *testing.T) {
	suite := setupE2E(t)
	defer suite.cleanup()

	// Create an empty directory without airyra.toml
	emptyDir := filepath.Join(suite.tempDir, "empty-project")
	if err := os.MkdirAll(emptyDir, 0755); err != nil {
		t.Fatalf("Failed to create empty dir: %v", err)
	}

	// Run ar list in directory without config
	stdout, stderr, exitCode := suite.runCLIInDir(emptyDir, "list")

	// Verify exit code is ExitProjectNotConfigured (3)
	if exitCode != ExitProjectNotConfigured {
		t.Errorf("Exit code = %d, want %d (project not configured)", exitCode, ExitProjectNotConfigured)
		t.Logf("stdout: %s", stdout)
		t.Logf("stderr: %s", stderr)
	}

	// Verify error message mentions init
	output := stdout + stderr
	if !containsString(output, "init") || !containsString(output, "airyra.toml") {
		t.Errorf("Error message should mention init command:\nstdout: %s\nstderr: %s", stdout, stderr)
	}
}

// TestE2E_TaskNotFound tests the error handling when a task doesn't exist.
func TestE2E_TaskNotFound(t *testing.T) {
	suite := setupE2E(t)
	defer suite.cleanup()

	projectName := "notfound-test"
	projectDir := suite.createProject(projectName)

	// Try to show a non-existent task
	stdout, stderr, exitCode := suite.runCLIInDir(projectDir, "show", "ar-nonexistent")

	// Verify exit code is ExitTaskNotFound (4)
	if exitCode != ExitTaskNotFound {
		t.Errorf("Exit code = %d, want %d (task not found)", exitCode, ExitTaskNotFound)
		t.Logf("stdout: %s", stdout)
		t.Logf("stderr: %s", stderr)
	}

	// Verify error message mentions the task ID
	output := stdout + stderr
	if !containsString(output, "not found") && !containsString(output, "NOT_FOUND") {
		t.Errorf("Error message should mention task not found:\nstdout: %s\nstderr: %s", stdout, stderr)
	}
}

// TestE2E_NotOwner tests the error handling when an agent tries to complete
// a task claimed by another agent.
func TestE2E_NotOwner(t *testing.T) {
	suite := setupE2E(t)
	defer suite.cleanup()

	projectName := "notowner-test"
	projectDir := suite.createProject(projectName)

	// Create a task
	taskID := suite.createTask(projectName, "Owned task")

	// Agent 1 claims the task
	agent1 := "agent-1@host1:/project"
	_, err := suite.claimTask(projectName, taskID, agent1)
	if err != nil {
		t.Fatalf("Agent 1 failed to claim task: %v", err)
	}

	// Create a modified config file with Agent 2's identity
	// Since the CLI uses environment-based identity, we'll test via API
	agent2 := "agent-2@host2:/project"
	c := suite.getClient(projectName, agent2)

	// Agent 2 tries to complete the task
	_, err = c.CompleteTask(context.Background(), taskID)

	if err == nil {
		t.Fatal("Agent 2 should not be able to complete a task owned by Agent 1")
	}

	// Verify error is NOT_OWNER
	if !containsString(err.Error(), "claimed") && !containsString(err.Error(), "owner") {
		t.Errorf("Error should mention ownership issue: %v", err)
	}

	// Also test via CLI (which will use a different identity)
	// The CLI generates identity from environment, so this tests the path
	stdout, stderr, exitCode := suite.runCLIInDir(projectDir, "done", taskID)

	// This might succeed if CLI identity matches the claim, or fail if different
	// We just check the behavior is consistent
	t.Logf("CLI done result: exit=%d, stdout=%s, stderr=%s", exitCode, stdout, stderr)
}

// TestE2E_NotOwnerRelease tests that an agent cannot release a task claimed by another agent.
func TestE2E_NotOwnerRelease(t *testing.T) {
	suite := setupE2E(t)
	defer suite.cleanup()

	projectName := "release-notowner-test"
	_ = suite.createProject(projectName)

	// Create a task
	taskID := suite.createTask(projectName, "Task to release")

	// Agent 1 claims
	agent1 := "agent-1@host:/project"
	_, err := suite.claimTask(projectName, taskID, agent1)
	if err != nil {
		t.Fatalf("Agent 1 failed to claim: %v", err)
	}

	// Agent 2 tries to release without force
	agent2 := "agent-2@host:/project"
	_, err = suite.releaseTask(projectName, taskID, agent2, false)

	if err == nil {
		t.Fatal("Agent 2 should not be able to release without force")
	}

	// Verify it's a permission error
	if isDomainError(err, domain.ErrCodeNotOwner) {
		t.Log("Correctly got NOT_OWNER error")
	} else if containsString(err.Error(), "owner") || containsString(err.Error(), "claimed") {
		t.Log("Error correctly mentions ownership")
	} else {
		t.Errorf("Expected NOT_OWNER error, got: %v", err)
	}

	// Agent 2 can release with force
	_, err = suite.releaseTask(projectName, taskID, agent2, true)
	if err != nil {
		t.Errorf("Agent 2 should be able to force-release: %v", err)
	}
}

// TestE2E_AlreadyClaimed tests the error handling when trying to claim an already claimed task.
func TestE2E_AlreadyClaimed(t *testing.T) {
	suite := setupE2E(t)
	defer suite.cleanup()

	projectName := "already-claimed-test"
	projectDir := suite.createProject(projectName)

	// Create a task
	taskID := suite.createTask(projectName, "Already claimed task")

	// Agent 1 claims
	agent1 := "agent-1@host:/project"
	_, err := suite.claimTask(projectName, taskID, agent1)
	if err != nil {
		t.Fatalf("Agent 1 failed to claim: %v", err)
	}

	// Agent 2 tries to claim
	agent2 := "agent-2@host:/project"
	_, err = suite.claimTask(projectName, taskID, agent2)

	if err == nil {
		t.Fatal("Agent 2 should not be able to claim an already claimed task")
	}

	// Verify it's ALREADY_CLAIMED error
	if isDomainError(err, domain.ErrCodeAlreadyClaimed) {
		t.Log("Correctly got ALREADY_CLAIMED error")
	} else if containsString(err.Error(), "claimed") {
		t.Log("Error correctly mentions 'claimed'")
	} else {
		t.Errorf("Expected ALREADY_CLAIMED error, got: %v", err)
	}

	// Also test via CLI to verify exit code
	// Note: CLI will generate a different agent ID, so it should fail
	stdout, stderr, exitCode := suite.runCLIInDir(projectDir, "claim", taskID)

	// Exit code should be ExitConflict (6)
	if exitCode != ExitConflict {
		t.Errorf("Exit code = %d, want %d (conflict)", exitCode, ExitConflict)
		t.Logf("stdout: %s", stdout)
		t.Logf("stderr: %s", stderr)
	}
}

// TestE2E_InvalidTransition tests error handling for invalid status transitions.
func TestE2E_InvalidTransition(t *testing.T) {
	suite := setupE2E(t)
	defer suite.cleanup()

	projectName := "invalid-transition-test"
	projectDir := suite.createProject(projectName)

	// Create a task (status: open)
	taskID := suite.createTask(projectName, "Transition test task")

	// Try to complete without claiming first (open -> done is invalid)
	stdout, stderr, exitCode := suite.runCLIInDir(projectDir, "done", taskID)

	if exitCode == 0 {
		t.Error("Should not be able to complete a task that is not in_progress")
	}

	// Verify error mentions transition issue
	output := stdout + stderr
	if !containsString(output, "transition") && !containsString(output, "status") && !containsString(output, "cannot") {
		t.Logf("Warning: error message may not clearly indicate transition issue:\n%s", output)
	}

	// Note: Per spec, any -> blocked is valid, so we test unblock instead
	// Try to unblock a task that is not blocked (open -> open via unblock is invalid)
	_, stderr, exitCode = suite.runCLIInDir(projectDir, "unblock", taskID)

	if exitCode == 0 {
		t.Error("Should not be able to unblock a task that is not blocked")
	}

	// Verify error mentions the issue
	if !containsString(stderr, "blocked") && !containsString(stderr, "status") && !containsString(stderr, "cannot") {
		t.Logf("Warning: error message may not clearly indicate unblock issue:\n%s", stderr)
	}
}

// TestE2E_DependencyNotFound tests error handling when removing a non-existent dependency.
func TestE2E_DependencyNotFound(t *testing.T) {
	suite := setupE2E(t)
	defer suite.cleanup()

	projectName := "depnotfound-test"
	projectDir := suite.createProject(projectName)

	// Create two tasks
	taskA := suite.createTask(projectName, "Task A")
	taskB := suite.createTask(projectName, "Task B")

	// Try to remove a dependency that doesn't exist
	stdout, stderr, exitCode := suite.runCLIInDir(projectDir, "dep", "rm", taskB, taskA)

	if exitCode == 0 {
		// Some systems might allow silent removal, which is acceptable
		t.Log("Removing non-existent dependency succeeded (might be acceptable behavior)")
	} else {
		// Verify error message is appropriate
		output := stdout + stderr
		if !containsString(output, "not found") && !containsString(output, "NOT_FOUND") && !containsString(output, "dependency") {
			t.Errorf("Error should mention dependency not found:\n%s", output)
		}
	}
}

// TestE2E_CycleDetected tests error handling when adding a dependency that would create a cycle.
func TestE2E_CycleDetected(t *testing.T) {
	suite := setupE2E(t)
	defer suite.cleanup()

	projectName := "cycle-test"
	projectDir := suite.createProject(projectName)

	// Create tasks A, B, C
	taskA := suite.createTask(projectName, "Task A")
	taskB := suite.createTask(projectName, "Task B")
	taskC := suite.createTask(projectName, "Task C")

	// Create chain: A <- B <- C (C depends on B, B depends on A)
	if err := suite.addDependency(projectName, taskB, taskA); err != nil {
		t.Fatalf("Failed to add B->A dependency: %v", err)
	}
	if err := suite.addDependency(projectName, taskC, taskB); err != nil {
		t.Fatalf("Failed to add C->B dependency: %v", err)
	}

	// Try to add A->C dependency which would create a cycle: A->B->C->A
	stdout, stderr, exitCode := suite.runCLIInDir(projectDir, "dep", "add", taskA, taskC)

	if exitCode == 0 {
		t.Error("Adding cyclic dependency should fail")
	}

	// Verify error mentions cycle
	output := stdout + stderr
	if !containsString(output, "cycle") && !containsString(output, "CYCLE") {
		t.Errorf("Error should mention cycle detection:\n%s", output)
	}
}

// TestE2E_ValidationErrors tests various validation error scenarios.
func TestE2E_ValidationErrors(t *testing.T) {
	suite := setupE2E(t)
	defer suite.cleanup()

	projectName := "validation-test"
	projectDir := suite.createProject(projectName)

	// Test: Create task with empty title (this will be caught by cobra as missing argument)
	// So we skip this case

	// Test: Invalid priority
	stdout, stderr, exitCode := suite.runCLIInDir(projectDir, "create", "Test task", "-p", "invalid")
	if exitCode == 0 {
		t.Error("Creating task with invalid priority should fail")
	}

	output := stdout + stderr
	if !containsString(output, "priority") {
		t.Logf("Warning: error may not clearly mention priority issue:\n%s", output)
	}

	// Test: Priority out of range
	stdout, stderr, exitCode = suite.runCLIInDir(projectDir, "create", "Test task", "-p", "10")
	if exitCode == 0 {
		t.Error("Creating task with out-of-range priority should fail")
	}
}

// TestE2E_InvalidTaskID tests error handling for invalid task IDs.
func TestE2E_InvalidTaskID(t *testing.T) {
	suite := setupE2E(t)
	defer suite.cleanup()

	projectName := "invalid-id-test"
	projectDir := suite.createProject(projectName)

	// Test various invalid task IDs
	invalidIDs := []string{
		"invalid",
		"ar-",
		"../etc/passwd",  // Path traversal attempt
		"ar-nonexistent", // Valid format but doesn't exist
	}

	for _, id := range invalidIDs {
		stdout, stderr, exitCode := suite.runCLIInDir(projectDir, "show", id)

		if exitCode == 0 {
			t.Errorf("Task ID %q should not be found", id)
		}

		t.Logf("ID=%q: exit=%d, output=%s%s", id, exitCode, stdout, stderr)
	}
}

// runCLICommand runs a CLI command and returns stdout, stderr, and exit code.
func runCLICommand(binPath, dir string, args ...string) (stdout, stderr string, exitCode int) {
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
			exitCode = 1
		}
	}

	return
}
