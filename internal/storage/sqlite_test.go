package storage

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// setupTestDB creates an in-memory SQLite database for testing
func setupTestDB(t *testing.T) *SQLiteStore {
	t.Helper()
	store, err := NewSQLiteStore(":memory:")
	if err != nil {
		t.Fatalf("failed to create test store: %v", err)
	}
	t.Cleanup(func() {
		store.Close()
	})
	return store
}

// createTestTask creates a task with sensible defaults for testing
func createTestTask(id, title string) *Task {
	now := time.Now().UTC().Truncate(time.Second)
	return &Task{
		ID:        id,
		Title:     title,
		Status:    StatusOpen,
		Priority:  PriorityMedium,
		CreatedAt: now,
		UpdatedAt: now,
	}
}

// ============================================================================
// Task Repository Tests
// ============================================================================

func TestTaskCreate(t *testing.T) {
	store := setupTestDB(t)
	ctx := context.Background()

	t.Run("creates task successfully", func(t *testing.T) {
		task := createTestTask("ar-0001", "Test Task")
		task.Description = StringPtr("A test description")

		err := store.Tasks().Create(ctx, task)
		if err != nil {
			t.Fatalf("expected no error, got: %v", err)
		}

		// Verify task was created
		got, err := store.Tasks().Get(ctx, "ar-0001")
		if err != nil {
			t.Fatalf("failed to get created task: %v", err)
		}
		if got.Title != "Test Task" {
			t.Errorf("expected title 'Test Task', got '%s'", got.Title)
		}
		if got.Description == nil || *got.Description != "A test description" {
			t.Errorf("expected description 'A test description', got %v", got.Description)
		}
		if got.Status != StatusOpen {
			t.Errorf("expected status 'open', got '%s'", got.Status)
		}
		if got.Priority != PriorityMedium {
			t.Errorf("expected priority %d, got %d", PriorityMedium, got.Priority)
		}
	})

	t.Run("creates task with parent", func(t *testing.T) {
		parent := createTestTask("ar-0002", "Parent Task")
		err := store.Tasks().Create(ctx, parent)
		if err != nil {
			t.Fatalf("failed to create parent task: %v", err)
		}

		child := createTestTask("ar-0003", "Child Task")
		child.ParentID = StringPtr("ar-0002")
		err = store.Tasks().Create(ctx, child)
		if err != nil {
			t.Fatalf("failed to create child task: %v", err)
		}

		got, err := store.Tasks().Get(ctx, "ar-0003")
		if err != nil {
			t.Fatalf("failed to get child task: %v", err)
		}
		if got.ParentID == nil || *got.ParentID != "ar-0002" {
			t.Errorf("expected parent_id 'ar-0002', got %v", got.ParentID)
		}
	})

	t.Run("fails on duplicate ID", func(t *testing.T) {
		task := createTestTask("ar-dup1", "First Task")
		err := store.Tasks().Create(ctx, task)
		if err != nil {
			t.Fatalf("failed to create first task: %v", err)
		}

		duplicate := createTestTask("ar-dup1", "Duplicate Task")
		err = store.Tasks().Create(ctx, duplicate)
		if err == nil {
			t.Error("expected error for duplicate ID, got nil")
		}
	})
}

func TestTaskGet(t *testing.T) {
	store := setupTestDB(t)
	ctx := context.Background()

	t.Run("returns task when exists", func(t *testing.T) {
		task := createTestTask("ar-get1", "Get Test Task")
		store.Tasks().Create(ctx, task)

		got, err := store.Tasks().Get(ctx, "ar-get1")
		if err != nil {
			t.Fatalf("expected no error, got: %v", err)
		}
		if got.ID != "ar-get1" {
			t.Errorf("expected ID 'ar-get1', got '%s'", got.ID)
		}
	})

	t.Run("returns ErrNotFound when task does not exist", func(t *testing.T) {
		_, err := store.Tasks().Get(ctx, "ar-noex")
		if !errors.Is(err, ErrNotFound) {
			t.Errorf("expected ErrNotFound, got: %v", err)
		}
	})
}

