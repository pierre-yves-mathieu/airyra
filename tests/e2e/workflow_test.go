package e2e

import (
	"context"
	"strings"
	"testing"

	"github.com/airyra/airyra/internal/domain"
)

// TestE2E_FullTaskLifecycle tests the complete task lifecycle through the API:
// 1. Create a project and task
// 2. List tasks (verify task appears)
// 3. Show task details
// 4. Claim the task (verify in_progress)
// 5. Mark task as done (verify done)
// 6. View task history (verify all actions logged)
func TestE2E_FullTaskLifecycle(t *testing.T) {
	suite := setupE2E(t)
	defer suite.cleanup()

	projectName := "lifecycle-test"
	projectDir := suite.createProject(projectName)

	// Step 1: Create a task via CLI
	stdout, stderr, exitCode := suite.runCLIInDir(projectDir, "create", "Build feature", "-d", "Implement the new feature")
	if exitCode != 0 {
		t.Fatalf("Failed to create task: exit=%d, stdout=%s, stderr=%s", exitCode, stdout, stderr)
	}

	taskID := extractTaskIDFromOutput(stdout)
	if taskID == "" {
		t.Fatalf("Could not extract task ID from output: %s", stdout)
	}
	t.Logf("Created task: %s", taskID)

	// Step 2: List tasks and verify the task appears
	stdout, stderr, exitCode = suite.runCLIInDir(projectDir, "list")
	if exitCode != 0 {
		t.Fatalf("Failed to list tasks: exit=%d, stderr=%s", exitCode, stderr)
	}

	if !containsString(stdout, taskID) {
		t.Errorf("Task %s not found in list output:\n%s", taskID, stdout)
	}
	if !containsString(stdout, "Build feature") {
		t.Errorf("Task title not found in list output:\n%s", stdout)
	}

	// Step 3: Show task details
	stdout, stderr, exitCode = suite.runCLIInDir(projectDir, "show", taskID)
	if exitCode != 0 {
		t.Fatalf("Failed to show task: exit=%d, stderr=%s", exitCode, stderr)
	}

	if !containsString(stdout, taskID) {
		t.Errorf("Task ID not in show output:\n%s", stdout)
	}
	if !containsString(stdout, "Build feature") {
		t.Errorf("Task title not in show output:\n%s", stdout)
	}
	if !containsString(stdout, "open") {
		t.Errorf("Task status should be 'open':\n%s", stdout)
	}

	// Step 4: Claim the task
	stdout, stderr, exitCode = suite.runCLIInDir(projectDir, "claim", taskID)
	if exitCode != 0 {
		t.Fatalf("Failed to claim task: exit=%d, stderr=%s", exitCode, stderr)
	}

	// Verify task is now in_progress
	task := suite.getTask(projectName, taskID)
	if task.Status != domain.StatusInProgress {
		t.Errorf("Task status after claim = %s, want in_progress", task.Status)
	}
	if task.ClaimedBy == nil {
		t.Error("Task should have ClaimedBy set after claim")
	}

	// Step 5: Mark task as done
	stdout, stderr, exitCode = suite.runCLIInDir(projectDir, "done", taskID)
	if exitCode != 0 {
		t.Fatalf("Failed to complete task: exit=%d, stderr=%s", exitCode, stderr)
	}

	// Verify task is now done
	task = suite.getTask(projectName, taskID)
	if task.Status != domain.StatusDone {
		t.Errorf("Task status after done = %s, want done", task.Status)
	}

	// Step 6: View task history
	stdout, stderr, exitCode = suite.runCLIInDir(projectDir, "history", taskID)
	if exitCode != 0 {
		t.Fatalf("Failed to get task history: exit=%d, stderr=%s", exitCode, stderr)
	}

	// Verify history contains create, claim actions
	if !containsString(stdout, "create") {
		t.Errorf("History should contain 'create' action:\n%s", stdout)
	}
	if !containsString(stdout, "claim") {
		t.Errorf("History should contain 'claim' action:\n%s", stdout)
	}

	// Verify via API that all actions are logged
	entries := suite.getTaskHistory(projectName, taskID)
	if len(entries) < 2 {
		t.Errorf("Expected at least 2 history entries (create, claim), got %d", len(entries))
	}

	actionsSeen := make(map[domain.AuditAction]bool)
	for _, entry := range entries {
		actionsSeen[entry.Action] = true
	}

	if !actionsSeen[domain.ActionCreate] {
		t.Error("History should include 'create' action")
	}
	if !actionsSeen[domain.ActionClaim] {
		t.Error("History should include 'claim' action")
	}
}

