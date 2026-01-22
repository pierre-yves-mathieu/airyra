package e2e

import (
	"context"
	"errors"
	"testing"

	"github.com/airyra/airyra/internal/client"
	"github.com/airyra/airyra/internal/domain"
)

// ========================
// Spec CRUD Tests
// ========================

func TestE2E_Spec_Create(t *testing.T) {
	suite := setupE2E(t)
	defer suite.cleanup()

	projectName := "spec-crud-test"
	suite.createProject(projectName)

	c := suite.getClient(projectName, "test-agent")
	ctx := context.Background()

	// Create a spec
	spec, err := c.CreateSpec(ctx, "Feature: User Authentication", "Implement OAuth2 login flow")
	if err != nil {
		t.Fatalf("Failed to create spec: %v", err)
	}

	// Verify spec properties
	if spec.ID == "" {
		t.Error("Expected spec to have an ID")
	}
	if spec.Title != "Feature: User Authentication" {
		t.Errorf("Expected title 'Feature: User Authentication', got %q", spec.Title)
	}
	if spec.Description == nil || *spec.Description != "Implement OAuth2 login flow" {
		t.Errorf("Expected description 'Implement OAuth2 login flow', got %v", spec.Description)
	}
	if spec.Status != "draft" {
		t.Errorf("Expected status 'draft' for new spec with no tasks, got %q", spec.Status)
	}
	if spec.TaskCount != 0 {
		t.Errorf("Expected task_count 0 for new spec, got %d", spec.TaskCount)
	}
	if spec.DoneCount != 0 {
		t.Errorf("Expected done_count 0 for new spec, got %d", spec.DoneCount)
	}
}

func TestE2E_Spec_Get(t *testing.T) {
	suite := setupE2E(t)
	defer suite.cleanup()

	projectName := "spec-get-test"
	suite.createProject(projectName)

	c := suite.getClient(projectName, "test-agent")
	ctx := context.Background()

	// Create a spec
	created, err := c.CreateSpec(ctx, "Test Spec", "Description")
	if err != nil {
		t.Fatalf("Failed to create spec: %v", err)
	}

	// Get the spec
	spec, err := c.GetSpec(ctx, created.ID)
	if err != nil {
		t.Fatalf("Failed to get spec: %v", err)
	}

	if spec.ID != created.ID {
		t.Errorf("Expected ID %q, got %q", created.ID, spec.ID)
	}
	if spec.Title != created.Title {
		t.Errorf("Expected title %q, got %q", created.Title, spec.Title)
	}
}

func TestE2E_Spec_List(t *testing.T) {
	suite := setupE2E(t)
	defer suite.cleanup()

	projectName := "spec-list-test"
	suite.createProject(projectName)

	c := suite.getClient(projectName, "test-agent")
	ctx := context.Background()

	// Create multiple specs
	_, err := c.CreateSpec(ctx, "Spec 1", "")
	if err != nil {
		t.Fatalf("Failed to create spec 1: %v", err)
	}
	_, err = c.CreateSpec(ctx, "Spec 2", "")
	if err != nil {
		t.Fatalf("Failed to create spec 2: %v", err)
	}

	// List all specs
	list, err := c.ListSpecs(ctx, "", 1, 50)
	if err != nil {
		t.Fatalf("Failed to list specs: %v", err)
	}

	if len(list.Data) != 2 {
		t.Errorf("Expected 2 specs, got %d", len(list.Data))
	}
	if list.Pagination.Total != 2 {
		t.Errorf("Expected total 2, got %d", list.Pagination.Total)
	}
}

func TestE2E_Spec_Update(t *testing.T) {
	suite := setupE2E(t)
	defer suite.cleanup()

	projectName := "spec-update-test"
	suite.createProject(projectName)

	c := suite.getClient(projectName, "test-agent")
	ctx := context.Background()

	// Create a spec
	spec, err := c.CreateSpec(ctx, "Original Title", "Original Description")
	if err != nil {
		t.Fatalf("Failed to create spec: %v", err)
	}

	// Update the spec
	newTitle := "Updated Title"
	newDesc := "Updated Description"
	updated, err := c.UpdateSpec(ctx, spec.ID, client.SpecUpdates{
		Title:       &newTitle,
		Description: &newDesc,
	})
	if err != nil {
		t.Fatalf("Failed to update spec: %v", err)
	}

	if updated.Title != "Updated Title" {
		t.Errorf("Expected title 'Updated Title', got %q", updated.Title)
	}
	if updated.Description == nil || *updated.Description != "Updated Description" {
		t.Errorf("Expected description 'Updated Description', got %v", updated.Description)
	}
}

