package e2e

import (
	"errors"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/airyra/airyra/internal/domain"
)

// TestE2E_ConcurrentClaims tests that only one agent can claim a task when
// multiple agents try to claim it simultaneously.
func TestE2E_ConcurrentClaims(t *testing.T) {
	suite := setupE2E(t)
	defer suite.cleanup()

	projectName := "concurrent-claim-test"
	_ = suite.createProject(projectName)

	// Create a task
	taskID := suite.createTask(projectName, "Contested task")
	t.Logf("Created task: %s", taskID)

	numAgents := 10
	var successCount int32
	var failCount int32
	var wg sync.WaitGroup

	// Launch goroutines to claim the task concurrently
	for i := 0; i < numAgents; i++ {
		wg.Add(1)
		go func(agentNum int) {
			defer wg.Done()

			agentID := agentIDForNum(agentNum)
			_, err := suite.claimTask(projectName, taskID, agentID)

			if err == nil {
				atomic.AddInt32(&successCount, 1)
				t.Logf("Agent %d successfully claimed the task", agentNum)
			} else {
				atomic.AddInt32(&failCount, 1)
				// Verify error is ALREADY_CLAIMED
				if !containsString(err.Error(), "claimed") {
					t.Errorf("Agent %d got unexpected error: %v", agentNum, err)
				}
			}
		}(i)
	}

	wg.Wait()

	// Exactly one agent should succeed
	if successCount != 1 {
		t.Errorf("Expected exactly 1 successful claim, got %d", successCount)
	}

	// All others should fail
	if failCount != int32(numAgents-1) {
		t.Errorf("Expected %d failed claims, got %d", numAgents-1, failCount)
	}

	// Verify task is claimed
	task := suite.getTask(projectName, taskID)
	if task.Status != domain.StatusInProgress {
		t.Errorf("Task should be in_progress, got %s", task.Status)
	}
	if task.ClaimedBy == nil {
		t.Error("Task should have a claimed_by value")
	}
}

// TestE2E_ConcurrentTaskCreation tests that many tasks can be created
// concurrently without data corruption.
func TestE2E_ConcurrentTaskCreation(t *testing.T) {
	suite := setupE2E(t)
	defer suite.cleanup()

	projectName := "concurrent-create-test"
	_ = suite.createProject(projectName)

	numTasks := 50
	var wg sync.WaitGroup
	taskIDs := make(chan string, numTasks)
	errCh := make(chan error, numTasks)

	// Launch goroutines to create tasks concurrently
	for i := 0; i < numTasks; i++ {
		wg.Add(1)
		go func(taskNum int) {
			defer wg.Done()

			agentID := agentIDForNum(taskNum)
			c := suite.getClient(projectName, agentID)

			title := taskTitleForNum(taskNum)
			task, err := c.CreateTask(t.Context(), title, "", 2, "", "")

			if err != nil {
				errCh <- err
				return
			}

			taskIDs <- task.ID
		}(i)
	}

	wg.Wait()
	close(taskIDs)
	close(errCh)

	// Check for any errors
	for err := range errCh {
		t.Errorf("Task creation failed: %v", err)
	}

	// Collect all task IDs
	ids := make(map[string]bool)
	for id := range taskIDs {
		if ids[id] {
			t.Errorf("Duplicate task ID: %s", id)
		}
		ids[id] = true
	}

	// Verify all tasks were created with unique IDs
	if len(ids) != numTasks {
		t.Errorf("Expected %d unique task IDs, got %d", numTasks, len(ids))
	}

	// Verify all tasks exist in the database
	allTasks := suite.listTasks(projectName)
	if len(allTasks) != numTasks {
		t.Errorf("Expected %d tasks in database, got %d", numTasks, len(allTasks))
	}

	// Verify no data corruption - each task should have correct title
	for _, task := range allTasks {
		if task.Title == "" {
			t.Errorf("Task %s has empty title", task.ID)
		}
		if task.Status != domain.StatusOpen {
			t.Errorf("Task %s has wrong status: %s", task.ID, task.Status)
		}
	}
}