// TestE2E_DependencyWorkflow tests the dependency management workflow:
// 1. Create tasks A, B, C
// 2. B depends on A
// 3. C depends on B
// 4. Only A should be ready initially
// 5. Complete A -> B becomes ready
// 6. Complete B -> C becomes ready
func TestE2E_DependencyWorkflow(t *testing.T) {
	suite := setupE2E(t)
	defer suite.cleanup()

	projectName := "deps-test"
	projectDir := suite.createProject(projectName)

	// Create three tasks via CLI
	stdout, _, exitCode := suite.runCLIInDir(projectDir, "create", "Task A - Foundation")
	if exitCode != 0 {
		t.Fatalf("Failed to create Task A: %s", stdout)
	}
	taskA := extractTaskIDFromOutput(stdout)

	stdout, _, exitCode = suite.runCLIInDir(projectDir, "create", "Task B - Build on A")
	if exitCode != 0 {
		t.Fatalf("Failed to create Task B")
	}
	taskB := extractTaskIDFromOutput(stdout)

	stdout, _, exitCode = suite.runCLIInDir(projectDir, "create", "Task C - Final step")
	if exitCode != 0 {
		t.Fatalf("Failed to create Task C")
	}
	taskC := extractTaskIDFromOutput(stdout)

	t.Logf("Created tasks: A=%s, B=%s, C=%s", taskA, taskB, taskC)

	// Add dependencies: B depends on A
	stdout, stderr, exitCode := suite.runCLIInDir(projectDir, "dep", "add", taskB, taskA)
	if exitCode != 0 {
		t.Fatalf("Failed to add dependency B->A: exit=%d, stderr=%s", exitCode, stderr)
	}

	// Add dependencies: C depends on B
	stdout, stderr, exitCode = suite.runCLIInDir(projectDir, "dep", "add", taskC, taskB)
	if exitCode != 0 {
		t.Fatalf("Failed to add dependency C->B: exit=%d, stderr=%s", exitCode, stderr)
	}

	// Check ready tasks - only A should be ready
	readyTasks := suite.listReadyTasks(projectName)
	readyIDs := make(map[string]bool)
	for _, task := range readyTasks {
		readyIDs[task.ID] = true
	}

	if !readyIDs[taskA] {
		t.Errorf("Task A should be ready, but it's not in ready list")
	}
	if readyIDs[taskB] {
		t.Errorf("Task B should NOT be ready (depends on A)")
	}
	if readyIDs[taskC] {
		t.Errorf("Task C should NOT be ready (depends on B)")
	}

	// Verify via CLI
	stdout, _, exitCode = suite.runCLIInDir(projectDir, "ready")
	if exitCode != 0 {
		t.Fatalf("Failed to list ready tasks")
	}

	if !containsString(stdout, "Task A") {
		t.Errorf("Ready list should contain Task A:\n%s", stdout)
	}

	// Claim and complete Task A
	_, err := suite.claimTask(projectName, taskA, "agent-1")
	if err != nil {
		t.Fatalf("Failed to claim Task A: %v", err)
	}

	_, err = suite.completeTask(projectName, taskA, "agent-1")
	if err != nil {
		t.Fatalf("Failed to complete Task A: %v", err)
	}

	// Now B should be ready, but not C
	readyTasks = suite.listReadyTasks(projectName)
	readyIDs = make(map[string]bool)
	for _, task := range readyTasks {
		readyIDs[task.ID] = true
	}

	if !readyIDs[taskB] {
		t.Errorf("Task B should be ready after A is done")
	}
	if readyIDs[taskC] {
		t.Errorf("Task C should NOT be ready (depends on B which is not done)")
	}

	// Claim and complete Task B
	_, err = suite.claimTask(projectName, taskB, "agent-1")
	if err != nil {
		t.Fatalf("Failed to claim Task B: %v", err)
	}

	_, err = suite.completeTask(projectName, taskB, "agent-1")
	if err != nil {
		t.Fatalf("Failed to complete Task B: %v", err)
	}

	// Now C should be ready
	readyTasks = suite.listReadyTasks(projectName)
	readyIDs = make(map[string]bool)
	for _, task := range readyTasks {
		readyIDs[task.ID] = true
	}

	if !readyIDs[taskC] {
		t.Errorf("Task C should be ready after B is done")
	}
}

