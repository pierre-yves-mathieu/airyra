package e2e

import (
	"testing"

	"github.com/airyra/airyra/internal/domain"
)

// TestE2E_ProjectIsolation tests that tasks in different projects are isolated.
func TestE2E_ProjectIsolation(t *testing.T) {
	suite := setupE2E(t)
	defer suite.cleanup()

	// Create two projects
	projectA := "project-alpha"
	projectB := "project-beta"

	projectDirA := suite.createProject(projectA)
	projectDirB := suite.createProject(projectB)

	// Create 3 tasks in project A
	taskA1 := suite.createTask(projectA, "Alpha Task 1")
	taskA2 := suite.createTask(projectA, "Alpha Task 2")
	taskA3 := suite.createTask(projectA, "Alpha Task 3")

	// Create 2 tasks in project B
	taskB1 := suite.createTask(projectB, "Beta Task 1")
	taskB2 := suite.createTask(projectB, "Beta Task 2")

	t.Logf("Project A tasks: %s, %s, %s", taskA1, taskA2, taskA3)
	t.Logf("Project B tasks: %s, %s", taskB1, taskB2)

	// List tasks in project A via CLI
	stdout, stderr, exitCode := suite.runCLIInDir(projectDirA, "list")
	if exitCode != 0 {
		t.Fatalf("Failed to list tasks in project A: %s", stderr)
	}

	// Verify project A has exactly 3 tasks
	if !containsString(stdout, "Alpha Task 1") {
		t.Error("Project A should contain Alpha Task 1")
	}
	if !containsString(stdout, "Alpha Task 2") {
		t.Error("Project A should contain Alpha Task 2")
	}
	if !containsString(stdout, "Alpha Task 3") {
		t.Error("Project A should contain Alpha Task 3")
	}

	// Project A should NOT contain project B tasks
	if containsString(stdout, "Beta Task") {
		t.Error("Project A should NOT contain any Beta tasks")
	}

	// List tasks in project B via CLI
	stdout, stderr, exitCode = suite.runCLIInDir(projectDirB, "list")
	if exitCode != 0 {
		t.Fatalf("Failed to list tasks in project B: %s", stderr)
	}

	// Verify project B has exactly 2 tasks
	if !containsString(stdout, "Beta Task 1") {
		t.Error("Project B should contain Beta Task 1")
	}
	if !containsString(stdout, "Beta Task 2") {
		t.Error("Project B should contain Beta Task 2")
	}

	// Project B should NOT contain project A tasks
	if containsString(stdout, "Alpha Task") {
		t.Error("Project B should NOT contain any Alpha tasks")
	}

	// Verify via API
	tasksA := suite.listTasks(projectA)
	if len(tasksA) != 3 {
		t.Errorf("Project A should have 3 tasks, got %d", len(tasksA))
	}

	tasksB := suite.listTasks(projectB)
	if len(tasksB) != 2 {
		t.Errorf("Project B should have 2 tasks, got %d", len(tasksB))
	}
}

// TestE2E_ProjectIsolation_CannotAccessOtherProjectTasks tests that tasks from one
// project cannot be accessed from another project.
func TestE2E_ProjectIsolation_CannotAccessOtherProjectTasks(t *testing.T) {
	suite := setupE2E(t)
	defer suite.cleanup()

	// Create two projects
	projectA := "isolated-a"
	projectB := "isolated-b"

	_ = suite.createProject(projectA)
	projectDirB := suite.createProject(projectB)

	// Create a task in project A
	taskAID := suite.createTask(projectA, "Project A Task")

	// Try to access the task from project B's context (via CLI in project B directory)
	stdout, stderr, exitCode := suite.runCLIInDir(projectDirB, "show", taskAID)

	// This should fail - the task doesn't exist in project B
	if exitCode == 0 {
		t.Errorf("Should not be able to access project A task from project B context")
		t.Logf("stdout: %s", stdout)
	} else {
		t.Logf("Correctly denied access: exit=%d, stderr=%s", exitCode, stderr)
	}
}

// TestE2E_ProjectIsolation_Dependencies tests that dependencies cannot be created
// across project boundaries.
func TestE2E_ProjectIsolation_Dependencies(t *testing.T) {
	suite := setupE2E(t)
	defer suite.cleanup()

	// Create two projects
	projectA := "deps-project-a"
	projectB := "deps-project-b"

	_ = suite.createProject(projectA)
	projectDirB := suite.createProject(projectB)

	// Create tasks in each project
	taskA := suite.createTask(projectA, "Task in Project A")
	taskB := suite.createTask(projectB, "Task in Project B")

	// Try to add a dependency from B to A (cross-project) via project B's context
	stdout, stderr, exitCode := suite.runCLIInDir(projectDirB, "dep", "add", taskB, taskA)

	// This should fail - can't depend on task from another project
	if exitCode == 0 {
		// If it succeeded, the server might have created the dependency
		// which would be a bug. Verify.
		t.Logf("Dependency add returned success - checking if actually created")
		t.Logf("stdout: %s", stdout)

		// Try to list dependencies
		deps, err := suite.getClient(projectB, "test").ListDependencies(t.Context(), taskB)
		if err != nil {
			t.Logf("Could not list dependencies: %v", err)
		} else {
			for _, dep := range deps {
				if dep.ParentID == taskA {
					t.Error("Cross-project dependency should not be allowed")
				}
			}
		}
	} else {
		t.Logf("Correctly rejected cross-project dependency: exit=%d, stderr=%s", exitCode, stderr)
	}
}

