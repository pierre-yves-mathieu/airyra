package sqlite

import (
	"database/sql"
	"time"

	"github.com/airyra/airyra/internal/domain"
)

// TaskRepository handles task persistence operations.
type TaskRepository struct {
	db *sql.DB
}

// NewTaskRepository creates a new TaskRepository.
func NewTaskRepository(db *sql.DB) *TaskRepository {
	return &TaskRepository{db: db}
}

// Create creates a new task.
func (r *TaskRepository) Create(task *domain.Task) error {
	query := `
		INSERT INTO tasks (id, parent_id, spec_id, title, description, status, priority, claimed_by, claimed_at, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	var claimedAt *string
	if task.ClaimedAt != nil {
		t := task.ClaimedAt.Format(time.RFC3339)
		claimedAt = &t
	}

	_, err := r.db.Exec(query,
		task.ID,
		task.ParentID,
		task.SpecID,
		task.Title,
		task.Description,
		string(task.Status),
		task.Priority,
		task.ClaimedBy,
		claimedAt,
		task.CreatedAt.Format(time.RFC3339),
		task.UpdatedAt.Format(time.RFC3339),
	)
	return err
}

// GetByID retrieves a task by its ID.
func (r *TaskRepository) GetByID(id string) (*domain.Task, error) {
	query := `
		SELECT id, parent_id, spec_id, title, description, status, priority, claimed_by, claimed_at, created_at, updated_at
		FROM tasks WHERE id = ?
	`
	row := r.db.QueryRow(query, id)
	return r.scanTask(row)
}

// List retrieves tasks with pagination and optional status filter.
func (r *TaskRepository) List(status *domain.TaskStatus, page, perPage int) ([]*domain.Task, int, error) {
	offset := (page - 1) * perPage

	// Count total
	countQuery := "SELECT COUNT(*) FROM tasks"
	args := []interface{}{}
	if status != nil {
		countQuery += " WHERE status = ?"
		args = append(args, string(*status))
	}

	var total int
	if err := r.db.QueryRow(countQuery, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	// Fetch tasks
	query := `
		SELECT id, parent_id, spec_id, title, description, status, priority, claimed_by, claimed_at, created_at, updated_at
		FROM tasks
	`
	if status != nil {
		query += " WHERE status = ?"
	}
	query += " ORDER BY priority ASC, created_at ASC LIMIT ? OFFSET ?"

	fetchArgs := args
	fetchArgs = append(fetchArgs, perPage, offset)

	rows, err := r.db.Query(query, fetchArgs...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var tasks []*domain.Task
	for rows.Next() {
		task, err := r.scanTaskRows(rows)
		if err != nil {
			return nil, 0, err
		}
		tasks = append(tasks, task)
	}

	return tasks, total, rows.Err()
}

// ListReady retrieves tasks that are ready to be worked on.
// A task is ready if it's open and has no incomplete dependencies.
func (r *TaskRepository) ListReady(page, perPage int) ([]*domain.Task, int, error) {
	offset := (page - 1) * perPage

	// Count ready tasks
	countQuery := `
		SELECT COUNT(*) FROM tasks t
		WHERE t.status = 'open'
		AND NOT EXISTS (
			SELECT 1 FROM dependencies d
			JOIN tasks dep ON d.parent_id = dep.id
			WHERE d.child_id = t.id AND dep.status != 'done'
		)
	`
	var total int
	if err := r.db.QueryRow(countQuery).Scan(&total); err != nil {
		return nil, 0, err
	}

	// Fetch ready tasks
	query := `
		SELECT t.id, t.parent_id, t.spec_id, t.title, t.description, t.status, t.priority, t.claimed_by, t.claimed_at, t.created_at, t.updated_at
		FROM tasks t
		WHERE t.status = 'open'
		AND NOT EXISTS (
			SELECT 1 FROM dependencies d
			JOIN tasks dep ON d.parent_id = dep.id
			WHERE d.child_id = t.id AND dep.status != 'done'
		)
		ORDER BY t.priority ASC, t.created_at ASC
		LIMIT ? OFFSET ?
	`

	rows, err := r.db.Query(query, perPage, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var tasks []*domain.Task
	for rows.Next() {
		task, err := r.scanTaskRows(rows)
		if err != nil {
			return nil, 0, err
		}
		tasks = append(tasks, task)
	}

	return tasks, total, rows.Err()
}

// Update updates a task's fields.
func (r *TaskRepository) Update(task *domain.Task) error {
	query := `
		UPDATE tasks
		SET parent_id = ?, spec_id = ?, title = ?, description = ?, status = ?, priority = ?, claimed_by = ?, claimed_at = ?, updated_at = ?
		WHERE id = ?
	`
	var claimedAt *string
	if task.ClaimedAt != nil {
		t := task.ClaimedAt.Format(time.RFC3339)
		claimedAt = &t
	}

	result, err := r.db.Exec(query,
		task.ParentID,
		task.SpecID,
		task.Title,
		task.Description,
		string(task.Status),
		task.Priority,
		task.ClaimedBy,
		claimedAt,
		task.UpdatedAt.Format(time.RFC3339),
		task.ID,
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

// Delete deletes a task by ID.
func (r *TaskRepository) Delete(id string) error {
	result, err := r.db.Exec("DELETE FROM tasks WHERE id = ?", id)
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

// AtomicClaim attempts to claim a task atomically.
// Returns the updated task if successful, or an error if the task cannot be claimed.
func (r *TaskRepository) AtomicClaim(taskID, agentID string, now time.Time) (*domain.Task, error) {
	nowStr := now.Format(time.RFC3339)

	result, err := r.db.Exec(`
		UPDATE tasks
		SET status = 'in_progress',
		    claimed_by = ?,
		    claimed_at = ?,
		    updated_at = ?
		WHERE id = ? AND status = 'open'
	`, agentID, nowStr, nowStr, taskID)
	if err != nil {
		return nil, err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return nil, err
	}
	if rowsAffected == 0 {
		// Task was not open - fetch current state to provide error context
		return r.GetByID(taskID)
	}

	return r.GetByID(taskID)
}

func (r *TaskRepository) scanTask(row *sql.Row) (*domain.Task, error) {
	var task domain.Task
	var parentID, specID, description, claimedBy, claimedAt sql.NullString
	var status string
	var createdAt, updatedAt string

	err := row.Scan(
		&task.ID,
		&parentID,
		&specID,
		&task.Title,
		&description,
		&status,
		&task.Priority,
		&claimedBy,
		&claimedAt,
		&createdAt,
		&updatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, err
		}
		return nil, err
	}

	task.Status = domain.TaskStatus(status)
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
		t, _ := time.Parse(time.RFC3339, claimedAt.String)
		task.ClaimedAt = &t
	}
	task.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
	task.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAt)

	return &task, nil
}

func (r *TaskRepository) scanTaskRows(rows *sql.Rows) (*domain.Task, error) {
	var task domain.Task
	var parentID, specID, description, claimedBy, claimedAt sql.NullString
	var status string
	var createdAt, updatedAt string

	err := rows.Scan(
		&task.ID,
		&parentID,
		&specID,
		&task.Title,
		&description,
		&status,
		&task.Priority,
		&claimedBy,
		&claimedAt,
		&createdAt,
		&updatedAt,
	)
	if err != nil {
		return nil, err
	}

	task.Status = domain.TaskStatus(status)
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
		t, _ := time.Parse(time.RFC3339, claimedAt.String)
		task.ClaimedAt = &t
	}
	task.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
	task.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAt)

	return &task, nil
}
