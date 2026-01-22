package storage

import (
	"database/sql"
	"testing"

	_ "github.com/mattn/go-sqlite3"
)

// setupRawDB creates an in-memory SQLite database without any schema
func setupRawDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite3", ":memory:?_foreign_keys=on")
	if err != nil {
		t.Fatalf("failed to create test database: %v", err)
	}
	t.Cleanup(func() {
		db.Close()
	})
	return db
}

func TestRunMigrations_FreshDatabase(t *testing.T) {
	db := setupRawDB(t)

	err := RunMigrations(db)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	// Verify schema version is set
	version, err := GetCurrentVersion(db)
	if err != nil {
		t.Fatalf("failed to get version: %v", err)
	}
	if version != 2 {
		t.Errorf("expected version 2, got %d", version)
	}

	// Verify tables were created
	tables := []string{"tasks", "dependencies", "audit_log", "_migrations", "specs", "spec_dependencies"}
	for _, table := range tables {
		var name string
		err := db.QueryRow(`
			SELECT name FROM sqlite_master
			WHERE type='table' AND name=?
		`, table).Scan(&name)
		if err != nil {
			t.Errorf("expected table %s to exist, got error: %v", table, err)
		}
	}
}

func TestRunMigrations_Idempotent(t *testing.T) {
	db := setupRawDB(t)

	// Run migrations first time
	err := RunMigrations(db)
	if err != nil {
		t.Fatalf("first migration run failed: %v", err)
	}

	// Run migrations second time - should be idempotent
	err = RunMigrations(db)
	if err != nil {
		t.Fatalf("second migration run failed: %v", err)
	}

	// Verify version is still 2
	version, err := GetCurrentVersion(db)
	if err != nil {
		t.Fatalf("failed to get version: %v", err)
	}
	if version != 2 {
		t.Errorf("expected version 2, got %d", version)
	}
}

func TestGetCurrentVersion_FreshDatabase(t *testing.T) {
	db := setupRawDB(t)

	version, err := GetCurrentVersion(db)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if version != 0 {
		t.Errorf("expected version 0 for fresh database, got %d", version)
	}
}

func TestGetCurrentVersion_AfterMigration(t *testing.T) {
	db := setupRawDB(t)

	err := RunMigrations(db)
	if err != nil {
		t.Fatalf("migration failed: %v", err)
	}

	version, err := GetCurrentVersion(db)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if version != 2 {
		t.Errorf("expected version 2, got %d", version)
	}
}

func TestGetCurrentVersion_EmptyMigrationsTable(t *testing.T) {
	db := setupRawDB(t)

	// Create _migrations table but don't insert any rows
	_, err := db.Exec(`
		CREATE TABLE _migrations (
			version INTEGER PRIMARY KEY,
			applied_at TEXT NOT NULL
		)
	`)
	if err != nil {
		t.Fatalf("failed to create _migrations table: %v", err)
	}

	version, err := GetCurrentVersion(db)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if version != 0 {
		t.Errorf("expected version 0 for empty _migrations table, got %d", version)
	}
}

func TestExtractVersion(t *testing.T) {
	tests := []struct {
		filename    string
		wantVersion int
		wantErr     bool
	}{
		{"001_initial_schema.sql", 1, false},
		{"002_add_feature.sql", 2, false},
		{"010_another_migration.sql", 10, false},
		{"100_big_version.sql", 100, false},
		{"invalid.sql", 0, true},
		{"abc_invalid.sql", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.filename, func(t *testing.T) {
			version, err := extractVersion(tt.filename)
			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error for filename %s", tt.filename)
				}
				return
			}
			if err != nil {
				t.Errorf("unexpected error for filename %s: %v", tt.filename, err)
				return
			}
			if version != tt.wantVersion {
				t.Errorf("expected version %d, got %d", tt.wantVersion, version)
			}
		})
	}
}