func TestTaskList(t *testing.T) {
	store := setupTestDB(t)
	ctx := context.Background()

	// Create test tasks
	tasks := []*Task{
		createTestTask("ar-lst1", "Task 1"),
		createTestTask("ar-lst2", "Task 2"),
		createTestTask("ar-lst3", "Task 3"),
	}
	tasks[0].Status = StatusOpen
	tasks[1].Status = StatusInProgress
	tasks[1].ClaimedBy = StringPtr("agent-1")
	tasks[1].ClaimedAt = TimePtr(time.Now().UTC())
	tasks[2].Status = StatusDone

	for _, task := range tasks {
		if err := store.Tasks().Create(ctx, task); err != nil {
			t.Fatalf("failed to create task: %v", err)
		}
	}

	t.Run("lists all tasks", func(t *testing.T) {
		got, total, err := store.Tasks().List(ctx, ListOptions{})
		if err != nil {
			t.Fatalf("expected no error, got: %v", err)
		}
		if total < 3 {
			t.Errorf("expected at least 3 tasks, got %d", total)
		}
		if len(got) < 3 {
			t.Errorf("expected at least 3 tasks returned, got %d", len(got))
		}
	})

	t.Run("filters by status", func(t *testing.T) {
		status := StatusOpen
		got, _, err := store.Tasks().List(ctx, ListOptions{Status: &status})
		if err != nil {
			t.Fatalf("expected no error, got: %v", err)
		}
		for _, task := range got {
			if task.Status != StatusOpen {
				t.Errorf("expected only open tasks, got status '%s'", task.Status)
			}
		}
	})

	t.Run("supports pagination", func(t *testing.T) {
		got, total, err := store.Tasks().List(ctx, ListOptions{Page: 1, PerPage: 2})
		if err != nil {
			t.Fatalf("expected no error, got: %v", err)
		}
		if len(got) > 2 {
			t.Errorf("expected at most 2 tasks, got %d", len(got))
		}
		if total < 3 {
			t.Errorf("expected total >= 3, got %d", total)
		}
	})
}

func TestTaskUpdate(t *testing.T) {
	store := setupTestDB(t)
	ctx := context.Background()

	t.Run("updates task successfully", func(t *testing.T) {
		task := createTestTask("ar-upd1", "Original Title")
		store.Tasks().Create(ctx, task)

		task.Title = "Updated Title"
		task.Priority = PriorityHigh
		task.UpdatedAt = time.Now().UTC().Truncate(time.Second)

		err := store.Tasks().Update(ctx, task)
		if err != nil {
			t.Fatalf("expected no error, got: %v", err)
		}

		got, _ := store.Tasks().Get(ctx, "ar-upd1")
		if got.Title != "Updated Title" {
			t.Errorf("expected title 'Updated Title', got '%s'", got.Title)
		}
		if got.Priority != PriorityHigh {
			t.Errorf("expected priority %d, got %d", PriorityHigh, got.Priority)
		}
	})

	t.Run("returns ErrNotFound for non-existent task", func(t *testing.T) {
		task := createTestTask("ar-updn", "Non-existent")
		err := store.Tasks().Update(ctx, task)
		if !errors.Is(err, ErrNotFound) {
			t.Errorf("expected ErrNotFound, got: %v", err)
		}
	})
}

func TestTaskDelete(t *testing.T) {
	store := setupTestDB(t)
	ctx := context.Background()

	t.Run("deletes task successfully", func(t *testing.T) {
		task := createTestTask("ar-del1", "Task to Delete")
		store.Tasks().Create(ctx, task)

		err := store.Tasks().Delete(ctx, "ar-del1")
		if err != nil {
			t.Fatalf("expected no error, got: %v", err)
		}

		_, err = store.Tasks().Get(ctx, "ar-del1")
		if !errors.Is(err, ErrNotFound) {
			t.Errorf("expected ErrNotFound after delete, got: %v", err)
		}
	})

	t.Run("returns ErrNotFound for non-existent task", func(t *testing.T) {
		err := store.Tasks().Delete(ctx, "ar-noex")
		if !errors.Is(err, ErrNotFound) {
			t.Errorf("expected ErrNotFound, got: %v", err)
		}
	})
}