func TestE2E_Spec_Delete(t *testing.T) {
	suite := setupE2E(t)
	defer suite.cleanup()

	projectName := "spec-delete-test"
	suite.createProject(projectName)

	c := suite.getClient(projectName, "test-agent")
	ctx := context.Background()

	// Create a spec
	spec, err := c.CreateSpec(ctx, "To Delete", "")
	if err != nil {
		t.Fatalf("Failed to create spec: %v", err)
	}

	// Delete the spec
	err = c.DeleteSpec(ctx, spec.ID)
	if err != nil {
		t.Fatalf("Failed to delete spec: %v", err)
	}

	// Verify it's deleted
	_, err = c.GetSpec(ctx, spec.ID)
	if err == nil {
		t.Error("Expected error when getting deleted spec")
	}

	var domainErr *domain.DomainError
	if !errors.As(err, &domainErr) || domainErr.Code != domain.ErrCodeSpecNotFound {
		t.Errorf("Expected SPEC_NOT_FOUND error, got %v", err)
	}
}

// ========================
// Spec Status Computation Tests
// ========================

func TestE2E_Spec_ComputedStatus_Draft(t *testing.T) {
	suite := setupE2E(t)
	defer suite.cleanup()

	projectName := "spec-status-draft"
	suite.createProject(projectName)

	c := suite.getClient(projectName, "test-agent")
	ctx := context.Background()

	// Create a spec with no tasks
	spec, err := c.CreateSpec(ctx, "Draft Spec", "")
	if err != nil {
		t.Fatalf("Failed to create spec: %v", err)
	}

	// Status should be draft
	if spec.Status != "draft" {
		t.Errorf("Expected status 'draft', got %q", spec.Status)
	}
	if spec.TaskCount != 0 {
		t.Errorf("Expected task_count 0, got %d", spec.TaskCount)
	}
}

func TestE2E_Spec_ComputedStatus_Active(t *testing.T) {
	suite := setupE2E(t)
	defer suite.cleanup()

	projectName := "spec-status-active"
	suite.createProject(projectName)

	c := suite.getClient(projectName, "test-agent")
	ctx := context.Background()

	// Create a spec
	spec, err := c.CreateSpec(ctx, "Active Spec", "")
	if err != nil {
		t.Fatalf("Failed to create spec: %v", err)
	}

	// Create a task under the spec
	task, err := c.CreateTask(ctx, "Task 1", "", 2, "", spec.ID)
	if err != nil {
		t.Fatalf("Failed to create task: %v", err)
	}

	// Verify task is associated with spec
	if task.SpecID == nil || *task.SpecID != spec.ID {
		t.Errorf("Expected task spec_id %q, got %v", spec.ID, task.SpecID)
	}

	// Refresh spec
	spec, err = c.GetSpec(ctx, spec.ID)
	if err != nil {
		t.Fatalf("Failed to get spec: %v", err)
	}

	// Status should be active (has incomplete tasks)
	if spec.Status != "active" {
		t.Errorf("Expected status 'active', got %q", spec.Status)
	}
	if spec.TaskCount != 1 {
		t.Errorf("Expected task_count 1, got %d", spec.TaskCount)
	}
	if spec.DoneCount != 0 {
		t.Errorf("Expected done_count 0, got %d", spec.DoneCount)
	}
}