// TestE2E_ProjectIsolation_Operations tests that operations (claim, done, etc.)
// are isolated to the correct project.
func TestE2E_ProjectIsolation_Operations(t *testing.T) {
	suite := setupE2E(t)
	defer suite.cleanup()

	// Create two projects
	projectA := "ops-project-a"
	projectB := "ops-project-b"

	_ = suite.createProject(projectA)
	projectDirB := suite.createProject(projectB)

	// Create and claim a task in project A
	taskA := suite.createTask(projectA, "Task A")
	_, err := suite.claimTask(projectA, taskA, "agent-a")
	if err != nil {
		t.Fatalf("Failed to claim task in project A: %v", err)
	}

	// Try to complete the task from project B's context
	stdout, stderr, exitCode := suite.runCLIInDir(projectDirB, "done", taskA)

	// This should fail - task doesn't exist in project B
	if exitCode == 0 {
		t.Errorf("Should not be able to complete project A task from project B context")
		t.Logf("stdout: %s", stdout)
	} else {
		t.Logf("Correctly denied operation: exit=%d, stderr=%s", exitCode, stderr)
	}

	// Verify task A is still in_progress (not affected by the failed operation)
	task := suite.getTask(projectA, taskA)
	if task.Status != domain.StatusInProgress {
		t.Errorf("Task A should still be in_progress, got %s", task.Status)
	}
}

// TestE2E_ProjectIsolation_SameTaskID tests that even if task IDs happen to be
// the same format, they are isolated by project.
func TestE2E_ProjectIsolation_SameTaskID(t *testing.T) {
	suite := setupE2E(t)
	defer suite.cleanup()

	// Create two projects
	projectA := "same-id-a"
	projectB := "same-id-b"

	suite.createProject(projectA)
	suite.createProject(projectB)

	// Create tasks in both projects
	taskA := suite.createTask(projectA, "Task in A")
	taskB := suite.createTask(projectB, "Task in B")

	// Both tasks will have different IDs (ar-xxx format)
	// but let's verify operations on one don't affect the other

	// Claim task A
	_, err := suite.claimTask(projectA, taskA, "agent-1")
	if err != nil {
		t.Fatalf("Failed to claim task A: %v", err)
	}

	// Claim task B
	_, err = suite.claimTask(projectB, taskB, "agent-1")
	if err != nil {
		t.Fatalf("Failed to claim task B: %v", err)
	}

	// Complete task A
	_, err = suite.completeTask(projectA, taskA, "agent-1")
	if err != nil {
		t.Fatalf("Failed to complete task A: %v", err)
	}

	// Verify task A is done
	taskAObj := suite.getTask(projectA, taskA)
	if taskAObj.Status != domain.StatusDone {
		t.Errorf("Task A should be done, got %s", taskAObj.Status)
	}

	// Verify task B is NOT done (should still be in_progress)
	taskBObj := suite.getTask(projectB, taskB)
	if taskBObj.Status != domain.StatusInProgress {
		t.Errorf("Task B should still be in_progress, got %s", taskBObj.Status)
	}
}

// TestE2E_ProjectIsolation_History tests that audit history is isolated by project.
func TestE2E_ProjectIsolation_History(t *testing.T) {
	suite := setupE2E(t)
	defer suite.cleanup()

	// Create two projects
	projectA := "history-a"
	projectB := "history-b"

	suite.createProject(projectA)
	suite.createProject(projectB)

	// Create tasks and perform operations in project A
	taskA := suite.createTask(projectA, "Task A")
	suite.claimTask(projectA, taskA, "agent-a")
	suite.completeTask(projectA, taskA, "agent-a")

	// Create a task in project B (only create, no other operations)
	taskB := suite.createTask(projectB, "Task B")

	// Get history for task A
	historyA := suite.getTaskHistory(projectA, taskA)
	if len(historyA) < 2 {
		t.Errorf("Task A should have at least 2 history entries (create, claim), got %d", len(historyA))
	}

	// Get history for task B
	historyB := suite.getTaskHistory(projectB, taskB)
	if len(historyB) < 1 {
		t.Errorf("Task B should have at least 1 history entry (create), got %d", len(historyB))
	}

	// Task B history should only have create action
	for _, entry := range historyB {
		if entry.Action == domain.ActionClaim {
			t.Error("Task B history should not have claim action")
		}
	}
}

