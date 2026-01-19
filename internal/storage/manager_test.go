package storage

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// setupTestManager creates a manager with a temporary directory for testing
func setupTestManager(t *testing.T) (*ProjectDBManager, string) {
	t.Helper()
	tmpDir, err := os.MkdirTemp("", "airyra-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	t.Cleanup(func() {
		os.RemoveAll(tmpDir)
	})

	manager, err := NewProjectDBManager(tmpDir)
	if err != nil {
		t.Fatalf("failed to create manager: %v", err)
	}
	t.Cleanup(func() {
		manager.CloseAll()
	})

	return manager, tmpDir
}

func TestGetStore_CreatesDatabase(t *testing.T) {
	manager, tmpDir := setupTestManager(t)

	t.Run("creates database file on first access", func(t *testing.T) {
		store, err := manager.GetStore("test-project")
		if err != nil {
			t.Fatalf("expected no error, got: %v", err)
		}
		if store == nil {
			t.Fatal("expected store to be non-nil")
		}

		// Verify database file exists
		dbPath := filepath.Join(tmpDir, "test-project.db")
		if _, err := os.Stat(dbPath); os.IsNotExist(err) {
			t.Errorf("expected database file to exist at %s", dbPath)
		}
	})

	t.Run("creates projects directory if not exists", func(t *testing.T) {
		newDir := filepath.Join(tmpDir, "subdir", "projects")
		newManager, err := NewProjectDBManager(newDir)
		if err != nil {
			t.Fatalf("expected no error, got: %v", err)
		}
		defer newManager.CloseAll()

		_, err = newManager.GetStore("new-project")
		if err != nil {
			t.Fatalf("expected no error, got: %v", err)
		}

		if _, err := os.Stat(newDir); os.IsNotExist(err) {
			t.Errorf("expected directory to be created at %s", newDir)
		}
	})
}

func TestGetStore_ReusesConnection(t *testing.T) {
	manager, _ := setupTestManager(t)

	t.Run("returns same store instance for same project", func(t *testing.T) {
		store1, err := manager.GetStore("reuse-project")
		if err != nil {
			t.Fatalf("expected no error, got: %v", err)
		}

		store2, err := manager.GetStore("reuse-project")
		if err != nil {
			t.Fatalf("expected no error, got: %v", err)
		}

		// Should be the same instance
		if store1 != store2 {
			t.Error("expected same store instance for same project")
		}
	})

	t.Run("returns different store instances for different projects", func(t *testing.T) {
		store1, err := manager.GetStore("project-a")
		if err != nil {
			t.Fatalf("expected no error, got: %v", err)
		}

		store2, err := manager.GetStore("project-b")
		if err != nil {
			t.Fatalf("expected no error, got: %v", err)
		}

		// Should be different instances
		if store1 == store2 {
			t.Error("expected different store instances for different projects")
		}
	})
}

func TestListProjects(t *testing.T) {
	manager, tmpDir := setupTestManager(t)

	t.Run("returns empty list when no projects exist", func(t *testing.T) {
		projects, err := manager.ListProjects()
		if err != nil {
			t.Fatalf("expected no error, got: %v", err)
		}
		if len(projects) != 0 {
			t.Errorf("expected empty list, got %d projects", len(projects))
		}
	})

	t.Run("lists created projects", func(t *testing.T) {
		// Create some projects
		manager.GetStore("alpha")
		manager.GetStore("beta")
		manager.GetStore("gamma")

		projects, err := manager.ListProjects()
		if err != nil {
			t.Fatalf("expected no error, got: %v", err)
		}
		if len(projects) != 3 {
			t.Errorf("expected 3 projects, got %d", len(projects))
		}

		// Verify project names
		names := make(map[string]bool)
		for _, p := range projects {
			names[p] = true
		}
		if !names["alpha"] || !names["beta"] || !names["gamma"] {
			t.Errorf("expected projects alpha, beta, gamma; got %v", projects)
		}
	})

	t.Run("ignores non-db files", func(t *testing.T) {
		// Create a non-db file
		nonDBFile := filepath.Join(tmpDir, "not-a-database.txt")
		os.WriteFile(nonDBFile, []byte("test"), 0644)

		// Create a directory
		subDir := filepath.Join(tmpDir, "subdirectory")
		os.MkdirAll(subDir, 0755)

		projects, err := manager.ListProjects()
		if err != nil {
			t.Fatalf("expected no error, got: %v", err)
		}

		for _, p := range projects {
			if p == "not-a-database" || p == "subdirectory" {
				t.Errorf("unexpected project name: %s", p)
			}
		}
	})
}

func TestCloseAll(t *testing.T) {
	manager, _ := setupTestManager(t)

	// Create multiple stores
	store1, _ := manager.GetStore("close-test-1")
	store2, _ := manager.GetStore("close-test-2")

	err := manager.CloseAll()
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	// Verify stores are closed by trying to use them
	// (they should return errors)
	_, err = store1.Tasks().Get(nil, "test")
	if err == nil {
		t.Error("expected error after close, got nil")
	}

	_, err = store2.Tasks().Get(nil, "test")
	if err == nil {
		t.Error("expected error after close, got nil")
	}
}

func TestManagerConcurrentAccess(t *testing.T) {
	manager, _ := setupTestManager(t)

	// Test concurrent access to the same project
	t.Run("handles concurrent GetStore calls", func(t *testing.T) {
		done := make(chan bool)
		errs := make(chan error, 10)

		for i := 0; i < 10; i++ {
			go func() {
				store, err := manager.GetStore("concurrent-project")
				if err != nil {
					errs <- err
				}
				if store == nil {
					errs <- err
				}
				done <- true
			}()
		}

		for i := 0; i < 10; i++ {
			<-done
		}

		select {
		case err := <-errs:
			t.Errorf("got error during concurrent access: %v", err)
		default:
			// No errors, test passed
		}
	})
}

func TestProjectDBManager_SchemaInitialization(t *testing.T) {
	manager, _ := setupTestManager(t)

	t.Run("initializes schema on new database", func(t *testing.T) {
		store, err := manager.GetStore("schema-test")
		if err != nil {
			t.Fatalf("expected no error, got: %v", err)
		}

		// Verify schema by creating a task
		ctx := testContext()
		now := testTime()
		task := &Task{
			ID:        "ar-sch1",
			Title:     "Schema Test Task",
			Status:    StatusOpen,
			Priority:  PriorityMedium,
			CreatedAt: now,
			UpdatedAt: now,
		}

		err = store.Tasks().Create(ctx, task)
		if err != nil {
			t.Fatalf("expected no error creating task, got: %v", err)
		}

		got, err := store.Tasks().Get(ctx, "ar-sch1")
		if err != nil {
			t.Fatalf("expected no error getting task, got: %v", err)
		}
		if got.Title != "Schema Test Task" {
			t.Errorf("expected title 'Schema Test Task', got '%s'", got.Title)
		}
	})
}

// Helper functions for manager tests
func testContext() context.Context {
	return context.Background()
}

func testTime() time.Time {
	return time.Now().UTC().Truncate(time.Second)
}