// ============================================================================
// Claim/Release/MarkDone Tests - Critical atomic operations
// ============================================================================

func TestTaskClaim_Success(t *testing.T) {
	store := setupTestDB(t)
	ctx := context.Background()

	task := createTestTask("ar-clm1", "Claimable Task")
	store.Tasks().Create(ctx, task)

	err := store.Tasks().Claim(ctx, "ar-clm1", "agent-001")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	got, _ := store.Tasks().Get(ctx, "ar-clm1")
	if got.Status != StatusInProgress {
		t.Errorf("expected status '%s', got '%s'", StatusInProgress, got.Status)
	}
	if got.ClaimedBy == nil || *got.ClaimedBy != "agent-001" {
		t.Errorf("expected claimed_by 'agent-001', got %v", got.ClaimedBy)
	}
	if got.ClaimedAt == nil {
		t.Error("expected claimed_at to be set")
	}
}

func TestTaskClaim_AlreadyClaimed(t *testing.T) {
	store := setupTestDB(t)
	ctx := context.Background()

	// Create and claim task
	task := createTestTask("ar-clm2", "Already Claimed Task")
	store.Tasks().Create(ctx, task)
	store.Tasks().Claim(ctx, "ar-clm2", "agent-001")

	// Try to claim again with different agent
	err := store.Tasks().Claim(ctx, "ar-clm2", "agent-002")
	if !errors.Is(err, ErrAlreadyClaimed) {
		t.Errorf("expected ErrAlreadyClaimed, got: %v", err)
	}
}

func TestTaskClaim_NotOpenTask(t *testing.T) {
	store := setupTestDB(t)
	ctx := context.Background()

	// Create a done task
	task := createTestTask("ar-clm3", "Done Task")
	task.Status = StatusDone
	store.Tasks().Create(ctx, task)

	// Try to claim it
	err := store.Tasks().Claim(ctx, "ar-clm3", "agent-001")
	if !errors.Is(err, ErrInvalidTransition) {
		t.Errorf("expected ErrInvalidTransition for claiming done task, got: %v", err)
	}
}

func TestTaskClaim_NonExistent(t *testing.T) {
	store := setupTestDB(t)
	ctx := context.Background()

	err := store.Tasks().Claim(ctx, "ar-noex", "agent-001")
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("expected ErrNotFound, got: %v", err)
	}
}

func TestTaskRelease(t *testing.T) {
	store := setupTestDB(t)
	ctx := context.Background()

	t.Run("releases task successfully", func(t *testing.T) {
		task := createTestTask("ar-rel1", "Task to Release")
		store.Tasks().Create(ctx, task)
		store.Tasks().Claim(ctx, "ar-rel1", "agent-001")

		err := store.Tasks().Release(ctx, "ar-rel1", "agent-001")
		if err != nil {
			t.Fatalf("expected no error, got: %v", err)
		}

		got, _ := store.Tasks().Get(ctx, "ar-rel1")
		if got.Status != StatusOpen {
			t.Errorf("expected status '%s', got '%s'", StatusOpen, got.Status)
		}
		if got.ClaimedBy != nil {
			t.Errorf("expected claimed_by to be nil, got %v", got.ClaimedBy)
		}
		if got.ClaimedAt != nil {
			t.Errorf("expected claimed_at to be nil, got %v", got.ClaimedAt)
		}
	})

	t.Run("fails if not owner", func(t *testing.T) {
		task := createTestTask("ar-rel2", "Task with Different Owner")
		store.Tasks().Create(ctx, task)
		store.Tasks().Claim(ctx, "ar-rel2", "agent-001")

		err := store.Tasks().Release(ctx, "ar-rel2", "agent-002")
		if !errors.Is(err, ErrNotOwner) {
			t.Errorf("expected ErrNotOwner, got: %v", err)
		}
	})

	t.Run("fails if not in progress", func(t *testing.T) {
		task := createTestTask("ar-rel3", "Open Task")
		store.Tasks().Create(ctx, task)

		err := store.Tasks().Release(ctx, "ar-rel3", "agent-001")
		if !errors.Is(err, ErrInvalidTransition) {
			t.Errorf("expected ErrInvalidTransition, got: %v", err)
		}
	})
}