func TestE2E_Spec_ComputedStatus_Done(t *testing.T) {
	suite := setupE2E(t)
	defer suite.cleanup()

	projectName := "spec-status-done"
	suite.createProject(projectName)

	c := suite.getClient(projectName, "test-agent")
	ctx := context.Background()

	// Create a spec
	spec, err := c.CreateSpec(ctx, "Done Spec", "")
	if err != nil {
		t.Fatalf("Failed to create spec: %v", err)
	}

	// Create tasks under the spec
	task1, err := c.CreateTask(ctx, "Task 1", "", 2, "", spec.ID)
	if err != nil {
		t.Fatalf("Failed to create task 1: %v", err)
	}
	task2, err := c.CreateTask(ctx, "Task 2", "", 2, "", spec.ID)
	if err != nil {
		t.Fatalf("Failed to create task 2: %v", err)
	}

	// Complete both tasks
	_, err = c.ClaimTask(ctx, task1.ID)
	if err != nil {
		t.Fatalf("Failed to claim task 1: %v", err)
	}
	_, err = c.CompleteTask(ctx, task1.ID)
	if err != nil {
		t.Fatalf("Failed to complete task 1: %v", err)
	}

	_, err = c.ClaimTask(ctx, task2.ID)
	if err != nil {
		t.Fatalf("Failed to claim task 2: %v", err)
	}
	_, err = c.CompleteTask(ctx, task2.ID)
	if err != nil {
		t.Fatalf("Failed to complete task 2: %v", err)
	}

	// Refresh spec
	spec, err = c.GetSpec(ctx, spec.ID)
	if err != nil {
		t.Fatalf("Failed to get spec: %v", err)
	}

	// Status should be done (all tasks completed)
	if spec.Status != "done" {
		t.Errorf("Expected status 'done', got %q", spec.Status)
	}
	if spec.TaskCount != 2 {
		t.Errorf("Expected task_count 2, got %d", spec.TaskCount)
	}
	if spec.DoneCount != 2 {
		t.Errorf("Expected done_count 2, got %d", spec.DoneCount)
	}
}

func TestE2E_Spec_ComputedStatus_Cancelled(t *testing.T) {
	suite := setupE2E(t)
	defer suite.cleanup()

	projectName := "spec-status-cancelled"
	suite.createProject(projectName)

	c := suite.getClient(projectName, "test-agent")
	ctx := context.Background()

	// Create a spec with tasks
	spec, err := c.CreateSpec(ctx, "Cancelled Spec", "")
	if err != nil {
		t.Fatalf("Failed to create spec: %v", err)
	}
	_, err = c.CreateTask(ctx, "Task 1", "", 2, "", spec.ID)
	if err != nil {
		t.Fatalf("Failed to create task: %v", err)
	}

	// Cancel the spec
	spec, err = c.CancelSpec(ctx, spec.ID)
	if err != nil {
		t.Fatalf("Failed to cancel spec: %v", err)
	}

	// Status should be cancelled (overrides computed status)
	if spec.Status != "cancelled" {
		t.Errorf("Expected status 'cancelled', got %q", spec.Status)
	}
}

func TestE2E_Spec_Reopen(t *testing.T) {
	suite := setupE2E(t)
	defer suite.cleanup()

	projectName := "spec-reopen"
	suite.createProject(projectName)

	c := suite.getClient(projectName, "test-agent")
	ctx := context.Background()

	// Create a spec with a task
	spec, err := c.CreateSpec(ctx, "Reopen Spec", "")
	if err != nil {
		t.Fatalf("Failed to create spec: %v", err)
	}
	_, err = c.CreateTask(ctx, "Task 1", "", 2, "", spec.ID)
	if err != nil {
		t.Fatalf("Failed to create task: %v", err)
	}

	// Cancel the spec
	spec, err = c.CancelSpec(ctx, spec.ID)
	if err != nil {
		t.Fatalf("Failed to cancel spec: %v", err)
	}
	if spec.Status != "cancelled" {
		t.Errorf("Expected status 'cancelled', got %q", spec.Status)
	}

	// Reopen the spec
	spec, err = c.ReopenSpec(ctx, spec.ID)
	if err != nil {
		t.Fatalf("Failed to reopen spec: %v", err)
	}

	// Status should revert to active (has incomplete tasks)
	if spec.Status != "active" {
		t.Errorf("Expected status 'active' after reopen, got %q", spec.Status)
	}
}