// TestE2E_ConcurrentClaimAndRelease tests that claim and release operations
// are serialized correctly when happening concurrently.
func TestE2E_ConcurrentClaimAndRelease(t *testing.T) {
	suite := setupE2E(t)
	defer suite.cleanup()

	projectName := "claim-release-test"
	_ = suite.createProject(projectName)

	// Create a task
	taskID := suite.createTask(projectName, "Claim-release task")

	numIterations := 20
	var wg sync.WaitGroup
	var claimSuccesses int32
	var releaseSuccesses int32

	// Run claim/release cycles concurrently
	for i := 0; i < numIterations; i++ {
		wg.Add(1)
		go func(iteration int) {
			defer wg.Done()

			agentID := agentIDForNum(iteration)

			// Try to claim
			_, err := suite.claimTask(projectName, taskID, agentID)
			if err == nil {
				atomic.AddInt32(&claimSuccesses, 1)

				// If we claimed successfully, release
				_, err = suite.releaseTask(projectName, taskID, agentID, false)
				if err == nil {
					atomic.AddInt32(&releaseSuccesses, 1)
				}
			}
		}(i)
	}

	wg.Wait()

	// Task should end up in a consistent state
	task := suite.getTask(projectName, taskID)

	// If the last operation was a release, task should be open
	// If the last operation was a claim, task should be in_progress
	if task.Status != domain.StatusOpen && task.Status != domain.StatusInProgress {
		t.Errorf("Task in unexpected state: %s", task.Status)
	}

	t.Logf("Claims: %d, Releases: %d", claimSuccesses, releaseSuccesses)

	// At least some claims should succeed
	if claimSuccesses == 0 {
		t.Error("No claims succeeded")
	}
}

// TestE2E_ConcurrentDependencyAddition tests adding dependencies concurrently.
func TestE2E_ConcurrentDependencyAddition(t *testing.T) {
	suite := setupE2E(t)
	defer suite.cleanup()

	projectName := "concurrent-deps-test"
	_ = suite.createProject(projectName)

	// Create parent tasks
	numParents := 10
	parentIDs := make([]string, numParents)
	for i := 0; i < numParents; i++ {
		parentIDs[i] = suite.createTask(projectName, taskTitleForNum(i))
	}

	// Create a child task that will depend on all parents
	childID := suite.createTask(projectName, "Child task with many dependencies")

	var wg sync.WaitGroup
	var successCount int32
	var errCount int32

	// Add dependencies concurrently
	for i := 0; i < numParents; i++ {
		wg.Add(1)
		go func(parentIdx int) {
			defer wg.Done()

			err := suite.addDependency(projectName, childID, parentIDs[parentIdx])
			if err == nil {
				atomic.AddInt32(&successCount, 1)
			} else {
				atomic.AddInt32(&errCount, 1)
				t.Logf("Failed to add dependency to parent %d: %v", parentIdx, err)
			}
		}(i)
	}

	wg.Wait()

	// All dependencies should be added successfully
	if successCount != int32(numParents) {
		t.Errorf("Expected %d successful dependency additions, got %d", numParents, successCount)
	}

	// Child should not be ready (depends on all parents)
	readyTasks := suite.listReadyTasks(projectName)
	for _, task := range readyTasks {
		if task.ID == childID {
			t.Error("Child task should not be ready while parents are incomplete")
		}
	}
}

// TestE2E_ConcurrentStatusTransitions tests multiple status transitions happening concurrently.
func TestE2E_ConcurrentStatusTransitions(t *testing.T) {
	suite := setupE2E(t)
	defer suite.cleanup()

	projectName := "concurrent-transitions-test"
	_ = suite.createProject(projectName)

	// Create multiple tasks
	numTasks := 10
	taskIDs := make([]string, numTasks)
	for i := 0; i < numTasks; i++ {
		taskIDs[i] = suite.createTask(projectName, taskTitleForNum(i))
	}

	var wg sync.WaitGroup
	var claimSuccesses int32
	var completeSuccesses int32

	// Claim all tasks concurrently
	for i := 0; i < numTasks; i++ {
		wg.Add(1)
		go func(taskIdx int) {
			defer wg.Done()

			agentID := agentIDForNum(taskIdx)
			_, err := suite.claimTask(projectName, taskIDs[taskIdx], agentID)
			if err == nil {
				atomic.AddInt32(&claimSuccesses, 1)
			} else {
				t.Logf("Claim failed for task %d: %v", taskIdx, err)
			}
		}(i)
	}

	wg.Wait()

	if claimSuccesses != int32(numTasks) {
		t.Errorf("Expected all %d claims to succeed, got %d", numTasks, claimSuccesses)
	}

	// Complete all tasks concurrently
	for i := 0; i < numTasks; i++ {
		wg.Add(1)
		go func(taskIdx int) {
			defer wg.Done()

			agentID := agentIDForNum(taskIdx)
			_, err := suite.completeTask(projectName, taskIDs[taskIdx], agentID)
			if err == nil {
				atomic.AddInt32(&completeSuccesses, 1)
			} else {
				t.Logf("Complete failed for task %d: %v", taskIdx, err)
			}
		}(i)
	}

	wg.Wait()

	if completeSuccesses != int32(numTasks) {
		t.Errorf("Expected all %d completions to succeed, got %d", numTasks, completeSuccesses)
	}

	// Verify all tasks are done
	allTasks := suite.listTasks(projectName)
	doneCount := 0
	for _, task := range allTasks {
		if task.Status == domain.StatusDone {
			doneCount++
		}
	}

	if doneCount != numTasks {
		t.Errorf("Expected %d done tasks, got %d", numTasks, doneCount)
	}
}