func TestTaskMarkDone(t *testing.T) {
	store := setupTestDB(t)
	ctx := context.Background()

	t.Run("marks task as done successfully", func(t *testing.T) {
		task := createTestTask("ar-don1", "Task to Complete")
		store.Tasks().Create(ctx, task)
		store.Tasks().Claim(ctx, "ar-don1", "agent-001")

		err := store.Tasks().MarkDone(ctx, "ar-don1", "agent-001")
		if err != nil {
			t.Fatalf("expected no error, got: %v", err)
		}

		got, _ := store.Tasks().Get(ctx, "ar-don1")
		if got.Status != StatusDone {
			t.Errorf("expected status '%s', got '%s'", StatusDone, got.Status)
		}
	})

	t.Run("fails if not owner", func(t *testing.T) {
		task := createTestTask("ar-don2", "Task with Different Owner")
		store.Tasks().Create(ctx, task)
		store.Tasks().Claim(ctx, "ar-don2", "agent-001")

		err := store.Tasks().MarkDone(ctx, "ar-don2", "agent-002")
		if !errors.Is(err, ErrNotOwner) {
			t.Errorf("expected ErrNotOwner, got: %v", err)
		}
	})

	t.Run("fails if not in progress", func(t *testing.T) {
		task := createTestTask("ar-don3", "Open Task")
		store.Tasks().Create(ctx, task)

		err := store.Tasks().MarkDone(ctx, "ar-don3", "agent-001")
		if !errors.Is(err, ErrInvalidTransition) {
			t.Errorf("expected ErrInvalidTransition, got: %v", err)
		}
	})
}

func TestTaskBlock(t *testing.T) {
	store := setupTestDB(t)
	ctx := context.Background()

	t.Run("blocks task successfully", func(t *testing.T) {
		task := createTestTask("ar-blk1", "Task to Block")
		store.Tasks().Create(ctx, task)

		err := store.Tasks().Block(ctx, "ar-blk1")
		if err != nil {
			t.Fatalf("expected no error, got: %v", err)
		}

		got, _ := store.Tasks().Get(ctx, "ar-blk1")
		if got.Status != StatusBlocked {
			t.Errorf("expected status '%s', got '%s'", StatusBlocked, got.Status)
		}
	})

	t.Run("returns ErrNotFound for non-existent task", func(t *testing.T) {
		err := store.Tasks().Block(ctx, "ar-noex")
		if !errors.Is(err, ErrNotFound) {
			t.Errorf("expected ErrNotFound, got: %v", err)
		}
	})
}

func TestTaskUnblock(t *testing.T) {
	store := setupTestDB(t)
	ctx := context.Background()

	t.Run("unblocks task successfully", func(t *testing.T) {
		task := createTestTask("ar-ubl1", "Blocked Task")
		task.Status = StatusBlocked
		store.Tasks().Create(ctx, task)

		err := store.Tasks().Unblock(ctx, "ar-ubl1")
		if err != nil {
			t.Fatalf("expected no error, got: %v", err)
		}

		got, _ := store.Tasks().Get(ctx, "ar-ubl1")
		if got.Status != StatusOpen {
			t.Errorf("expected status '%s', got '%s'", StatusOpen, got.Status)
		}
	})

	t.Run("fails if not blocked", func(t *testing.T) {
		task := createTestTask("ar-ubl2", "Open Task")
		store.Tasks().Create(ctx, task)

		err := store.Tasks().Unblock(ctx, "ar-ubl2")
		if !errors.Is(err, ErrInvalidTransition) {
			t.Errorf("expected ErrInvalidTransition, got: %v", err)
		}
	})
}

