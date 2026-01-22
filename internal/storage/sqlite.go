package storage

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// SQLiteStore implements the Store interface using SQLite.
type SQLiteStore struct {
	db       *sql.DB
	closed   bool
	tasks    *sqliteTaskRepository
	deps     *sqliteDependencyRepository
	specs    *sqliteSpecRepository
	specDeps *sqliteSpecDependencyRepository
	audit    *sqliteAuditRepository
}

// NewSQLiteStore creates a new SQLite-backed store.
// The dsn can be a file path or ":memory:" for in-memory database.
func NewSQLiteStore(dsn string) (*SQLiteStore, error) {
	// Configure connection string with pragmas
	connStr := dsn
	if !strings.Contains(dsn, "?") {
		connStr += "?"
	} else {
		connStr += "&"
	}
	connStr += "_journal_mode=WAL&_busy_timeout=5000&_foreign_keys=on&_synchronous=NORMAL"

	db, err := sql.Open("sqlite3", connStr)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Run migrations to initialize/update schema
	if err := RunMigrations(db); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to run migrations: %w", err)
	}

	store := &SQLiteStore{db: db, closed: false}
	store.tasks = &sqliteTaskRepository{db: db}
	store.deps = &sqliteDependencyRepository{db: db}
	store.specs = &sqliteSpecRepository{db: db}
	store.specDeps = &sqliteSpecDependencyRepository{db: db}
	store.audit = &sqliteAuditRepository{db: db}

	return store, nil
}

// Tasks returns the task repository.
func (s *SQLiteStore) Tasks() TaskRepository {
	return s.tasks
}

// Dependencies returns the dependency repository.
func (s *SQLiteStore) Dependencies() DependencyRepository {
	return s.deps
}

// AuditLogs returns the audit log repository.
func (s *SQLiteStore) AuditLogs() AuditRepository {
	return s.audit
}

// Specs returns the spec repository.
func (s *SQLiteStore) Specs() SpecRepository {
	return s.specs
}

// SpecDependencies returns the spec dependency repository.
func (s *SQLiteStore) SpecDependencies() SpecDependencyRepository {
	return s.specDeps
}