func TestE2E_Spec_StatusTransition_ActiveToDone(t *testing.T) {
	suite := setupE2E(t)
	defer suite.cleanup()

	projectName := "spec-transition"
	suite.createProject(projectName)

	c := suite.getClient(projectName, "test-agent")
	ctx := context.Background()

	// Create a spec
	spec, err := c.CreateSpec(ctx, "Transition Spec", "")
	if err != nil {
		t.Fatalf("Failed to create spec: %v", err)
	}

	// Verify initial status is draft
	if spec.Status != "draft" {
		t.Errorf("Expected initial status 'draft', got %q", spec.Status)
	}

	// Add a task -> status becomes active
	task, err := c.CreateTask(ctx, "Only Task", "", 2, "", spec.ID)
	if err != nil {
		t.Fatalf("Failed to create task: %v", err)
	}
	spec, _ = c.GetSpec(ctx, spec.ID)
	if spec.Status != "active" {
		t.Errorf("Expected status 'active' after adding task, got %q", spec.Status)
	}

	// Complete the task -> status becomes done
	_, _ = c.ClaimTask(ctx, task.ID)
	_, _ = c.CompleteTask(ctx, task.ID)
	spec, _ = c.GetSpec(ctx, spec.ID)
	if spec.Status != "done" {
		t.Errorf("Expected status 'done' after completing all tasks, got %q", spec.Status)
	}
}

// ========================
// Spec Dependency Tests
// ========================

func TestE2E_SpecDependency_Add(t *testing.T) {
	suite := setupE2E(t)
	defer suite.cleanup()

	projectName := "spec-dep-add"
	suite.createProject(projectName)

	c := suite.getClient(projectName, "test-agent")
	ctx := context.Background()

	// Create two specs
	parent, err := c.CreateSpec(ctx, "Parent Spec", "")
	if err != nil {
		t.Fatalf("Failed to create parent spec: %v", err)
	}
	child, err := c.CreateSpec(ctx, "Child Spec", "")
	if err != nil {
		t.Fatalf("Failed to create child spec: %v", err)
	}

	// Add dependency: child depends on parent
	err = c.AddSpecDependency(ctx, child.ID, parent.ID)
	if err != nil {
		t.Fatalf("Failed to add spec dependency: %v", err)
	}

	// Verify dependency exists
	deps, err := c.ListSpecDependencies(ctx, child.ID)
	if err != nil {
		t.Fatalf("Failed to list spec dependencies: %v", err)
	}
	if len(deps) != 1 {
		t.Errorf("Expected 1 dependency, got %d", len(deps))
	}
	if deps[0].ParentID != parent.ID {
		t.Errorf("Expected parent ID %q, got %q", parent.ID, deps[0].ParentID)
	}
}

func TestE2E_SpecDependency_Remove(t *testing.T) {
	suite := setupE2E(t)
	defer suite.cleanup()

	projectName := "spec-dep-remove"
	suite.createProject(projectName)

	c := suite.getClient(projectName, "test-agent")
	ctx := context.Background()

	// Create two specs with a dependency
	parent, _ := c.CreateSpec(ctx, "Parent", "")
	child, _ := c.CreateSpec(ctx, "Child", "")
	c.AddSpecDependency(ctx, child.ID, parent.ID)

	// Remove the dependency
	err := c.RemoveSpecDependency(ctx, child.ID, parent.ID)
	if err != nil {
		t.Fatalf("Failed to remove spec dependency: %v", err)
	}

	// Verify dependency is removed
	deps, err := c.ListSpecDependencies(ctx, child.ID)
	if err != nil {
		t.Fatalf("Failed to list spec dependencies: %v", err)
	}
	if len(deps) != 0 {
		t.Errorf("Expected 0 dependencies after removal, got %d", len(deps))
	}
}

func TestE2E_SpecDependency_CycleDetection_Direct(t *testing.T) {
	suite := setupE2E(t)
	defer suite.cleanup()

	projectName := "spec-cycle-direct"
	suite.createProject(projectName)

	c := suite.getClient(projectName, "test-agent")
	ctx := context.Background()

	// Create two specs
	specA, _ := c.CreateSpec(ctx, "Spec A", "")
	specB, _ := c.CreateSpec(ctx, "Spec B", "")

	// A depends on B
	err := c.AddSpecDependency(ctx, specA.ID, specB.ID)
	if err != nil {
		t.Fatalf("Failed to add A->B dependency: %v", err)
	}

	// Try to make B depend on A (would create cycle: A->B->A)
	err = c.AddSpecDependency(ctx, specB.ID, specA.ID)
	if err == nil {
		t.Fatal("Expected error when creating cycle, got nil")
	}

	var domainErr *domain.DomainError
	if !errors.As(err, &domainErr) || domainErr.Code != domain.ErrCodeCycleDetected {
		t.Errorf("Expected CYCLE_DETECTED error, got %v", err)
	}
}