func TestListReady(t *testing.T) {
	store := setupTestDB(t)
	ctx := context.Background()

	// Create a parent task (done)
	parent := createTestTask("ar-rdy1", "Done Parent")
	parent.Status = StatusDone
	store.Tasks().Create(ctx, parent)

	// Create a task that depends on the done parent (should be ready)
	ready := createTestTask("ar-rdy2", "Ready Task")
	store.Tasks().Create(ctx, ready)
	store.Dependencies().Add(ctx, "ar-rdy2", "ar-rdy1")

	// Create a blocking parent (not done)
	blocker := createTestTask("ar-rdy3", "Blocking Parent")
	store.Tasks().Create(ctx, blocker)

	// Create a task that depends on the blocker (should NOT be ready)
	blocked := createTestTask("ar-rdy4", "Blocked Task")
	store.Tasks().Create(ctx, blocked)
	store.Dependencies().Add(ctx, "ar-rdy4", "ar-rdy3")

	// Create a task with no dependencies (should be ready)
	noDeps := createTestTask("ar-rdy5", "No Dependencies")
	store.Tasks().Create(ctx, noDeps)

	t.Run("returns only ready tasks", func(t *testing.T) {
		got, _, err := store.Tasks().ListReady(ctx, ListOptions{})
		if err != nil {
			t.Fatalf("expected no error, got: %v", err)
		}

		// Should include ar-rdy2 (deps done) and ar-rdy5 (no deps)
		// Should NOT include ar-rdy4 (deps not done)
		var readyIDs []string
		for _, task := range got {
			readyIDs = append(readyIDs, task.ID)
		}

		foundRdy2 := false
		foundRdy5 := false
		for _, id := range readyIDs {
			if id == "ar-rdy2" {
				foundRdy2 = true
			}
			if id == "ar-rdy5" {
				foundRdy5 = true
			}
			if id == "ar-rdy4" {
				t.Errorf("ar-rdy4 should not be ready (has incomplete dependency)")
			}
		}

		if !foundRdy2 {
			t.Error("ar-rdy2 should be ready (all deps done)")
		}
		if !foundRdy5 {
			t.Error("ar-rdy5 should be ready (no deps)")
		}
	})
}

// ============================================================================
// Dependency Repository Tests
// ============================================================================

func TestDependencyAdd(t *testing.T) {
	store := setupTestDB(t)
	ctx := context.Background()

	parent := createTestTask("ar-dep1", "Parent Task")
	child := createTestTask("ar-dep2", "Child Task")
	store.Tasks().Create(ctx, parent)
	store.Tasks().Create(ctx, child)

	t.Run("adds dependency successfully", func(t *testing.T) {
		err := store.Dependencies().Add(ctx, "ar-dep2", "ar-dep1")
		if err != nil {
			t.Fatalf("expected no error, got: %v", err)
		}

		deps, err := store.Dependencies().ListForTask(ctx, "ar-dep2")
		if err != nil {
			t.Fatalf("failed to list dependencies: %v", err)
		}

		found := false
		for _, dep := range deps {
			if dep.ChildID == "ar-dep2" && dep.ParentID == "ar-dep1" {
				found = true
				break
			}
		}
		if !found {
			t.Error("expected to find the added dependency")
		}
	})

	t.Run("fails on duplicate dependency", func(t *testing.T) {
		// Already added above
		err := store.Dependencies().Add(ctx, "ar-dep2", "ar-dep1")
		if err == nil {
			t.Error("expected error for duplicate dependency")
		}
	})
}

func TestDependencyRemove(t *testing.T) {
	store := setupTestDB(t)
	ctx := context.Background()

	parent := createTestTask("ar-drm1", "Parent Task")
	child := createTestTask("ar-drm2", "Child Task")
	store.Tasks().Create(ctx, parent)
	store.Tasks().Create(ctx, child)
	store.Dependencies().Add(ctx, "ar-drm2", "ar-drm1")

	t.Run("removes dependency successfully", func(t *testing.T) {
		err := store.Dependencies().Remove(ctx, "ar-drm2", "ar-drm1")
		if err != nil {
			t.Fatalf("expected no error, got: %v", err)
		}

		deps, _ := store.Dependencies().ListForTask(ctx, "ar-drm2")
		for _, dep := range deps {
			if dep.ChildID == "ar-drm2" && dep.ParentID == "ar-drm1" {
				t.Error("expected dependency to be removed")
			}
		}
	})
}