// TestE2E_MultiAgent tests multi-agent claim behavior:
// 1. Create a task
// 2. Agent 1 claims (success)
// 3. Agent 2 tries to claim (fails with ALREADY_CLAIMED)
// 4. Agent 1 releases
// 5. Agent 2 claims (success)
func TestE2E_MultiAgent(t *testing.T) {
	suite := setupE2E(t)
	defer suite.cleanup()

	projectName := "multi-agent-test"
	_ = suite.createProject(projectName)

	// Create a task
	taskID := suite.createTask(projectName, "Shared task")
	t.Logf("Created task: %s", taskID)

	agent1 := "agent-1@host1:/project"
	agent2 := "agent-2@host2:/project"

	// Agent 1 claims task
	task, err := suite.claimTask(projectName, taskID, agent1)
	if err != nil {
		t.Fatalf("Agent 1 failed to claim task: %v", err)
	}

	if task.Status != domain.StatusInProgress {
		t.Errorf("Task status after Agent 1 claim = %s, want in_progress", task.Status)
	}
	if task.ClaimedBy == nil || *task.ClaimedBy != agent1 {
		t.Errorf("Task should be claimed by Agent 1")
	}

	// Agent 2 tries to claim - should fail
	_, err = suite.claimTask(projectName, taskID, agent2)
	if err == nil {
		t.Fatal("Agent 2 should not be able to claim a task already claimed by Agent 1")
	}

	// Verify error is ALREADY_CLAIMED
	errMsg := err.Error()
	if !containsString(errMsg, "claimed") {
		t.Errorf("Error should mention 'claimed': %v", err)
	}

	// Agent 1 releases task
	task, err = suite.releaseTask(projectName, taskID, agent1, false)
	if err != nil {
		t.Fatalf("Agent 1 failed to release task: %v", err)
	}

	if task.Status != domain.StatusOpen {
		t.Errorf("Task status after release = %s, want open", task.Status)
	}
	if task.ClaimedBy != nil {
		t.Errorf("Task should not have ClaimedBy after release")
	}

	// Agent 2 claims task - should succeed now
	task, err = suite.claimTask(projectName, taskID, agent2)
	if err != nil {
		t.Fatalf("Agent 2 should be able to claim after release: %v", err)
	}

	if task.Status != domain.StatusInProgress {
		t.Errorf("Task status after Agent 2 claim = %s, want in_progress", task.Status)
	}
	if task.ClaimedBy == nil || *task.ClaimedBy != agent2 {
		t.Errorf("Task should be claimed by Agent 2, got %v", task.ClaimedBy)
	}
}