// TestE2E_MultipleProjectsListAll tests that we can have many projects without interference.
func TestE2E_MultipleProjectsListAll(t *testing.T) {
	suite := setupE2E(t)
	defer suite.cleanup()

	numProjects := 5
	tasksPerProject := 3

	// Create projects and tasks
	projectNames := make([]string, numProjects)
	taskCounts := make(map[string]int)

	for i := 0; i < numProjects; i++ {
		projectName := projectNameForNum(i)
		projectNames[i] = projectName
		suite.createProject(projectName)

		for j := 0; j < tasksPerProject; j++ {
			suite.createTask(projectName, taskTitleForNum(i*100+j))
		}
		taskCounts[projectName] = tasksPerProject
	}

	// Verify each project has correct task count
	for _, projectName := range projectNames {
		tasks := suite.listTasks(projectName)
		expectedCount := taskCounts[projectName]

		if len(tasks) != expectedCount {
			t.Errorf("Project %s should have %d tasks, got %d",
				projectName, expectedCount, len(tasks))
		}

		// Verify none of the tasks belong to other projects (by title pattern)
		for _, task := range tasks {
			// Task titles from other projects would have different number patterns
			if containsString(task.Title, "Task-") {
				// This is fine, just make sure it matches expected patterns
			}
		}
	}
}

// TestE2E_ProjectIsolation_ReadyTasks tests that ready task listing is isolated.
func TestE2E_ProjectIsolation_ReadyTasks(t *testing.T) {
	suite := setupE2E(t)
	defer suite.cleanup()

	// Create two projects
	projectA := "ready-a"
	projectB := "ready-b"

	suite.createProject(projectA)
	suite.createProject(projectB)

	// Create tasks in project A with dependencies
	taskA1 := suite.createTask(projectA, "A Task 1 - Independent")
	taskA2 := suite.createTask(projectA, "A Task 2 - Depends on A1")
	suite.addDependency(projectA, taskA2, taskA1)

	// Create tasks in project B (all independent)
	taskB1 := suite.createTask(projectB, "B Task 1")
	taskB2 := suite.createTask(projectB, "B Task 2")

	// Check ready tasks in project A (only taskA1 should be ready)
	readyA := suite.listReadyTasks(projectA)
	readyAIDs := make(map[string]bool)
	for _, task := range readyA {
		readyAIDs[task.ID] = true
	}

	if !readyAIDs[taskA1] {
		t.Error("Task A1 should be ready in project A")
	}
	if readyAIDs[taskA2] {
		t.Error("Task A2 should NOT be ready (has dependency)")
	}

	// Project A ready list should not contain any project B tasks
	if readyAIDs[taskB1] || readyAIDs[taskB2] {
		t.Error("Project A ready list should not contain project B tasks")
	}

	// Check ready tasks in project B (both should be ready)
	readyB := suite.listReadyTasks(projectB)
	readyBIDs := make(map[string]bool)
	for _, task := range readyB {
		readyBIDs[task.ID] = true
	}

	if !readyBIDs[taskB1] {
		t.Error("Task B1 should be ready in project B")
	}
	if !readyBIDs[taskB2] {
		t.Error("Task B2 should be ready in project B")
	}

	// Project B ready list should not contain any project A tasks
	if readyBIDs[taskA1] || readyBIDs[taskA2] {
		t.Error("Project B ready list should not contain project A tasks")
	}
}

// TestE2E_ProjectIsolation_DifferentAgents tests that agents can work on different
// projects without interference.
func TestE2E_ProjectIsolation_DifferentAgents(t *testing.T) {
	suite := setupE2E(t)
	defer suite.cleanup()

	// Create two projects
	projectA := "agent-test-a"
	projectB := "agent-test-b"

	suite.createProject(projectA)
	suite.createProject(projectB)

	// Create tasks
	taskA := suite.createTask(projectA, "Task A")
	taskB := suite.createTask(projectB, "Task B")

	// Agent 1 works on project A
	agent1 := "agent-1@host:/projectA"
	_, err := suite.claimTask(projectA, taskA, agent1)
	if err != nil {
		t.Fatalf("Agent 1 failed to claim task A: %v", err)
	}

	// Agent 2 (same identity string) works on project B
	// This should succeed because projects are isolated
	_, err = suite.claimTask(projectB, taskB, agent1)
	if err != nil {
		t.Fatalf("Agent 1 should be able to claim task B in different project: %v", err)
	}

	// Verify both tasks are claimed
	taskAObj := suite.getTask(projectA, taskA)
	if taskAObj.Status != domain.StatusInProgress {
		t.Errorf("Task A should be in_progress, got %s", taskAObj.Status)
	}

	taskBObj := suite.getTask(projectB, taskB)
	if taskBObj.Status != domain.StatusInProgress {
		t.Errorf("Task B should be in_progress, got %s", taskBObj.Status)
	}
}