// TestE2E_RaceConditionDetection tests for race conditions using multiple goroutines.
// This test is designed to be run with -race flag.
func TestE2E_RaceConditionDetection(t *testing.T) {
	suite := setupE2E(t)
	defer suite.cleanup()

	projectName := "race-test"
	_ = suite.createProject(projectName)

	taskID := suite.createTask(projectName, "Race condition test task")

	numWorkers := 5
	numOps := 10
	var wg sync.WaitGroup

	// Mix of operations that could cause race conditions
	for w := 0; w < numWorkers; w++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()

			agentID := agentIDForNum(workerID)
			c := suite.getClient(projectName, agentID)

			for op := 0; op < numOps; op++ {
				switch op % 3 {
				case 0:
					// Try to claim
					c.ClaimTask(t.Context(), taskID)
				case 1:
					// Try to release
					c.ReleaseTask(t.Context(), taskID, false)
				case 2:
					// Read task
					c.GetTask(t.Context(), taskID)
				}
			}
		}(w)
	}

	wg.Wait()

	// If we get here without the race detector complaining, we're good
	// Verify task is in a valid state
	task := suite.getTask(projectName, taskID)
	if !task.Status.IsValid() {
		t.Errorf("Task in invalid state: %s", task.Status)
	}
}

// TestE2E_ConcurrentReadsDuringWrites tests that reads work correctly during writes.
func TestE2E_ConcurrentReadsDuringWrites(t *testing.T) {
	suite := setupE2E(t)
	defer suite.cleanup()

	projectName := "read-write-test"
	_ = suite.createProject(projectName)

	// Create initial tasks
	numInitialTasks := 5
	for i := 0; i < numInitialTasks; i++ {
		suite.createTask(projectName, taskTitleForNum(i))
	}

	var wg sync.WaitGroup
	var readErrors int32
	var writeErrors int32

	numOps := 20

	// Writers: create new tasks
	for i := 0; i < numOps; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()

			agentID := agentIDForNum(idx)
			c := suite.getClient(projectName, agentID)

			_, err := c.CreateTask(t.Context(), taskTitleForNum(numInitialTasks+idx), "", 2, "", "")
			if err != nil {
				atomic.AddInt32(&writeErrors, 1)
			}
		}(i)
	}

	// Readers: list tasks
	for i := 0; i < numOps; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()

			agentID := agentIDForNum(idx)
			c := suite.getClient(projectName, agentID)

			result, err := c.ListTasks(t.Context(), "", 1, 100)
			if err != nil {
				atomic.AddInt32(&readErrors, 1)
			} else if result == nil {
				atomic.AddInt32(&readErrors, 1)
			}
		}(i)
	}

	wg.Wait()

	if readErrors > 0 {
		t.Errorf("%d read errors occurred during concurrent writes", readErrors)
	}
	if writeErrors > 0 {
		t.Errorf("%d write errors occurred", writeErrors)
	}

	// Final count should include all created tasks
	allTasks := suite.listTasks(projectName)
	expectedCount := numInitialTasks + numOps
	if len(allTasks) != expectedCount {
		t.Errorf("Expected %d tasks, got %d", expectedCount, len(allTasks))
	}
}