// TestE2E_TaskPriority tests that tasks are ordered by priority in ready list.
func TestE2E_TaskPriority(t *testing.T) {
	suite := setupE2E(t)
	defer suite.cleanup()

	projectName := "priority-test"
	_ = suite.createProject(projectName)

	// Create tasks with different priorities
	lowID := suite.createTaskWithPriority(projectName, "Low priority task", domain.PriorityLow)
	normalID := suite.createTaskWithPriority(projectName, "Normal priority task", domain.PriorityNormal)
	highID := suite.createTaskWithPriority(projectName, "High priority task", domain.PriorityHigh)
	criticalID := suite.createTaskWithPriority(projectName, "Critical priority task", domain.PriorityCritical)

	t.Logf("Created tasks: critical=%s, high=%s, normal=%s, low=%s", criticalID, highID, normalID, lowID)

	// Get ready tasks - should be sorted by priority (highest first)
	readyTasks := suite.listReadyTasks(projectName)

	if len(readyTasks) < 4 {
		t.Fatalf("Expected 4 ready tasks, got %d", len(readyTasks))
	}

	// Verify order: critical (0) should be first
	if readyTasks[0].ID != criticalID {
		t.Errorf("First task should be critical (ID=%s), got %s", criticalID, readyTasks[0].ID)
	}

	// Verify priority ordering
	for i := 1; i < len(readyTasks); i++ {
		if readyTasks[i].Priority < readyTasks[i-1].Priority {
			t.Errorf("Tasks not sorted by priority: task %d has priority %d, but previous task has priority %d",
				i, readyTasks[i].Priority, readyTasks[i-1].Priority)
		}
	}
}

// TestE2E_TaskUpdate tests updating task properties.
func TestE2E_TaskUpdate(t *testing.T) {
	suite := setupE2E(t)
	defer suite.cleanup()

	projectName := "update-test"
	projectDir := suite.createProject(projectName)

	// Create a task
	stdout, _, exitCode := suite.runCLIInDir(projectDir, "create", "Original title")
	if exitCode != 0 {
		t.Fatalf("Failed to create task: %s", stdout)
	}
	taskID := extractTaskIDFromOutput(stdout)

	// Update title via CLI
	stdout, stderr, exitCode := suite.runCLIInDir(projectDir, "edit", taskID, "-t", "Updated title")
	if exitCode != 0 {
		t.Fatalf("Failed to update task: exit=%d, stderr=%s", exitCode, stderr)
	}

	// Verify update
	task := suite.getTask(projectName, taskID)
	if task.Title != "Updated title" {
		t.Errorf("Task title = %q, want %q", task.Title, "Updated title")
	}

	// Update priority via CLI
	stdout, stderr, exitCode = suite.runCLIInDir(projectDir, "edit", taskID, "-p", "high")
	if exitCode != 0 {
		t.Fatalf("Failed to update priority: exit=%d, stderr=%s", exitCode, stderr)
	}

	// Verify priority update
	task = suite.getTask(projectName, taskID)
	if task.Priority != domain.PriorityHigh {
		t.Errorf("Task priority = %d, want %d (high)", task.Priority, domain.PriorityHigh)
	}
}