func TestDependencyCheckCycle(t *testing.T) {
	store := setupTestDB(t)
	ctx := context.Background()

	// Create tasks: A -> B -> C
	taskA := createTestTask("ar-cyc1", "Task A")
	taskB := createTestTask("ar-cyc2", "Task B")
	taskC := createTestTask("ar-cyc3", "Task C")
	store.Tasks().Create(ctx, taskA)
	store.Tasks().Create(ctx, taskB)
	store.Tasks().Create(ctx, taskC)

	// B depends on A (B -> A)
	store.Dependencies().Add(ctx, "ar-cyc2", "ar-cyc1")
	// C depends on B (C -> B)
	store.Dependencies().Add(ctx, "ar-cyc3", "ar-cyc2")

	t.Run("detects cycle when adding A depends on C", func(t *testing.T) {
		// Adding A -> C would create: A -> C -> B -> A (cycle)
		hasCycle, path, err := store.Dependencies().CheckCycle(ctx, "ar-cyc1", "ar-cyc3")
		if err != nil {
			t.Fatalf("expected no error, got: %v", err)
		}
		if !hasCycle {
			t.Error("expected cycle to be detected")
		}
		if len(path) == 0 {
			t.Error("expected cycle path to be returned")
		}
	})

	t.Run("no cycle for valid dependency", func(t *testing.T) {
		// D depends on C should be fine (no cycle)
		taskD := createTestTask("ar-cyc4", "Task D")
		store.Tasks().Create(ctx, taskD)

		hasCycle, _, err := store.Dependencies().CheckCycle(ctx, "ar-cyc4", "ar-cyc3")
		if err != nil {
			t.Fatalf("expected no error, got: %v", err)
		}
		if hasCycle {
			t.Error("expected no cycle for valid dependency")
		}
	})

	t.Run("Add fails on cycle", func(t *testing.T) {
		// Try to add the cyclic dependency
		err := store.Dependencies().Add(ctx, "ar-cyc1", "ar-cyc3")
		if !errors.Is(err, ErrCycleDetected) {
			t.Errorf("expected ErrCycleDetected, got: %v", err)
		}
	})
}

func TestDependencyListForTask(t *testing.T) {
	store := setupTestDB(t)
	ctx := context.Background()

	// Create tasks
	taskA := createTestTask("ar-dlf1", "Task A")
	taskB := createTestTask("ar-dlf2", "Task B")
	taskC := createTestTask("ar-dlf3", "Task C")
	store.Tasks().Create(ctx, taskA)
	store.Tasks().Create(ctx, taskB)
	store.Tasks().Create(ctx, taskC)

	// B depends on A
	// B depends on C
	store.Dependencies().Add(ctx, "ar-dlf2", "ar-dlf1")
	store.Dependencies().Add(ctx, "ar-dlf2", "ar-dlf3")

	t.Run("lists dependencies for task as child", func(t *testing.T) {
		deps, err := store.Dependencies().ListForTask(ctx, "ar-dlf2")
		if err != nil {
			t.Fatalf("expected no error, got: %v", err)
		}

		// B should have 2 dependencies (things it depends on)
		childDeps := 0
		for _, dep := range deps {
			if dep.ChildID == "ar-dlf2" {
				childDeps++
			}
		}
		if childDeps != 2 {
			t.Errorf("expected 2 child dependencies, got %d", childDeps)
		}
	})

	t.Run("lists dependencies for task as parent", func(t *testing.T) {
		deps, err := store.Dependencies().ListForTask(ctx, "ar-dlf1")
		if err != nil {
			t.Fatalf("expected no error, got: %v", err)
		}

		// A should appear as parent in 1 dependency
		parentDeps := 0
		for _, dep := range deps {
			if dep.ParentID == "ar-dlf1" {
				parentDeps++
			}
		}
		if parentDeps != 1 {
			t.Errorf("expected 1 parent dependency, got %d", parentDeps)
		}
	})
}

