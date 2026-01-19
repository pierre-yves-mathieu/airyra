package sqlite

import (
	"database/sql"

	"github.com/airyra/airyra/internal/domain"
)

// DependencyRepository handles dependency persistence operations.
type DependencyRepository struct {
	db *sql.DB
}

// NewDependencyRepository creates a new DependencyRepository.
func NewDependencyRepository(db *sql.DB) *DependencyRepository {
	return &DependencyRepository{db: db}
}

// Add adds a dependency (child depends on parent).
func (r *DependencyRepository) Add(childID, parentID string) error {
	_, err := r.db.Exec(
		"INSERT INTO dependencies (child_id, parent_id) VALUES (?, ?)",
		childID, parentID,
	)
	return err
}

// Remove removes a dependency.
func (r *DependencyRepository) Remove(childID, parentID string) error {
	result, err := r.db.Exec(
		"DELETE FROM dependencies WHERE child_id = ? AND parent_id = ?",
		childID, parentID,
	)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return sql.ErrNoRows
	}
	return nil
}

// ListByChild returns all dependencies for a given child task.
func (r *DependencyRepository) ListByChild(childID string) ([]*domain.Dependency, error) {
	rows, err := r.db.Query(
		"SELECT child_id, parent_id FROM dependencies WHERE child_id = ?",
		childID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var deps []*domain.Dependency
	for rows.Next() {
		var dep domain.Dependency
		if err := rows.Scan(&dep.ChildID, &dep.ParentID); err != nil {
			return nil, err
		}
		deps = append(deps, &dep)
	}

	return deps, rows.Err()
}

// ListByParent returns all tasks that depend on a given parent task.
func (r *DependencyRepository) ListByParent(parentID string) ([]*domain.Dependency, error) {
	rows, err := r.db.Query(
		"SELECT child_id, parent_id FROM dependencies WHERE parent_id = ?",
		parentID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var deps []*domain.Dependency
	for rows.Next() {
		var dep domain.Dependency
		if err := rows.Scan(&dep.ChildID, &dep.ParentID); err != nil {
			return nil, err
		}
		deps = append(deps, &dep)
	}

	return deps, rows.Err()
}

// Exists checks if a dependency exists.
func (r *DependencyRepository) Exists(childID, parentID string) (bool, error) {
	var count int
	err := r.db.QueryRow(
		"SELECT COUNT(*) FROM dependencies WHERE child_id = ? AND parent_id = ?",
		childID, parentID,
	).Scan(&count)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// WouldCreateCycle checks if adding a dependency would create a cycle.
// Returns the cycle path if a cycle would be created, nil otherwise.
func (r *DependencyRepository) WouldCreateCycle(childID, parentID string) ([]string, error) {
	// Use BFS to check if childID is reachable from parentID via existing dependencies.
	// If so, adding childID -> parentID would create a cycle.
	// Example: If A->B exists and we try to add B->A, we check if A is reachable from B.
	// Starting from B (parentID), we traverse dependencies. If we reach A (childID), there's a cycle.

	visited := make(map[string]bool)
	cameFrom := make(map[string]string) // tracks BFS traversal path
	queue := []string{parentID}
	visited[parentID] = true

	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]

		if current == childID {
			// Found a path from parentID to childID, so adding childID -> parentID creates a cycle
			// Reconstruct: childID -> ... -> parentID -> childID
			path := []string{current}
			node := current
			for node != parentID {
				prev := cameFrom[node]
				path = append(path, prev)
				node = prev
			}
			// Add childID at the end to show the complete cycle
			path = append(path, childID)
			return path, nil
		}

		// Get all tasks that current depends on
		deps, err := r.ListByChild(current)
		if err != nil {
			return nil, err
		}

		for _, dep := range deps {
			if !visited[dep.ParentID] {
				visited[dep.ParentID] = true
				cameFrom[dep.ParentID] = current
				queue = append(queue, dep.ParentID)
			}
		}
	}

	return nil, nil
}