func TestE2E_SpecDependency_CycleDetection_Indirect(t *testing.T) {
	suite := setupE2E(t)
	defer suite.cleanup()

	projectName := "spec-cycle-indirect"
	suite.createProject(projectName)

	c := suite.getClient(projectName, "test-agent")
	ctx := context.Background()

	// Create three specs: A -> B -> C
	specA, _ := c.CreateSpec(ctx, "Spec A", "")
	specB, _ := c.CreateSpec(ctx, "Spec B", "")
	specC, _ := c.CreateSpec(ctx, "Spec C", "")

	// A depends on B
	err := c.AddSpecDependency(ctx, specA.ID, specB.ID)
	if err != nil {
		t.Fatalf("Failed to add A->B dependency: %v", err)
	}

	// B depends on C
	err = c.AddSpecDependency(ctx, specB.ID, specC.ID)
	if err != nil {
		t.Fatalf("Failed to add B->C dependency: %v", err)
	}

	// Try to make C depend on A (would create cycle: A->B->C->A)
	err = c.AddSpecDependency(ctx, specC.ID, specA.ID)
	if err == nil {
		t.Fatal("Expected error when creating indirect cycle, got nil")
	}

	var domainErr *domain.DomainError
	if !errors.As(err, &domainErr) || domainErr.Code != domain.ErrCodeCycleDetected {
		t.Errorf("Expected CYCLE_DETECTED error, got %v", err)
	}
}

func TestE2E_SpecDependency_SelfReference(t *testing.T) {
	suite := setupE2E(t)
	defer suite.cleanup()

	projectName := "spec-self-ref"
	suite.createProject(projectName)

	c := suite.getClient(projectName, "test-agent")
	ctx := context.Background()

	// Create a spec
	spec, _ := c.CreateSpec(ctx, "Self Ref Spec", "")

	// Try to make spec depend on itself
	err := c.AddSpecDependency(ctx, spec.ID, spec.ID)
	if err == nil {
		t.Fatal("Expected error when creating self-reference, got nil")
	}
}

func TestE2E_SpecDependency_ListReady(t *testing.T) {
	suite := setupE2E(t)
	defer suite.cleanup()

	projectName := "spec-ready"
	suite.createProject(projectName)

	c := suite.getClient(projectName, "test-agent")
	ctx := context.Background()

	// Create three specs
	specA, _ := c.CreateSpec(ctx, "Spec A (no deps)", "")
	specB, _ := c.CreateSpec(ctx, "Spec B (no deps)", "")
	specC, _ := c.CreateSpec(ctx, "Spec C (depends on A)", "")

	// C depends on A
	c.AddSpecDependency(ctx, specC.ID, specA.ID)

	// List ready specs (specs with no unfinished dependencies)
	// A and B are ready, C is blocked by A
	ready, err := c.ListReadySpecs(ctx, 1, 50)
	if err != nil {
		t.Fatalf("Failed to list ready specs: %v", err)
	}

	// A and B should be ready (draft status, no blocking deps)
	readyIDs := make(map[string]bool)
	for _, s := range ready.Data {
		readyIDs[s.ID] = true
	}

	if !readyIDs[specA.ID] {
		t.Errorf("Expected Spec A to be ready")
	}
	if !readyIDs[specB.ID] {
		t.Errorf("Expected Spec B to be ready")
	}
	if readyIDs[specC.ID] {
		t.Errorf("Expected Spec C to NOT be ready (blocked by A)")
	}
}

// ========================
// Spec-Task Relationship Tests
// ========================