// WithTx executes a function within a transaction.
func (s *SQLiteStore) WithTx(ctx context.Context, fn func(TxStore) error) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	txStore := &sqliteTxStore{
		tx:       tx,
		tasks:    &sqliteTaskRepository{tx: tx},
		deps:     &sqliteDependencyRepository{tx: tx},
		specs:    &sqliteSpecRepository{tx: tx},
		specDeps: &sqliteSpecDependencyRepository{tx: tx},
		audit:    &sqliteAuditRepository{tx: tx},
	}

	if err := fn(txStore); err != nil {
		if rbErr := tx.Rollback(); rbErr != nil {
			return fmt.Errorf("rollback failed: %v (original error: %w)", rbErr, err)
		}
		return err
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// Close closes the database connection.
func (s *SQLiteStore) Close() error {
	if s.closed {
		return nil
	}
	s.closed = true
	return s.db.Close()
}

// sqliteTxStore implements TxStore for transaction operations.
type sqliteTxStore struct {
	tx       *sql.Tx
	tasks    *sqliteTaskRepository
	deps     *sqliteDependencyRepository
	specs    *sqliteSpecRepository
	specDeps *sqliteSpecDependencyRepository
	audit    *sqliteAuditRepository
}

func (s *sqliteTxStore) Tasks() TaskRepository {
	return s.tasks
}

func (s *sqliteTxStore) Dependencies() DependencyRepository {
	return s.deps
}

func (s *sqliteTxStore) Specs() SpecRepository {
	return s.specs
}

func (s *sqliteTxStore) SpecDependencies() SpecDependencyRepository {
	return s.specDeps
}

func (s *sqliteTxStore) AuditLogs() AuditRepository {
	return s.audit
}

// dbExecutor is an interface for database operations that works with both *sql.DB and *sql.Tx
type dbExecutor interface {
	ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row
}

// ============================================================================
// Task Repository Implementation
// ============================================================================

type sqliteTaskRepository struct {
	db *sql.DB
	tx *sql.Tx
}

func (r *sqliteTaskRepository) executor() dbExecutor {
	if r.tx != nil {
		return r.tx
	}
	return r.db
}

func (r *sqliteTaskRepository) Create(ctx context.Context, task *Task) error {
	query := `
		INSERT INTO tasks (id, parent_id, spec_id, title, description, status, priority, claimed_by, claimed_at, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	var description *string
	if task.Description != nil {
		description = task.Description
	}

	var claimedAt *string
	if task.ClaimedAt != nil {
		ts := task.ClaimedAt.UTC().Format(time.RFC3339)
		claimedAt = &ts
	}

	_, err := r.executor().ExecContext(ctx, query,
		task.ID,
		task.ParentID,
		task.SpecID,
		task.Title,
		description,
		task.Status,
		task.Priority,
		task.ClaimedBy,
		claimedAt,
		task.CreatedAt.UTC().Format(time.RFC3339),
		task.UpdatedAt.UTC().Format(time.RFC3339),
	)
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE constraint failed") ||
			strings.Contains(err.Error(), "PRIMARY KEY constraint failed") {
			return fmt.Errorf("task with ID %s already exists", task.ID)
		}
		return fmt.Errorf("failed to create task: %w", err)
	}

	return nil
}

func (r *sqliteTaskRepository) Get(ctx context.Context, id string) (*Task, error) {
	query := `
		SELECT id, parent_id, spec_id, title, description, status, priority, claimed_by, claimed_at, created_at, updated_at
		FROM tasks WHERE id = ?
	`

	var task Task
	var description, claimedBy, claimedAt, createdAt, updatedAt sql.NullString
	var parentID, specID sql.NullString

	err := r.executor().QueryRowContext(ctx, query, id).Scan(
		&task.ID,
		&parentID,
		&specID,
		&task.Title,
		&description,
		&task.Status,
		&task.Priority,
		&claimedBy,
		&claimedAt,
		&createdAt,
		&updatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get task: %w", err)
	}

	if parentID.Valid {
		task.ParentID = &parentID.String
	}
	if specID.Valid {
		task.SpecID = &specID.String
	}
	if description.Valid {
		task.Description = &description.String
	}
	if claimedBy.Valid {
		task.ClaimedBy = &claimedBy.String
	}
	if claimedAt.Valid {
		if t, err := time.Parse(time.RFC3339, claimedAt.String); err == nil {
			task.ClaimedAt = &t
		}
	}
	if createdAt.Valid {
		if t, err := time.Parse(time.RFC3339, createdAt.String); err == nil {
			task.CreatedAt = t
		}
	}
	if updatedAt.Valid {
		if t, err := time.Parse(time.RFC3339, updatedAt.String); err == nil {
			task.UpdatedAt = t
		}
	}

	return &task, nil
}

func (r *sqliteTaskRepository) List(ctx context.Context, opts ListOptions) ([]*Task, int, error) {
	opts.Normalize()

	// Build WHERE clause
	var conditions []string
	var args []interface{}

	if opts.Status != nil {
		conditions = append(conditions, "status = ?")
		args = append(args, *opts.Status)
	}
	if opts.ParentID != nil {
		conditions = append(conditions, "parent_id = ?")
		args = append(args, *opts.ParentID)
	}
	if opts.Priority != nil {
		conditions = append(conditions, "priority = ?")
		args = append(args, *opts.Priority)
	}

	whereClause := ""
	if len(conditions) > 0 {
		whereClause = "WHERE " + strings.Join(conditions, " AND ")
	}

	// Count total
	countQuery := "SELECT COUNT(*) FROM tasks " + whereClause
	var total int
	if err := r.executor().QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("failed to count tasks: %w", err)
	}

	// Get tasks with pagination
	offset := (opts.Page - 1) * opts.PerPage
	query := fmt.Sprintf(`
		SELECT id, parent_id, spec_id, title, description, status, priority, claimed_by, claimed_at, created_at, updated_at
		FROM tasks %s
		ORDER BY priority DESC, created_at ASC
		LIMIT ? OFFSET ?
	`, whereClause)

	args = append(args, opts.PerPage, offset)
	rows, err := r.executor().QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list tasks: %w", err)
	}
	defer rows.Close()

	return scanTasks(rows, total)
}

func (r *sqliteTaskRepository) ListReady(ctx context.Context, opts ListOptions) ([]*Task, int, error) {
	opts.Normalize()

	// A task is ready if:
	// 1. It has status = 'open'
	// 2. All its dependencies (parent tasks) are in 'done' status
	// This includes tasks with no dependencies

	// Build additional WHERE conditions
	var conditions []string
	var args []interface{}

	conditions = append(conditions, "t.status = 'open'")

	if opts.ParentID != nil {
		conditions = append(conditions, "t.parent_id = ?")
		args = append(args, *opts.ParentID)
	}
	if opts.Priority != nil {
		conditions = append(conditions, "t.priority = ?")
		args = append(args, *opts.Priority)
	}

	whereClause := "WHERE " + strings.Join(conditions, " AND ")

	// Query for tasks that are open and have no incomplete dependencies
	countQuery := fmt.Sprintf(`
		SELECT COUNT(*)
		FROM tasks t
		%s
		AND NOT EXISTS (
			SELECT 1 FROM dependencies d
			INNER JOIN tasks p ON d.parent_id = p.id
			WHERE d.child_id = t.id AND p.status != 'done'
		)
	`, whereClause)

	var total int
	if err := r.executor().QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("failed to count ready tasks: %w", err)
	}

	offset := (opts.Page - 1) * opts.PerPage
	query := fmt.Sprintf(`
		SELECT t.id, t.parent_id, t.spec_id, t.title, t.description, t.status, t.priority, t.claimed_by, t.claimed_at, t.created_at, t.updated_at
		FROM tasks t
		%s
		AND NOT EXISTS (
			SELECT 1 FROM dependencies d
			INNER JOIN tasks p ON d.parent_id = p.id
			WHERE d.child_id = t.id AND p.status != 'done'
		)
		ORDER BY t.priority DESC, t.created_at ASC
		LIMIT ? OFFSET ?
	`, whereClause)

	args = append(args, opts.PerPage, offset)
	rows, err := r.executor().QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list ready tasks: %w", err)
	}
	defer rows.Close()

	return scanTasks(rows, total)
}

func scanTasks(rows *sql.Rows, total int) ([]*Task, int, error) {
	var tasks []*Task
	for rows.Next() {
		var task Task
		var description, claimedBy, claimedAt, createdAt, updatedAt sql.NullString
		var parentID, specID sql.NullString

		if err := rows.Scan(
			&task.ID,
			&parentID,
			&specID,
			&task.Title,
			&description,
			&task.Status,
			&task.Priority,
			&claimedBy,
			&claimedAt,
			&createdAt,
			&updatedAt,
		); err != nil {
			return nil, 0, fmt.Errorf("failed to scan task: %w", err)
		}

		if parentID.Valid {
			task.ParentID = &parentID.String
		}
		if specID.Valid {
			task.SpecID = &specID.String
		}
		if description.Valid {
			task.Description = &description.String
		}
		if claimedBy.Valid {
			task.ClaimedBy = &claimedBy.String
		}
		if claimedAt.Valid {
			if t, err := time.Parse(time.RFC3339, claimedAt.String); err == nil {
				task.ClaimedAt = &t
			}
		}
		if createdAt.Valid {
			if t, err := time.Parse(time.RFC3339, createdAt.String); err == nil {
				task.CreatedAt = t
			}
		}
		if updatedAt.Valid {
			if t, err := time.Parse(time.RFC3339, updatedAt.String); err == nil {
				task.UpdatedAt = t
			}
		}

		tasks = append(tasks, &task)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("error iterating tasks: %w", err)
	}

	return tasks, total, nil
}

func (r *sqliteTaskRepository) Update(ctx context.Context, task *Task) error {
	query := `
		UPDATE tasks SET
			parent_id = ?,
			spec_id = ?,
			title = ?,
			description = ?,
			status = ?,
			priority = ?,
			claimed_by = ?,
			claimed_at = ?,
			updated_at = ?
		WHERE id = ?
	`

	var claimedAt *string
	if task.ClaimedAt != nil {
		ts := task.ClaimedAt.UTC().Format(time.RFC3339)
		claimedAt = &ts
	}

	result, err := r.executor().ExecContext(ctx, query,
		task.ParentID,
		task.SpecID,
		task.Title,
		task.Description,
		task.Status,
		task.Priority,
		task.ClaimedBy,
		claimedAt,
		task.UpdatedAt.UTC().Format(time.RFC3339),
		task.ID,
	)
	if err != nil {
		return fmt.Errorf("failed to update task: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return ErrNotFound
	}

	return nil
}

func (r *sqliteTaskRepository) Delete(ctx context.Context, id string) error {
	result, err := r.executor().ExecContext(ctx, "DELETE FROM tasks WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("failed to delete task: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return ErrNotFound
	}

	return nil
}

// Claim atomically claims a task for an agent.
// Uses UPDATE ... WHERE status='open' pattern for atomicity.
func (r *sqliteTaskRepository) Claim(ctx context.Context, id, agentID string) error {
	now := time.Now().UTC()

	// Atomic claim: only succeeds if status is 'open'
	query := `
		UPDATE tasks SET
			status = ?,
			claimed_by = ?,
			claimed_at = ?,
			updated_at = ?
		WHERE id = ? AND status = ?
	`

	result, err := r.executor().ExecContext(ctx, query,
		StatusInProgress,
		agentID,
		now.Format(time.RFC3339),
		now.Format(time.RFC3339),
		id,
		StatusOpen,
	)
	if err != nil {
		return fmt.Errorf("failed to claim task: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		// Task either doesn't exist or is not in 'open' status
		task, err := r.Get(ctx, id)
		if err != nil {
			return ErrNotFound
		}

		// Task exists but is not open
		if task.Status == StatusInProgress {
			return ErrAlreadyClaimed
		}
		return &TransitionError{
			TaskID:     id,
			FromStatus: task.Status,
			ToStatus:   StatusInProgress,
		}
	}

	return nil
}

// Release releases a claimed task back to 'open' status.
func (r *sqliteTaskRepository) Release(ctx context.Context, id, agentID string) error {
	now := time.Now().UTC()

	// Only release if claimed by the same agent
	query := `
		UPDATE tasks SET
			status = ?,
			claimed_by = NULL,
			claimed_at = NULL,
			updated_at = ?
		WHERE id = ? AND status = ? AND claimed_by = ?
	`

	result, err := r.executor().ExecContext(ctx, query,
		StatusOpen,
		now.Format(time.RFC3339),
		id,
		StatusInProgress,
		agentID,
	)
	if err != nil {
		return fmt.Errorf("failed to release task: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		task, err := r.Get(ctx, id)
		if err != nil {
			return ErrNotFound
		}

		if task.Status != StatusInProgress {
			return &TransitionError{
				TaskID:     id,
				FromStatus: task.Status,
				ToStatus:   StatusOpen,
			}
		}

		if task.ClaimedBy != nil && *task.ClaimedBy != agentID {
			return ErrNotOwner
		}

		return ErrNotFound
	}

	return nil
}

// MarkDone marks a task as done.
func (r *sqliteTaskRepository) MarkDone(ctx context.Context, id, agentID string) error {
	now := time.Now().UTC()

	// Only mark done if claimed by the same agent
	query := `
		UPDATE tasks SET
			status = ?,
			updated_at = ?
		WHERE id = ? AND status = ? AND claimed_by = ?
	`

	result, err := r.executor().ExecContext(ctx, query,
		StatusDone,
		now.Format(time.RFC3339),
		id,
		StatusInProgress,
		agentID,
	)
	if err != nil {
		return fmt.Errorf("failed to mark task done: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		task, err := r.Get(ctx, id)
		if err != nil {
			return ErrNotFound
		}

		if task.Status != StatusInProgress {
			return &TransitionError{
				TaskID:     id,
				FromStatus: task.Status,
				ToStatus:   StatusDone,
			}
		}

		if task.ClaimedBy != nil && *task.ClaimedBy != agentID {
			return ErrNotOwner
		}

		return ErrNotFound
	}

	return nil
}

// Block transitions a task to 'blocked' status.
func (r *sqliteTaskRepository) Block(ctx context.Context, id string) error {
	now := time.Now().UTC()

	query := `
		UPDATE tasks SET
			status = ?,
			updated_at = ?
		WHERE id = ?
	`

	result, err := r.executor().ExecContext(ctx, query,
		StatusBlocked,
		now.Format(time.RFC3339),
		id,
	)
	if err != nil {
		return fmt.Errorf("failed to block task: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return ErrNotFound
	}

	return nil
}

// Unblock transitions a task from 'blocked' back to 'open' status.
func (r *sqliteTaskRepository) Unblock(ctx context.Context, id string) error {
	now := time.Now().UTC()

	query := `
		UPDATE tasks SET
			status = ?,
			updated_at = ?
		WHERE id = ? AND status = ?
	`

	result, err := r.executor().ExecContext(ctx, query,
		StatusOpen,
		now.Format(time.RFC3339),
		id,
		StatusBlocked,
	)
	if err != nil {
		return fmt.Errorf("failed to unblock task: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		task, err := r.Get(ctx, id)
		if err != nil {
			return ErrNotFound
		}

		return &TransitionError{
			TaskID:     id,
			FromStatus: task.Status,
			ToStatus:   StatusOpen,
		}
	}

	return nil
}

// ============================================================================
// Dependency Repository Implementation
// ============================================================================

type sqliteDependencyRepository struct {
	db *sql.DB
	tx *sql.Tx
}

func (r *sqliteDependencyRepository) executor() dbExecutor {
	if r.tx != nil {
		return r.tx
	}
	return r.db
}

// Add creates a dependency between two tasks.
func (r *sqliteDependencyRepository) Add(ctx context.Context, childID, parentID string) error {
	// First check for cycles
	hasCycle, path, err := r.CheckCycle(ctx, childID, parentID)
	if err != nil {
		return fmt.Errorf("failed to check for cycles: %w", err)
	}
	if hasCycle {
		return &CycleError{
			ChildID:  childID,
			ParentID: parentID,
			Path:     path,
		}
	}

	query := "INSERT INTO dependencies (child_id, parent_id) VALUES (?, ?)"
	_, err = r.executor().ExecContext(ctx, query, childID, parentID)
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE constraint failed") ||
			strings.Contains(err.Error(), "PRIMARY KEY constraint failed") {
			return fmt.Errorf("dependency already exists")
		}
		return fmt.Errorf("failed to add dependency: %w", err)
	}

	return nil
}

// Remove deletes a dependency between two tasks.
func (r *sqliteDependencyRepository) Remove(ctx context.Context, childID, parentID string) error {
	query := "DELETE FROM dependencies WHERE child_id = ? AND parent_id = ?"
	_, err := r.executor().ExecContext(ctx, query, childID, parentID)
	if err != nil {
		return fmt.Errorf("failed to remove dependency: %w", err)
	}
	return nil
}

// ListForTask returns all dependencies for a task (both as child and parent).
func (r *sqliteDependencyRepository) ListForTask(ctx context.Context, taskID string) ([]Dependency, error) {
	query := `
		SELECT child_id, parent_id FROM dependencies
		WHERE child_id = ? OR parent_id = ?
	`

	rows, err := r.executor().QueryContext(ctx, query, taskID, taskID)
	if err != nil {
		return nil, fmt.Errorf("failed to list dependencies: %w", err)
	}
	defer rows.Close()

	var deps []Dependency
	for rows.Next() {
		var dep Dependency
		if err := rows.Scan(&dep.ChildID, &dep.ParentID); err != nil {
			return nil, fmt.Errorf("failed to scan dependency: %w", err)
		}
		deps = append(deps, dep)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating dependencies: %w", err)
	}

	return deps, nil
}

// CheckCycle checks if adding a dependency from childID to parentID would create a cycle.
// It uses BFS to traverse from parentID to see if we can reach childID.
func (r *sqliteDependencyRepository) CheckCycle(ctx context.Context, childID, parentID string) (bool, []string, error) {
	// If childID depends on parentID, we need to check if parentID eventually depends on childID
	// This would create a cycle: childID -> parentID -> ... -> childID

	// Use BFS to find a path from parentID to childID through the dependency graph
	visited := make(map[string]bool)
	parent := make(map[string]string) // To reconstruct the path

	queue := []string{parentID}
	visited[parentID] = true

	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]

		// Check if we reached the target
		if current == childID {
			// Reconstruct the path
			path := []string{childID}
			for node := childID; node != parentID; {
				prev, ok := parent[node]
				if !ok {
					break
				}
				path = append([]string{prev}, path...)
				node = prev
			}
			// Add the original childID at the start to show the full cycle
			path = append([]string{childID}, path...)
			return true, path, nil
		}

		// Get all tasks that current depends on (where current is the child)
		parents, err := r.getParents(ctx, current)
		if err != nil {
			return false, nil, err
		}

		for _, p := range parents {
			if !visited[p] {
				visited[p] = true
				parent[p] = current
				queue = append(queue, p)
			}
		}
	}

	return false, nil, nil
}

// getParents returns all parent IDs for a given child (tasks it depends on)
func (r *sqliteDependencyRepository) getParents(ctx context.Context, childID string) ([]string, error) {
	query := "SELECT parent_id FROM dependencies WHERE child_id = ?"
	rows, err := r.executor().QueryContext(ctx, query, childID)
	if err != nil {
		return nil, fmt.Errorf("failed to get dependencies: %w", err)
	}
	defer rows.Close()

	var parents []string
	for rows.Next() {
		var parentID string
		if err := rows.Scan(&parentID); err != nil {
			return nil, fmt.Errorf("failed to scan dependency: %w", err)
		}
		parents = append(parents, parentID)
	}

	return parents, rows.Err()
}

// ============================================================================
// Audit Repository Implementation
// ============================================================================

type sqliteAuditRepository struct {
	db *sql.DB
	tx *sql.Tx
}

func (r *sqliteAuditRepository) executor() dbExecutor {
	if r.tx != nil {
		return r.tx
	}
	return r.db
}

// Log records an audit entry.
func (r *sqliteAuditRepository) Log(ctx context.Context, entry *AuditEntry) error {
	query := `
		INSERT INTO audit_log (task_id, action, field, old_value, new_value, changed_at, changed_by)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`

	result, err := r.executor().ExecContext(ctx, query,
		entry.TaskID,
		entry.Action,
		entry.Field,
		entry.OldValue,
		entry.NewValue,
		entry.ChangedAt.UTC().Format(time.RFC3339),
		entry.ChangedBy,
	)
	if err != nil {
		return fmt.Errorf("failed to log audit entry: %w", err)
	}

	id, err := result.LastInsertId()
	if err == nil {
		entry.ID = id
	}

	return nil
}

// ListForTask returns all audit entries for a specific task.
func (r *sqliteAuditRepository) ListForTask(ctx context.Context, taskID string) ([]*AuditEntry, error) {
	query := `
		SELECT id, task_id, action, field, old_value, new_value, changed_at, changed_by
		FROM audit_log
		WHERE task_id = ?
		ORDER BY changed_at DESC
	`

	rows, err := r.executor().QueryContext(ctx, query, taskID)
	if err != nil {
		return nil, fmt.Errorf("failed to list audit entries: %w", err)
	}
	defer rows.Close()

	return scanAuditEntries(rows)
}

// Query returns audit entries matching the specified options.
func (r *sqliteAuditRepository) Query(ctx context.Context, opts AuditQueryOptions) ([]*AuditEntry, error) {
	var conditions []string
	var args []interface{}

	if opts.Action != nil {
		conditions = append(conditions, "action = ?")
		args = append(args, *opts.Action)
	}
	if opts.ChangedBy != nil {
		conditions = append(conditions, "changed_by = ?")
		args = append(args, *opts.ChangedBy)
	}
	if opts.Since != nil {
		conditions = append(conditions, "changed_at >= ?")
		args = append(args, opts.Since.UTC().Format(time.RFC3339))
	}
	if opts.Until != nil {
		conditions = append(conditions, "changed_at <= ?")
		args = append(args, opts.Until.UTC().Format(time.RFC3339))
	}

	whereClause := ""
	if len(conditions) > 0 {
		whereClause = "WHERE " + strings.Join(conditions, " AND ")
	}

	query := fmt.Sprintf(`
		SELECT id, task_id, action, field, old_value, new_value, changed_at, changed_by
		FROM audit_log
		%s
		ORDER BY changed_at DESC
	`, whereClause)

	rows, err := r.executor().QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query audit entries: %w", err)
	}
	defer rows.Close()

	return scanAuditEntries(rows)
}

func scanAuditEntries(rows *sql.Rows) ([]*AuditEntry, error) {
	var entries []*AuditEntry
	for rows.Next() {
		var entry AuditEntry
		var field, oldValue, newValue, changedAt sql.NullString

		if err := rows.Scan(
			&entry.ID,
			&entry.TaskID,
			&entry.Action,
			&field,
			&oldValue,
			&newValue,
			&changedAt,
			&entry.ChangedBy,
		); err != nil {
			return nil, fmt.Errorf("failed to scan audit entry: %w", err)
		}

		if field.Valid {
			entry.Field = &field.String
		}
		if oldValue.Valid {
			entry.OldValue = &oldValue.String
		}
		if newValue.Valid {
			entry.NewValue = &newValue.String
		}
		if changedAt.Valid {
			if t, err := time.Parse(time.RFC3339, changedAt.String); err == nil {
				entry.ChangedAt = t
			}
		}

		entries = append(entries, &entry)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating audit entries: %w", err)
	}

	return entries, nil
}

// ============================================================================
// Spec Repository Implementation
// ============================================================================

type sqliteSpecRepository struct {
	db *sql.DB
	tx *sql.Tx
}

func (r *sqliteSpecRepository) executor() dbExecutor {
	if r.tx != nil {
		return r.tx
	}
	return r.db
}

// Create inserts a new spec into the store.
func (r *sqliteSpecRepository) Create(ctx context.Context, spec *Spec) error {
	query := `
		INSERT INTO specs (id, title, description, manual_status, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?)
	`
	_, err := r.executor().ExecContext(ctx, query,
		spec.ID,
		spec.Title,
		spec.Description,
		spec.ManualStatus,
		spec.CreatedAt.UTC().Format(time.RFC3339),
		spec.UpdatedAt.UTC().Format(time.RFC3339),
	)
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE constraint failed") ||
			strings.Contains(err.Error(), "PRIMARY KEY constraint failed") {
			return fmt.Errorf("spec with ID %s already exists", spec.ID)
		}
		return fmt.Errorf("failed to create spec: %w", err)
	}
	return nil
}

// Get retrieves a spec by its ID with computed task counts.
func (r *sqliteSpecRepository) Get(ctx context.Context, id string) (*Spec, error) {
	query := `
		SELECT
			s.id,
			s.title,
			s.description,
			s.manual_status,
			s.created_at,
			s.updated_at,
			COALESCE((SELECT COUNT(*) FROM tasks WHERE spec_id = s.id), 0) as task_count,
			COALESCE((SELECT COUNT(*) FROM tasks WHERE spec_id = s.id AND status = 'done'), 0) as done_count
		FROM specs s
		WHERE s.id = ?
	`

	var spec Spec
	var description, manualStatus, createdAt, updatedAt sql.NullString

	err := r.executor().QueryRowContext(ctx, query, id).Scan(
		&spec.ID,
		&spec.Title,
		&description,
		&manualStatus,
		&createdAt,
		&updatedAt,
		&spec.TaskCount,
		&spec.DoneCount,
	)
	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get spec: %w", err)
	}

	if description.Valid {
		spec.Description = &description.String
	}
	if manualStatus.Valid {
		spec.ManualStatus = &manualStatus.String
	}
	if createdAt.Valid {
		if t, err := time.Parse(time.RFC3339, createdAt.String); err == nil {
			spec.CreatedAt = t
		}
	}
	if updatedAt.Valid {
		if t, err := time.Parse(time.RFC3339, updatedAt.String); err == nil {
			spec.UpdatedAt = t
		}
	}

	return &spec, nil
}

// List returns specs matching the specified options.
func (r *sqliteSpecRepository) List(ctx context.Context, opts SpecListOptions) ([]*Spec, int, error) {
	opts.Normalize()
	offset := (opts.Page - 1) * opts.PerPage

	baseQuery := `
		SELECT
			s.id,
			s.title,
			s.description,
			s.manual_status,
			s.created_at,
			s.updated_at,
			COALESCE((SELECT COUNT(*) FROM tasks WHERE spec_id = s.id), 0) as task_count,
			COALESCE((SELECT COUNT(*) FROM tasks WHERE spec_id = s.id AND status = 'done'), 0) as done_count
		FROM specs s
	`

	// For status filtering, we need to compute and filter
	if opts.Status != nil {
		var statusCondition string
		switch *opts.Status {
		case SpecStatusCancelled:
			statusCondition = "WHERE s.manual_status = 'cancelled'"
		case SpecStatusDraft:
			statusCondition = `WHERE s.manual_status IS NULL AND
				(SELECT COUNT(*) FROM tasks WHERE spec_id = s.id) = 0`
		case SpecStatusDone:
			statusCondition = `WHERE s.manual_status IS NULL AND
				(SELECT COUNT(*) FROM tasks WHERE spec_id = s.id) > 0 AND
				(SELECT COUNT(*) FROM tasks WHERE spec_id = s.id) =
				(SELECT COUNT(*) FROM tasks WHERE spec_id = s.id AND status = 'done')`
		case SpecStatusActive:
			statusCondition = `WHERE s.manual_status IS NULL AND
				(SELECT COUNT(*) FROM tasks WHERE spec_id = s.id) > 0 AND
				(SELECT COUNT(*) FROM tasks WHERE spec_id = s.id) !=
				(SELECT COUNT(*) FROM tasks WHERE spec_id = s.id AND status = 'done')`
		default:
			return nil, 0, fmt.Errorf("invalid status: %s", *opts.Status)
		}

		// Count with filter
		countQuery := "SELECT COUNT(*) FROM (" + baseQuery + statusCondition + ")"
		var total int
		if err := r.executor().QueryRowContext(ctx, countQuery).Scan(&total); err != nil {
			return nil, 0, fmt.Errorf("failed to count specs: %w", err)
		}

		// Fetch with filter
		query := baseQuery + statusCondition + " ORDER BY s.created_at DESC LIMIT ? OFFSET ?"
		rows, err := r.executor().QueryContext(ctx, query, opts.PerPage, offset)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to list specs: %w", err)
		}
		defer rows.Close()

		specs, err := scanSpecs(rows)
		if err != nil {
			return nil, 0, err
		}
		return specs, total, nil
	}

	// No filter - simple query
	countQuery := "SELECT COUNT(*) FROM specs"
	var total int
	if err := r.executor().QueryRowContext(ctx, countQuery).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("failed to count specs: %w", err)
	}

	query := baseQuery + " ORDER BY s.created_at DESC LIMIT ? OFFSET ?"
	rows, err := r.executor().QueryContext(ctx, query, opts.PerPage, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list specs: %w", err)
	}
	defer rows.Close()

	specs, err := scanSpecs(rows)
	if err != nil {
		return nil, 0, err
	}
	return specs, total, nil
}

// ListReady returns specs that have no unmet dependencies and are ready to be worked on.
func (r *sqliteSpecRepository) ListReady(ctx context.Context, opts SpecListOptions) ([]*Spec, int, error) {
	opts.Normalize()
	offset := (opts.Page - 1) * opts.PerPage

	// A spec is ready if:
	// 1. Not cancelled
	// 2. Not already done
	// 3. All parent specs (dependencies) are done
	query := `
		SELECT
			s.id,
			s.title,
			s.description,
			s.manual_status,
			s.created_at,
			s.updated_at,
			COALESCE((SELECT COUNT(*) FROM tasks WHERE spec_id = s.id), 0) as task_count,
			COALESCE((SELECT COUNT(*) FROM tasks WHERE spec_id = s.id AND status = 'done'), 0) as done_count
		FROM specs s
		WHERE s.manual_status IS NULL
		AND NOT (
			(SELECT COUNT(*) FROM tasks WHERE spec_id = s.id) > 0 AND
			(SELECT COUNT(*) FROM tasks WHERE spec_id = s.id) =
			(SELECT COUNT(*) FROM tasks WHERE spec_id = s.id AND status = 'done')
		)
		AND NOT EXISTS (
			SELECT 1 FROM spec_dependencies sd
			JOIN specs parent ON sd.parent_id = parent.id
			WHERE sd.child_id = s.id
			AND (
				parent.manual_status = 'cancelled'
				OR (SELECT COUNT(*) FROM tasks WHERE spec_id = parent.id) = 0
				OR (SELECT COUNT(*) FROM tasks WHERE spec_id = parent.id) !=
				   (SELECT COUNT(*) FROM tasks WHERE spec_id = parent.id AND status = 'done')
			)
		)
	`

	countQuery := "SELECT COUNT(*) FROM (" + query + ")"
	var total int
	if err := r.executor().QueryRowContext(ctx, countQuery).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("failed to count ready specs: %w", err)
	}

	query += " ORDER BY s.created_at ASC LIMIT ? OFFSET ?"
	rows, err := r.executor().QueryContext(ctx, query, opts.PerPage, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list ready specs: %w", err)
	}
	defer rows.Close()

	specs, err := scanSpecs(rows)
	if err != nil {
		return nil, 0, err
	}
	return specs, total, nil
}

// ListTasks returns tasks belonging to a spec.
func (r *sqliteSpecRepository) ListTasks(ctx context.Context, specID string, opts ListOptions) ([]*Task, int, error) {
	opts.Normalize()
	offset := (opts.Page - 1) * opts.PerPage

	// Build WHERE clause
	var conditions []string
	var args []interface{}

	conditions = append(conditions, "spec_id = ?")
	args = append(args, specID)

	if opts.Status != nil {
		conditions = append(conditions, "status = ?")
		args = append(args, *opts.Status)
	}
	if opts.Priority != nil {
		conditions = append(conditions, "priority = ?")
		args = append(args, *opts.Priority)
	}

	whereClause := "WHERE " + strings.Join(conditions, " AND ")

	countQuery := "SELECT COUNT(*) FROM tasks " + whereClause
	var total int
	if err := r.executor().QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("failed to count tasks: %w", err)
	}

	query := fmt.Sprintf(`
		SELECT id, parent_id, spec_id, title, description, status, priority, claimed_by, claimed_at, created_at, updated_at
		FROM tasks %s
		ORDER BY priority DESC, created_at ASC
		LIMIT ? OFFSET ?
	`, whereClause)

	args = append(args, opts.PerPage, offset)
	rows, err := r.executor().QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list tasks: %w", err)
	}
	defer rows.Close()

	return scanTasksWithSpecID(rows, total)
}

// Update modifies an existing spec.
func (r *sqliteSpecRepository) Update(ctx context.Context, spec *Spec) error {
	query := `
		UPDATE specs SET
			title = ?,
			description = ?,
			manual_status = ?,
			updated_at = ?
		WHERE id = ?
	`

	result, err := r.executor().ExecContext(ctx, query,
		spec.Title,
		spec.Description,
		spec.ManualStatus,
		spec.UpdatedAt.UTC().Format(time.RFC3339),
		spec.ID,
	)
	if err != nil {
		return fmt.Errorf("failed to update spec: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return ErrNotFound
	}

	return nil
}

// Delete removes a spec by its ID.
func (r *sqliteSpecRepository) Delete(ctx context.Context, id string) error {
	result, err := r.executor().ExecContext(ctx, "DELETE FROM specs WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("failed to delete spec: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return ErrNotFound
	}

	return nil
}

func scanSpecs(rows *sql.Rows) ([]*Spec, error) {
	var specs []*Spec
	for rows.Next() {
		var spec Spec
		var description, manualStatus, createdAt, updatedAt sql.NullString

		if err := rows.Scan(
			&spec.ID,
			&spec.Title,
			&description,
			&manualStatus,
			&createdAt,
			&updatedAt,
			&spec.TaskCount,
			&spec.DoneCount,
		); err != nil {
			return nil, fmt.Errorf("failed to scan spec: %w", err)
		}

		if description.Valid {
			spec.Description = &description.String
		}
		if manualStatus.Valid {
			spec.ManualStatus = &manualStatus.String
		}
		if createdAt.Valid {
			if t, err := time.Parse(time.RFC3339, createdAt.String); err == nil {
				spec.CreatedAt = t
			}
		}
		if updatedAt.Valid {
			if t, err := time.Parse(time.RFC3339, updatedAt.String); err == nil {
				spec.UpdatedAt = t
			}
		}

		specs = append(specs, &spec)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating specs: %w", err)
	}

	return specs, nil
}

func scanTasksWithSpecID(rows *sql.Rows, total int) ([]*Task, int, error) {
	var tasks []*Task
	for rows.Next() {
		var task Task
		var description, claimedBy, claimedAt, createdAt, updatedAt sql.NullString
		var parentID, specID sql.NullString

		if err := rows.Scan(
			&task.ID,
			&parentID,
			&specID,
			&task.Title,
			&description,
			&task.Status,
			&task.Priority,
			&claimedBy,
			&claimedAt,
			&createdAt,
			&updatedAt,
		); err != nil {
			return nil, 0, fmt.Errorf("failed to scan task: %w", err)
		}

		if parentID.Valid {
			task.ParentID = &parentID.String
		}
		if specID.Valid {
			task.SpecID = &specID.String
		}
		if description.Valid {
			task.Description = &description.String
		}
		if claimedBy.Valid {
			task.ClaimedBy = &claimedBy.String
		}
		if claimedAt.Valid {
			if t, err := time.Parse(time.RFC3339, claimedAt.String); err == nil {
				task.ClaimedAt = &t
			}
		}
		if createdAt.Valid {
			if t, err := time.Parse(time.RFC3339, createdAt.String); err == nil {
				task.CreatedAt = t
			}
		}
		if updatedAt.Valid {
			if t, err := time.Parse(time.RFC3339, updatedAt.String); err == nil {
				task.UpdatedAt = t
			}
		}

		tasks = append(tasks, &task)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("error iterating tasks: %w", err)
	}

	return tasks, total, nil
}

// ============================================================================
// Spec Dependency Repository Implementation
// ============================================================================

type sqliteSpecDependencyRepository struct {
	db *sql.DB
	tx *sql.Tx
}

func (r *sqliteSpecDependencyRepository) executor() dbExecutor {
	if r.tx != nil {
		return r.tx
	}
	return r.db
}

// Add creates a dependency between two specs.
func (r *sqliteSpecDependencyRepository) Add(ctx context.Context, childID, parentID string) error {
	// First check for cycles
	hasCycle, path, err := r.CheckCycle(ctx, childID, parentID)
	if err != nil {
		return fmt.Errorf("failed to check for cycles: %w", err)
	}
	if hasCycle {
		return &CycleError{
			ChildID:  childID,
			ParentID: parentID,
			Path:     path,
		}
	}

	query := "INSERT INTO spec_dependencies (child_id, parent_id) VALUES (?, ?)"
	_, err = r.executor().ExecContext(ctx, query, childID, parentID)
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE constraint failed") ||
			strings.Contains(err.Error(), "PRIMARY KEY constraint failed") {
			return fmt.Errorf("spec dependency already exists")
		}
		return fmt.Errorf("failed to add spec dependency: %w", err)
	}

	return nil
}

// Remove deletes a spec dependency.
func (r *sqliteSpecDependencyRepository) Remove(ctx context.Context, childID, parentID string) error {
	query := "DELETE FROM spec_dependencies WHERE child_id = ? AND parent_id = ?"
	result, err := r.executor().ExecContext(ctx, query, childID, parentID)
	if err != nil {
		return fmt.Errorf("failed to remove spec dependency: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return ErrNotFound
	}

	return nil
}

// ListForSpec returns all dependencies for a spec (both as child and parent).
func (r *sqliteSpecDependencyRepository) ListForSpec(ctx context.Context, specID string) ([]SpecDependency, error) {
	query := `
		SELECT child_id, parent_id FROM spec_dependencies
		WHERE child_id = ? OR parent_id = ?
	`

	rows, err := r.executor().QueryContext(ctx, query, specID, specID)
	if err != nil {
		return nil, fmt.Errorf("failed to list spec dependencies: %w", err)
	}
	defer rows.Close()

	var deps []SpecDependency
	for rows.Next() {
		var dep SpecDependency
		if err := rows.Scan(&dep.ChildID, &dep.ParentID); err != nil {
			return nil, fmt.Errorf("failed to scan spec dependency: %w", err)
		}
		deps = append(deps, dep)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating spec dependencies: %w", err)
	}

	return deps, nil
}

// CheckCycle checks if adding a dependency from childID to parentID would create a cycle.
func (r *sqliteSpecDependencyRepository) CheckCycle(ctx context.Context, childID, parentID string) (bool, []string, error) {
	// Use BFS to find a path from parentID to childID through the dependency graph
	visited := make(map[string]bool)
	parent := make(map[string]string)

	queue := []string{parentID}
	visited[parentID] = true

	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]

		if current == childID {
			// Reconstruct the path
			path := []string{childID}
			for node := childID; node != parentID; {
				prev, ok := parent[node]
				if !ok {
					break
				}
				path = append([]string{prev}, path...)
				node = prev
			}
			path = append([]string{childID}, path...)
			return true, path, nil
		}

		// Get all specs that current depends on
		parents, err := r.getParents(ctx, current)
		if err != nil {
			return false, nil, err
		}

		for _, p := range parents {
			if !visited[p] {
				visited[p] = true
				parent[p] = current
				queue = append(queue, p)
			}
		}
	}

	return false, nil, nil
}

// getParents returns all parent IDs for a given child spec
func (r *sqliteSpecDependencyRepository) getParents(ctx context.Context, childID string) ([]string, error) {
	query := "SELECT parent_id FROM spec_dependencies WHERE child_id = ?"
	rows, err := r.executor().QueryContext(ctx, query, childID)
	if err != nil {
		return nil, fmt.Errorf("failed to get spec dependencies: %w", err)
	}
	defer rows.Close()

	var parents []string
	for rows.Next() {
		var parentID string
		if err := rows.Scan(&parentID); err != nil {
			return nil, fmt.Errorf("failed to scan spec dependency: %w", err)
		}
		parents = append(parents, parentID)
	}

	return parents, rows.Err()
}