// ============================================================================
// Audit Repository Tests
// ============================================================================

func TestAuditLog(t *testing.T) {
	store := setupTestDB(t)
	ctx := context.Background()

	task := createTestTask("ar-aud1", "Task for Audit")
	store.Tasks().Create(ctx, task)

	t.Run("logs audit entry successfully", func(t *testing.T) {
		entry := &AuditEntry{
			TaskID:    "ar-aud1",
			Action:    ActionCreate,
			ChangedAt: time.Now().UTC(),
			ChangedBy: "agent-001",
		}

		err := store.AuditLogs().Log(ctx, entry)
		if err != nil {
			t.Fatalf("expected no error, got: %v", err)
		}

		entries, err := store.AuditLogs().ListForTask(ctx, "ar-aud1")
		if err != nil {
			t.Fatalf("failed to list audit entries: %v", err)
		}
		if len(entries) == 0 {
			t.Error("expected at least one audit entry")
		}
	})

	t.Run("logs entry with field changes", func(t *testing.T) {
		oldVal := "open"
		newVal := "in_progress"
		field := "status"
		entry := &AuditEntry{
			TaskID:    "ar-aud1",
			Action:    ActionUpdate,
			Field:     &field,
			OldValue:  &oldVal,
			NewValue:  &newVal,
			ChangedAt: time.Now().UTC(),
			ChangedBy: "agent-001",
		}

		err := store.AuditLogs().Log(ctx, entry)
		if err != nil {
			t.Fatalf("expected no error, got: %v", err)
		}

		entries, _ := store.AuditLogs().ListForTask(ctx, "ar-aud1")
		found := false
		for _, e := range entries {
			if e.Field != nil && *e.Field == "status" {
				found = true
				if e.OldValue == nil || *e.OldValue != "open" {
					t.Errorf("expected old value 'open', got %v", e.OldValue)
				}
				if e.NewValue == nil || *e.NewValue != "in_progress" {
					t.Errorf("expected new value 'in_progress', got %v", e.NewValue)
				}
			}
		}
		if !found {
			t.Error("expected to find audit entry with field 'status'")
		}
	})
}

func TestAuditListForTask(t *testing.T) {
	store := setupTestDB(t)
	ctx := context.Background()

	task := createTestTask("ar-alf1", "Task for Audit List")
	store.Tasks().Create(ctx, task)

	// Log multiple entries
	for i := 0; i < 5; i++ {
		entry := &AuditEntry{
			TaskID:    "ar-alf1",
			Action:    ActionUpdate,
			ChangedAt: time.Now().UTC(),
			ChangedBy: "agent-001",
		}
		store.AuditLogs().Log(ctx, entry)
	}

	t.Run("lists all entries for task", func(t *testing.T) {
		entries, err := store.AuditLogs().ListForTask(ctx, "ar-alf1")
		if err != nil {
			t.Fatalf("expected no error, got: %v", err)
		}
		if len(entries) < 5 {
			t.Errorf("expected at least 5 entries, got %d", len(entries))
		}
	})

	t.Run("returns empty for non-existent task", func(t *testing.T) {
		entries, err := store.AuditLogs().ListForTask(ctx, "ar-noex")
		if err != nil {
			t.Fatalf("expected no error, got: %v", err)
		}
		if len(entries) != 0 {
			t.Errorf("expected 0 entries, got %d", len(entries))
		}
	})
}