func TestE2E_Spec_ListTasks(t *testing.T) {
	suite := setupE2E(t)
	defer suite.cleanup()

	projectName := "spec-list-tasks"
	suite.createProject(projectName)

	c := suite.getClient(projectName, "test-agent")
	ctx := context.Background()

	// Create a spec
	spec, err := c.CreateSpec(ctx, "Multi-task Spec", "")
	if err != nil {
		t.Fatalf("Failed to create spec: %v", err)
	}

	// Create tasks under the spec
	_, err = c.CreateTask(ctx, "Task 1", "", 2, "", spec.ID)
	if err != nil {
		t.Fatalf("Failed to create task 1: %v", err)
	}
	_, err = c.CreateTask(ctx, "Task 2", "", 2, "", spec.ID)
	if err != nil {
		t.Fatalf("Failed to create task 2: %v", err)
	}
	_, err = c.CreateTask(ctx, "Task 3", "", 2, "", spec.ID)
	if err != nil {
		t.Fatalf("Failed to create task 3: %v", err)
	}

	// Also create a task NOT under the spec
	_, err = c.CreateTask(ctx, "Unrelated Task", "", 2, "", "")
	if err != nil {
		t.Fatalf("Failed to create unrelated task: %v", err)
	}

	// List tasks in the spec
	tasks, err := c.ListSpecTasks(ctx, spec.ID, 1, 50)
	if err != nil {
		t.Fatalf("Failed to list spec tasks: %v", err)
	}

	if len(tasks.Data) != 3 {
		t.Errorf("Expected 3 tasks in spec, got %d", len(tasks.Data))
	}

	// Verify all tasks belong to the spec
	for _, task := range tasks.Data {
		if task.SpecID == nil || *task.SpecID != spec.ID {
			t.Errorf("Task %s has wrong spec_id: %v", task.ID, task.SpecID)
		}
	}
}

func TestE2E_Spec_DeleteCascadesTasks(t *testing.T) {
	suite := setupE2E(t)
	defer suite.cleanup()

	projectName := "spec-cascade"
	suite.createProject(projectName)

	c := suite.getClient(projectName, "test-agent")
	ctx := context.Background()

	// Create a spec with tasks
	spec, _ := c.CreateSpec(ctx, "Cascade Spec", "")
	task1, _ := c.CreateTask(ctx, "Task 1", "", 2, "", spec.ID)
	task2, _ := c.CreateTask(ctx, "Task 2", "", 2, "", spec.ID)

	// Delete the spec
	err := c.DeleteSpec(ctx, spec.ID)
	if err != nil {
		t.Fatalf("Failed to delete spec: %v", err)
	}

	// Verify tasks still exist but have no spec_id (ON DELETE SET NULL)
	t1, err := c.GetTask(ctx, task1.ID)
	if err != nil {
		t.Fatalf("Task 1 should still exist: %v", err)
	}
	if t1.SpecID != nil {
		t.Errorf("Task 1 spec_id should be nil after spec deletion, got %v", t1.SpecID)
	}

	t2, err := c.GetTask(ctx, task2.ID)
	if err != nil {
		t.Fatalf("Task 2 should still exist: %v", err)
	}
	if t2.SpecID != nil {
		t.Errorf("Task 2 spec_id should be nil after spec deletion, got %v", t2.SpecID)
	}
}

func TestE2E_Spec_ListByStatus(t *testing.T) {
	suite := setupE2E(t)
	defer suite.cleanup()

	projectName := "spec-list-status"
	suite.createProject(projectName)

	c := suite.getClient(projectName, "test-agent")
	ctx := context.Background()

	// Create specs with different statuses
	draftSpec, _ := c.CreateSpec(ctx, "Draft Spec", "")

	activeSpec, _ := c.CreateSpec(ctx, "Active Spec", "")
	_, _ = c.CreateTask(ctx, "Task", "", 2, "", activeSpec.ID)

	doneSpec, _ := c.CreateSpec(ctx, "Done Spec", "")
	task, _ := c.CreateTask(ctx, "Done Task", "", 2, "", doneSpec.ID)
	c.ClaimTask(ctx, task.ID)
	c.CompleteTask(ctx, task.ID)

	cancelledSpec, _ := c.CreateSpec(ctx, "Cancelled Spec", "")
	c.CancelSpec(ctx, cancelledSpec.ID)

	// List by status
	testCases := []struct {
		status   string
		expected []string
	}{
		{"draft", []string{draftSpec.ID}},
		{"active", []string{activeSpec.ID}},
		{"done", []string{doneSpec.ID}},
		{"cancelled", []string{cancelledSpec.ID}},
	}

	for _, tc := range testCases {
		list, err := c.ListSpecs(ctx, tc.status, 1, 50)
		if err != nil {
			t.Fatalf("Failed to list specs by status %q: %v", tc.status, err)
		}

		if len(list.Data) != len(tc.expected) {
			t.Errorf("Status %q: expected %d specs, got %d", tc.status, len(tc.expected), len(list.Data))
		}

		for _, s := range list.Data {
			if s.Status != tc.status {
				t.Errorf("Status filter %q: got spec with status %q", tc.status, s.Status)
			}
		}
	}
}
