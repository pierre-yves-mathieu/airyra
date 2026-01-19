package storage

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

// ProjectDBManager manages SQLite database connections for multiple projects.
// It provides lazy database creation and connection caching.
type ProjectDBManager struct {
	basePath string
	stores   map[string]*SQLiteStore
	mu       sync.RWMutex
}

// NewProjectDBManager creates a new manager with the given base path.
// The base path is where project databases will be stored (e.g., ~/.airyra/projects/).
func NewProjectDBManager(basePath string) (*ProjectDBManager, error) {
	if err := os.MkdirAll(basePath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create database directory: %w", err)
	}

	return &ProjectDBManager{
		basePath: basePath,
		stores:   make(map[string]*SQLiteStore),
	}, nil
}

// GetStore returns the Store for a project, creating it if necessary.
// The database file is created lazily on first access.
func (m *ProjectDBManager) GetStore(project string) (Store, error) {
	// Validate project name
	if project == "" {
		return nil, fmt.Errorf("project name cannot be empty")
	}
	if strings.ContainsAny(project, "/\\:") {
		return nil, fmt.Errorf("project name contains invalid characters")
	}

	// Fast path: check if store already exists
	m.mu.RLock()
	if store, ok := m.stores[project]; ok {
		m.mu.RUnlock()
		return store, nil
	}
	m.mu.RUnlock()

	// Slow path: create new store
	m.mu.Lock()
	defer m.mu.Unlock()

	// Double-check after acquiring write lock
	if store, ok := m.stores[project]; ok {
		return store, nil
	}

	// Create the store
	dbPath := filepath.Join(m.basePath, project+".db")
	store, err := NewSQLiteStore(dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create store for project %s: %w", project, err)
	}

	m.stores[project] = store
	return store, nil
}

// ListProjects returns a list of all known projects.
// Projects are identified by .db files in the base path.
func (m *ProjectDBManager) ListProjects() ([]string, error) {
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
		if strings.HasSuffix(name, ".db") {
			// Extract project name by removing .db extension
			projectName := strings.TrimSuffix(name, ".db")
			// Skip WAL and SHM files
			if !strings.HasSuffix(projectName, "-wal") && !strings.HasSuffix(projectName, "-shm") {
				projects = append(projects, projectName)
			}
		}
	}

	return projects, nil
}

// CloseAll closes all database connections.
func (m *ProjectDBManager) CloseAll() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	var errs []error
	for project, store := range m.stores {
		if err := store.Close(); err != nil {
			errs = append(errs, fmt.Errorf("failed to close %s: %w", project, err))
		}
	}

	// Clear the stores map
	m.stores = make(map[string]*SQLiteStore)

	if len(errs) > 0 {
		return fmt.Errorf("errors closing databases: %v", errs)
	}
	return nil
}

// Close closes a specific project's database connection.
func (m *ProjectDBManager) Close(project string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	store, ok := m.stores[project]
	if !ok {
		return nil // Already closed or never opened
	}

	err := store.Close()
	delete(m.stores, project)
	return err
}

// BasePath returns the base path where databases are stored.
func (m *ProjectDBManager) BasePath() string {
	return m.basePath
}