func TestAuditQuery(t *testing.T) {
	store := setupTestDB(t)
	ctx := context.Background()

	task := createTestTask("ar-aqr1", "Task for Audit Query")
	store.Tasks().Create(ctx, task)

	// Log entries with different actions and agents
	actions := []string{ActionCreate, ActionUpdate, ActionClaim, ActionRelease}
	agents := []string{"agent-001", "agent-002"}

	for _, action := range actions {
		for _, agent := range agents {
			entry := &AuditEntry{
				TaskID:    "ar-aqr1",
				Action:    action,
				ChangedAt: time.Now().UTC(),
				ChangedBy: agent,
			}
			store.AuditLogs().Log(ctx, entry)
		}
	}

	t.Run("filters by action", func(t *testing.T) {
		action := ActionClaim
		entries, err := store.AuditLogs().Query(ctx, AuditQueryOptions{Action: &action})
		if err != nil {
			t.Fatalf("expected no error, got: %v", err)
		}
		for _, e := range entries {
			if e.Action != ActionClaim {
				t.Errorf("expected action '%s', got '%s'", ActionClaim, e.Action)
			}
		}
	})

	t.Run("filters by changed_by", func(t *testing.T) {
		agent := "agent-002"
		entries, err := store.AuditLogs().Query(ctx, AuditQueryOptions{ChangedBy: &agent})
		if err != nil {
			t.Fatalf("expected no error, got: %v", err)
		}
		for _, e := range entries {
			if e.ChangedBy != "agent-002" {
				t.Errorf("expected changed_by 'agent-002', got '%s'", e.ChangedBy)
			}
		}
	})
}

// ============================================================================
// Transaction Tests
// ============================================================================

func TestWithTx(t *testing.T) {
	store := setupTestDB(t)
	ctx := context.Background()

	t.Run("commits on success", func(t *testing.T) {
		err := store.WithTx(ctx, func(tx TxStore) error {
			task := createTestTask("ar-tx01", "Transaction Task")
			return tx.Tasks().Create(ctx, task)
		})
		if err != nil {
			t.Fatalf("expected no error, got: %v", err)
		}

		// Verify task exists
		_, err = store.Tasks().Get(ctx, "ar-tx01")
		if err != nil {
			t.Errorf("expected task to exist after commit, got: %v", err)
		}
	})

	t.Run("rolls back on error", func(t *testing.T) {
		err := store.WithTx(ctx, func(tx TxStore) error {
			task := createTestTask("ar-tx02", "Rollback Task")
			if err := tx.Tasks().Create(ctx, task); err != nil {
				return err
			}
			return errors.New("intentional error")
		})
		if err == nil {
			t.Fatal("expected error, got nil")
		}

		// Verify task does NOT exist
		_, err = store.Tasks().Get(ctx, "ar-tx02")
		if !errors.Is(err, ErrNotFound) {
			t.Errorf("expected task to not exist after rollback, got: %v", err)
		}
	})

	t.Run("transaction operations are isolated", func(t *testing.T) {
		task := createTestTask("ar-tx03", "Isolation Task")
		store.Tasks().Create(ctx, task)

		err := store.WithTx(ctx, func(tx TxStore) error {
			// Update within transaction
			task.Title = "Updated in TX"
			task.UpdatedAt = time.Now().UTC()
			if err := tx.Tasks().Update(ctx, task); err != nil {
				return err
			}
			// Claim within transaction
			return tx.Tasks().Claim(ctx, "ar-tx03", "agent-001")
		})
		if err != nil {
			t.Fatalf("expected no error, got: %v", err)
		}

		got, _ := store.Tasks().Get(ctx, "ar-tx03")
		if got.Title != "Updated in TX" {
			t.Errorf("expected title 'Updated in TX', got '%s'", got.Title)
		}
		if got.Status != StatusInProgress {
			t.Errorf("expected status '%s', got '%s'", StatusInProgress, got.Status)
		}
	})
}

// ============================================================================
// Store Close Test
// ============================================================================

func TestStoreClose(t *testing.T) {
	store, err := NewSQLiteStore(":memory:")
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}

	err = store.Close()
	if err != nil {
		t.Errorf("expected no error on close, got: %v", err)
	}

	// Operations after close should fail
	_, err = store.Tasks().Get(context.Background(), "ar-0001")
	if err == nil {
		t.Error("expected error after close, got nil")
	}
}

// ============================================================================
// Helper function to verify raw DB access for testing atomic operations
// ============================================================================

func getTaskStatusDirectly(db *sql.DB, id string) (string, error) {
	var status string
	err := db.QueryRow("SELECT status FROM tasks WHERE id = ?", id).Scan(&status)
	return status, err
}