// TestE2E_TaskDelete tests deleting a task.
func TestE2E_TaskDelete(t *testing.T) {
	suite := setupE2E(t)
	defer suite.cleanup()

	projectName := "delete-test"
	projectDir := suite.createProject(projectName)

	// Create a task
	taskID := suite.createTask(projectName, "Task to delete")

	// Verify task exists
	tasks := suite.listTasks(projectName)
	found := false
	for _, task := range tasks {
		if task.ID == taskID {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("Task should exist before delete")
	}

	// Delete via CLI
	stdout, stderr, exitCode := suite.runCLIInDir(projectDir, "delete", taskID)
	if exitCode != 0 {
		t.Fatalf("Failed to delete task: exit=%d, stderr=%s", exitCode, stderr)
	}

	if !containsString(stdout, "deleted") {
		t.Errorf("Delete output should confirm deletion: %s", stdout)
	}

	// Verify task is gone
	c := suite.getClient(projectName, "test-agent")
	_, err := c.GetTask(context.Background(), taskID)
	if err == nil {
		t.Error("Task should not exist after delete")
	}
}

// TestE2E_BlockUnblock tests blocking and unblocking a task.
func TestE2E_BlockUnblock(t *testing.T) {
	suite := setupE2E(t)
	defer suite.cleanup()

	projectName := "block-test"
	projectDir := suite.createProject(projectName)

	// Create and claim a task
	taskID := suite.createTask(projectName, "Task to block")
	_, err := suite.claimTask(projectName, taskID, "test-agent")
	if err != nil {
		t.Fatalf("Failed to claim task: %v", err)
	}

	// Block the task via CLI
	_, stderr, exitCode := suite.runCLIInDir(projectDir, "block", taskID)
	if exitCode != 0 {
		t.Fatalf("Failed to block task: exit=%d, stderr=%s", exitCode, stderr)
	}

	// Verify task is blocked
	task := suite.getTask(projectName, taskID)
	if task.Status != domain.StatusBlocked {
		t.Errorf("Task status after block = %s, want blocked", task.Status)
	}

	// Unblock the task via CLI
	_, stderr, exitCode = suite.runCLIInDir(projectDir, "unblock", taskID)
	if exitCode != 0 {
		t.Fatalf("Failed to unblock task: exit=%d, stderr=%s", exitCode, stderr)
	}

	// Verify task is open again
	task = suite.getTask(projectName, taskID)
	if task.Status != domain.StatusOpen {
		t.Errorf("Task status after unblock = %s, want open", task.Status)
	}
}

// TestE2E_DependencyList tests listing task dependencies.
func TestE2E_DependencyList(t *testing.T) {
	suite := setupE2E(t)
	defer suite.cleanup()

	projectName := "deplist-test"
	projectDir := suite.createProject(projectName)

	// Create tasks
	taskA := suite.createTask(projectName, "Task A")
	taskB := suite.createTask(projectName, "Task B")
	taskC := suite.createTask(projectName, "Task C")

	// B depends on A, C depends on B
	err := suite.addDependency(projectName, taskB, taskA)
	if err != nil {
		t.Fatalf("Failed to add dependency B->A: %v", err)
	}

	err = suite.addDependency(projectName, taskC, taskB)
	if err != nil {
		t.Fatalf("Failed to add dependency C->B: %v", err)
	}

	// List dependencies for task B via CLI
	stdout, stderr, exitCode := suite.runCLIInDir(projectDir, "dep", "list", taskB)
	if exitCode != 0 {
		t.Fatalf("Failed to list dependencies: exit=%d, stderr=%s", exitCode, stderr)
	}

	// B depends on A
	if !containsString(stdout, taskA) {
		t.Errorf("Dependency list should show that B depends on A:\n%s", stdout)
	}

	// Also verify C blocks B (B blocks C)
	stdout, stderr, exitCode = suite.runCLIInDir(projectDir, "dep", "list", taskB)
	if exitCode != 0 {
		t.Fatalf("Failed to list dependencies: exit=%d, stderr=%s", exitCode, stderr)
	}

	// Output should show both relationships
	if !containsString(stdout, "depends on") || !strings.Contains(stdout, taskA) {
		t.Errorf("Dependency list should show B depends on A:\n%s", stdout)
	}
}

// TestE2E_RemoveDependency tests removing a dependency between tasks.
func TestE2E_RemoveDependency(t *testing.T) {
	suite := setupE2E(t)
	defer suite.cleanup()

	projectName := "deprm-test"
	projectDir := suite.createProject(projectName)

	// Create tasks
	taskA := suite.createTask(projectName, "Task A")
	taskB := suite.createTask(projectName, "Task B")

	// Add dependency: B depends on A
	err := suite.addDependency(projectName, taskB, taskA)
	if err != nil {
		t.Fatalf("Failed to add dependency: %v", err)
	}

	// Task B should NOT be ready (depends on A)
	readyTasks := suite.listReadyTasks(projectName)
	for _, task := range readyTasks {
		if task.ID == taskB {
			t.Error("Task B should not be ready before removing dependency")
		}
	}

	// Remove dependency via CLI
	stdout, stderr, exitCode := suite.runCLIInDir(projectDir, "dep", "rm", taskB, taskA)
	if exitCode != 0 {
		t.Fatalf("Failed to remove dependency: exit=%d, stderr=%s", exitCode, stderr)
	}

	if !containsString(stdout, "no longer depends") {
		t.Errorf("Output should confirm dependency removal: %s", stdout)
	}

	// Task B should now be ready
	readyTasks = suite.listReadyTasks(projectName)
	found := false
	for _, task := range readyTasks {
		if task.ID == taskB {
			found = true
			break
		}
	}
	if !found {
		t.Error("Task B should be ready after removing dependency")
	}
}
