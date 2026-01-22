package store

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	_ "github.com/mattn/go-sqlite3"
)

// initialSchema is the SQL schema for initializing a new project database.
const initialSchema = `
-- Enable WAL mode for better concurrent read performance
PRAGMA journal_mode=WAL;

-- Specs table
CREATE TABLE IF NOT EXISTS specs (
    id            TEXT PRIMARY KEY,
    title         TEXT NOT NULL,
    description   TEXT,
    manual_status TEXT CHECK (manual_status IS NULL OR manual_status = 'cancelled'),
    created_at    TEXT NOT NULL,
    updated_at    TEXT NOT NULL
);

-- Spec dependencies table
CREATE TABLE IF NOT EXISTS spec_dependencies (
    child_id  TEXT NOT NULL,
    parent_id TEXT NOT NULL,
    PRIMARY KEY (child_id, parent_id),
    CHECK (child_id != parent_id),
    FOREIGN KEY (child_id) REFERENCES specs(id) ON DELETE CASCADE,
    FOREIGN KEY (parent_id) REFERENCES specs(id) ON DELETE CASCADE
);

-- Tasks table
CREATE TABLE IF NOT EXISTS tasks (
    id          TEXT PRIMARY KEY,
    parent_id   TEXT REFERENCES tasks(id) ON DELETE CASCADE,
    spec_id     TEXT REFERENCES specs(id) ON DELETE SET NULL,
    title       TEXT NOT NULL,
    description TEXT,
    status      TEXT NOT NULL DEFAULT 'open'
                CHECK (status IN ('open', 'in_progress', 'blocked', 'done')),
    priority    INTEGER NOT NULL DEFAULT 2 CHECK (priority BETWEEN 0 AND 4),
    claimed_by  TEXT,
    claimed_at  TEXT,
    created_at  TEXT NOT NULL,
    updated_at  TEXT NOT NULL
);

-- Index for listing tasks by status
CREATE INDEX IF NOT EXISTS idx_tasks_status ON tasks(status);

-- Index for listing tasks by priority
CREATE INDEX IF NOT EXISTS idx_tasks_priority ON tasks(priority);

-- Index for finding subtasks
CREATE INDEX IF NOT EXISTS idx_tasks_parent_id ON tasks(parent_id);

-- Index for finding tasks in a spec
CREATE INDEX IF NOT EXISTS idx_tasks_spec_id ON tasks(spec_id);

-- Dependencies table (DAG edges)
CREATE TABLE IF NOT EXISTS dependencies (
    child_id  TEXT NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
    parent_id TEXT NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
    PRIMARY KEY (child_id, parent_id),
    CHECK (child_id != parent_id)
);

-- Index for finding what a task depends on
CREATE INDEX IF NOT EXISTS idx_dependencies_child ON dependencies(child_id);

-- Index for finding what depends on a task
CREATE INDEX IF NOT EXISTS idx_dependencies_parent ON dependencies(parent_id);

-- Audit log table
CREATE TABLE IF NOT EXISTS audit_log (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    task_id    TEXT NOT NULL,
    action     TEXT NOT NULL,
    field      TEXT,
    old_value  TEXT,
    new_value  TEXT,
    changed_at TEXT NOT NULL,
    changed_by TEXT NOT NULL
);

-- Index for querying audit log by task
CREATE INDEX IF NOT EXISTS idx_audit_log_task_id ON audit_log(task_id);

-- Index for querying audit log by action
CREATE INDEX IF NOT EXISTS idx_audit_log_action ON audit_log(action);

-- Index for querying audit log by agent
CREATE INDEX IF NOT EXISTS idx_audit_log_changed_by ON audit_log(changed_by);

-- Index for querying audit log by time
CREATE INDEX IF NOT EXISTS idx_audit_log_changed_at ON audit_log(changed_at);
`

// Manager handles multiple SQLite database connections, one per project.
type Manager struct {
	basePath string
	dbs      map[string]*sql.DB
	mu       sync.RWMutex
}

// NewManager creates a new database manager.
// basePath is the directory where project databases are stored (e.g., ~/.airyra/projects/).
func NewManager(basePath string) (*Manager, error) {
	if err := os.MkdirAll(basePath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create database directory: %w", err)
	}
	return &Manager{
		basePath: basePath,
		dbs:      make(map[string]*sql.DB),
	}, nil
}

// GetDB returns the database connection for a project, creating it if necessary.
func (m *Manager) GetDB(project string) (*sql.DB, error) {
	m.mu.RLock()
	if db, ok := m.dbs[project]; ok {
		m.mu.RUnlock()
		return db, nil
	}
	m.mu.RUnlock()

	m.mu.Lock()
	defer m.mu.Unlock()

	// Double-check after acquiring write lock
	if db, ok := m.dbs[project]; ok {
		return db, nil
	}

	dbPath := filepath.Join(m.basePath, project+".db")
	db, err := sql.Open("sqlite3", dbPath+"?_journal_mode=WAL&_busy_timeout=5000&_foreign_keys=on")
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Initialize schema
	if _, err := db.Exec(initialSchema); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to initialize schema: %w", err)
	}

	m.dbs[project] = db
	return db, nil
}

// ListProjects returns a list of all known projects (based on existing database files).
func (m *Manager) ListProjects() ([]string, error) {
	entries, err := os.ReadDir(m.basePath)
	if err != nil {
		if os.IsNotExist(err) {
			return []string{}, nil
		}
		return nil, fmt.Errorf("failed to list projects: %w", err)
	}

	var projects []string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if filepath.Ext(name) == ".db" {
			projects = append(projects, name[:len(name)-3])
		}
	}
	return projects, nil
}

// Close closes all database connections.
func (m *Manager) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	var errs []error
	for project, db := range m.dbs {
		if err := db.Close(); err != nil {
			errs = append(errs, fmt.Errorf("failed to close %s: %w", project, err))
		}
	}
	m.dbs = make(map[string]*sql.DB)

	if len(errs) > 0 {
		return fmt.Errorf("errors closing databases: %v", errs)
	}
	return nil
}