func TestMigrations_CreateTablesWithCorrectStructure(t *testing.T) {
	db := setupRawDB(t)

	err := RunMigrations(db)
	if err != nil {
		t.Fatalf("migration failed: %v", err)
	}

	t.Run("tasks table has correct columns", func(t *testing.T) {
		// Insert a task to verify columns exist
		_, err := db.Exec(`
			INSERT INTO tasks (id, title, status, priority, created_at, updated_at)
			VALUES ('ar-test', 'Test Task', 'open', 2, datetime('now'), datetime('now'))
		`)
		if err != nil {
			t.Errorf("failed to insert into tasks: %v", err)
		}

		// Verify we can read it back with all expected columns
		var id, title, status string
		var priority int
		err = db.QueryRow(`
			SELECT id, title, status, priority FROM tasks WHERE id = 'ar-test'
		`).Scan(&id, &title, &status, &priority)
		if err != nil {
			t.Errorf("failed to query tasks: %v", err)
		}
	})

	t.Run("dependencies table has correct columns", func(t *testing.T) {
		// Create parent task first
		_, err := db.Exec(`
			INSERT INTO tasks (id, title, status, priority, created_at, updated_at)
			VALUES ('ar-parent', 'Parent', 'open', 2, datetime('now'), datetime('now'))
		`)
		if err != nil {
			t.Errorf("failed to insert parent task: %v", err)
		}

		// Insert dependency
		_, err = db.Exec(`
			INSERT INTO dependencies (child_id, parent_id)
			VALUES ('ar-test', 'ar-parent')
		`)
		if err != nil {
			t.Errorf("failed to insert into dependencies: %v", err)
		}
	})

	t.Run("audit_log table has correct columns", func(t *testing.T) {
		_, err := db.Exec(`
			INSERT INTO audit_log (task_id, action, changed_at, changed_by)
			VALUES ('ar-test', 'create', datetime('now'), 'agent-001')
		`)
		if err != nil {
			t.Errorf("failed to insert into audit_log: %v", err)
		}
	})

	t.Run("indexes exist", func(t *testing.T) {
		indexes := []string{
			"idx_tasks_status",
			"idx_tasks_parent",
			"idx_tasks_priority",
			"idx_deps_parent",
			"idx_audit_task",
			"idx_audit_time",
		}

		for _, idx := range indexes {
			var name string
			err := db.QueryRow(`
				SELECT name FROM sqlite_master
				WHERE type='index' AND name=?
			`, idx).Scan(&name)
			if err != nil {
				t.Errorf("expected index %s to exist, got error: %v", idx, err)
			}
		}
	})
}

func TestMigrations_VersionTracking(t *testing.T) {
	db := setupRawDB(t)

	err := RunMigrations(db)
	if err != nil {
		t.Fatalf("migration failed: %v", err)
	}

	// Verify _migrations table has the correct entry
	var version int
	var appliedAt string
	err = db.QueryRow("SELECT version, applied_at FROM _migrations WHERE version = 1").
		Scan(&version, &appliedAt)
	if err != nil {
		t.Fatalf("failed to query _migrations: %v", err)
	}

	if version != 1 {
		t.Errorf("expected version 1 in _migrations, got %d", version)
	}
	if appliedAt == "" {
		t.Error("expected applied_at to be set")
	}
}

func TestLoadMigrations(t *testing.T) {
	migrations, err := loadMigrations()
	if err != nil {
		t.Fatalf("failed to load migrations: %v", err)
	}

	if len(migrations) == 0 {
		t.Fatal("expected at least one migration")
	}

	// Verify the initial migration is present
	found := false
	for _, m := range migrations {
		if m.Version == 1 {
			found = true
			if m.Name != "001_initial_schema.sql" {
				t.Errorf("expected migration name '001_initial_schema.sql', got '%s'", m.Name)
			}
			if m.SQL == "" {
				t.Error("expected migration SQL to be non-empty")
			}
			break
		}
	}

	if !found {
		t.Error("expected to find version 1 migration")
	}
}
