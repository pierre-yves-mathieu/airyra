package sqlite

import (
	"database/sql"

	"github.com/airyra/airyra/internal/domain"
)

// SpecDependencyRepository handles spec dependency persistence operations.
type SpecDependencyRepository struct {
	db *sql.DB
}

// NewSpecDependencyRepository creates a new SpecDependencyRepository.
func NewSpecDependencyRepository(db *sql.DB) *SpecDependencyRepository {
	return &SpecDependencyRepository{db: db}
}

// Add adds a spec dependency (child depends on parent).
func (r *SpecDependencyRepository) Add(childID, parentID string) error {
	_, err := r.db.Exec(
		"INSERT INTO spec_dependencies (child_id, parent_id) VALUES (?, ?)",
		childID, parentID,
	)
	return err
}

// Remove removes a spec dependency.
func (r *SpecDependencyRepository) Remove(childID, parentID string) error {
	result, err := r.db.Exec(
		"DELETE FROM spec_dependencies WHERE child_id = ? AND parent_id = ?",
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

// ListByChild returns all dependencies for a given child spec.
func (r *SpecDependencyRepository) ListByChild(childID string) ([]*domain.SpecDependency, error) {
	rows, err := r.db.Query(
		"SELECT child_id, parent_id FROM spec_dependencies WHERE child_id = ?",
		childID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var deps []*domain.SpecDependency
	for rows.Next() {
		var dep domain.SpecDependency
		if err := rows.Scan(&dep.ChildID, &dep.ParentID); err != nil {
			return nil, err
		}
		deps = append(deps, &dep)
	}

	return deps, rows.Err()
}

// ListByParent returns all specs that depend on a given parent spec.
func (r *SpecDependencyRepository) ListByParent(parentID string) ([]*domain.SpecDependency, error) {
	rows, err := r.db.Query(
		"SELECT child_id, parent_id FROM spec_dependencies WHERE parent_id = ?",
		parentID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var deps []*domain.SpecDependency
	for rows.Next() {
		var dep domain.SpecDependency
		if err := rows.Scan(&dep.ChildID, &dep.ParentID); err != nil {
			return nil, err
		}
		deps = append(deps, &dep)
	}

	return deps, rows.Err()
}

// Exists checks if a spec dependency exists.
func (r *SpecDependencyRepository) Exists(childID, parentID string) (bool, error) {
	var count int
	err := r.db.QueryRow(
		"SELECT COUNT(*) FROM spec_dependencies WHERE child_id = ? AND parent_id = ?",
		childID, parentID,
	).Scan(&count)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// WouldCreateCycle checks if adding a dependency would create a cycle.
// Uses BFS to traverse from parentID looking for childID.
// Returns the cycle path if a cycle would be created, nil otherwise.
func (r *SpecDependencyRepository) WouldCreateCycle(childID, parentID string) ([]string, error) {
	// Use BFS to check if childID is reachable from parentID via existing dependencies.
	// If so, adding childID -> parentID would create a cycle.

	visited := make(map[string]bool)
	cameFrom := make(map[string]string)
	queue := []string{parentID}
	visited[parentID] = true

	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]

		if current == childID {
			// Found a path from parentID to childID
			path := []string{current}
			node := current
			for node != parentID {
				prev := cameFrom[node]
				path = append(path, prev)
				node = prev
			}
			path = append(path, childID)
			return path, nil
		}

		// Get all specs that current depends on
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