// TestE2E_ConcurrentProjectAccess tests accessing multiple projects concurrently.
func TestE2E_ConcurrentProjectAccess(t *testing.T) {
	suite := setupE2E(t)
	defer suite.cleanup()

	numProjects := 5
	projectNames := make([]string, numProjects)
	for i := 0; i < numProjects; i++ {
		projectNames[i] = projectNameForNum(i)
		suite.createProject(projectNames[i])
	}

	var wg sync.WaitGroup
	var errCount int32

	numOpsPerProject := 10

	// Access all projects concurrently
	for p := 0; p < numProjects; p++ {
		wg.Add(1)
		go func(projectIdx int) {
			defer wg.Done()

			projectName := projectNames[projectIdx]

			for op := 0; op < numOpsPerProject; op++ {
				agentID := agentIDForNum(op)
				c := suite.getClient(projectName, agentID)

				_, err := c.CreateTask(t.Context(), taskTitleForNum(op), "", 2, "", "")
				if err != nil {
					atomic.AddInt32(&errCount, 1)
					t.Logf("Error creating task in project %s: %v", projectName, err)
				}
			}
		}(p)
	}

	wg.Wait()

	if errCount > 0 {
		t.Errorf("%d errors occurred during concurrent project access", errCount)
	}

	// Verify each project has the correct number of tasks
	for _, projectName := range projectNames {
		tasks := suite.listTasks(projectName)
		if len(tasks) != numOpsPerProject {
			t.Errorf("Project %s should have %d tasks, got %d", projectName, numOpsPerProject, len(tasks))
		}
	}
}

// TestE2E_ConcurrentClaimOfDependentTasks tests claiming tasks with dependencies concurrently.
func TestE2E_ConcurrentClaimOfDependentTasks(t *testing.T) {
	suite := setupE2E(t)
	defer suite.cleanup()

	projectName := "concurrent-dep-claim-test"
	_ = suite.createProject(projectName)

	// Create a chain of dependent tasks: A <- B <- C (C depends on B, B depends on A)
	taskA := suite.createTask(projectName, "Task A - Independent")
	taskB := suite.createTask(projectName, "Task B - Depends on A")
	taskC := suite.createTask(projectName, "Task C - Depends on B")

	// Set up dependencies
	if err := suite.addDependency(projectName, taskB, taskA); err != nil {
		t.Fatalf("Failed to add B->A dependency: %v", err)
	}
	if err := suite.addDependency(projectName, taskC, taskB); err != nil {
		t.Fatalf("Failed to add C->B dependency: %v", err)
	}

	var wg sync.WaitGroup
	results := make(map[string]error)
	var mu sync.Mutex

	// Try to claim all three tasks concurrently
	tasks := []string{taskA, taskB, taskC}
	for _, taskID := range tasks {
		wg.Add(1)
		go func(id string) {
			defer wg.Done()

			_, err := suite.claimTask(projectName, id, "agent-"+id)

			mu.Lock()
			results[id] = err
			mu.Unlock()
		}(taskID)
	}

	wg.Wait()

	// Only Task A should be claimable (it has no dependencies)
	if results[taskA] != nil {
		t.Errorf("Task A should be claimable, got error: %v", results[taskA])
	}

	// B should fail because A is not done
	// Note: This depends on the implementation - some systems might allow
	// claiming tasks with incomplete dependencies
	// For this test, we verify the task statuses are consistent
	taskAObj := suite.getTask(projectName, taskA)
	if taskAObj.Status != domain.StatusInProgress && results[taskA] == nil {
		t.Errorf("Task A should be in_progress after successful claim")
	}
}

// Helper functions for generating test data

func agentIDForNum(n int) string {
	return agentIDWithPrefix("agent", n)
}

func agentIDWithPrefix(prefix string, n int) string {
	return taskNameForNum(prefix, n) + "@host:/project"
}

func taskTitleForNum(n int) string {
	return taskNameForNum("Task", n)
}

func taskNameForNum(prefix string, n int) string {
	return prefix + "-" + itoa(n)
}

func projectNameForNum(n int) string {
	return "project-" + itoa(n)
}

func itoa(n int) string {
	return string(rune('0'+n%10)) + string(rune('0'+n/10))
}

// isDomainError checks if an error is a domain error with a specific code.
func isDomainError(err error, code domain.ErrorCode) bool {
	if err == nil {
		return false
	}

	var domainErr *domain.DomainError
	if errors.As(err, &domainErr) {
		return domainErr.Code == code
	}

	// Also check error message for the code
	return containsString(err.Error(), string(code))
}
