package sqlite

import (
	"database/sql"
	"time"

	"github.com/airyra/airyra/internal/domain"
)

// SpecRepository handles spec persistence operations.
type SpecRepository struct {
	db *sql.DB
}

// NewSpecRepository creates a new SpecRepository.
func NewSpecRepository(db *sql.DB) *SpecRepository {
	return &SpecRepository{db: db}
}

// Create creates a new spec.
func (r *SpecRepository) Create(spec *domain.Spec) error {
	query := `
		INSERT INTO specs (id, title, description, manual_status, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?)
	`
	_, err := r.db.Exec(query,
		spec.ID,
		spec.Title,
		spec.Description,
		spec.ManualStatus,
		spec.CreatedAt.Format(time.RFC3339),
		spec.UpdatedAt.Format(time.RFC3339),
	)
	return err
}

// GetByID retrieves a spec by its ID with computed task counts.
func (r *SpecRepository) GetByID(id string) (*domain.Spec, error) {
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
	row := r.db.QueryRow(query, id)
	return r.scanSpec(row)
}

// List retrieves specs with pagination and optional status filter.
// Status is computed from task counts, so filtering happens via HAVING clause.
func (r *SpecRepository) List(status *domain.SpecStatus, page, perPage int) ([]*domain.Spec, int, error) {
	offset := (page - 1) * perPage

	// Build query with computed status
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
	if status != nil {
		var statusCondition string
		switch *status {
		case domain.SpecStatusCancelled:
			statusCondition = "WHERE s.manual_status = 'cancelled'"
		case domain.SpecStatusDraft:
			statusCondition = `WHERE s.manual_status IS NULL AND
				(SELECT COUNT(*) FROM tasks WHERE spec_id = s.id) = 0`
		case domain.SpecStatusDone:
			statusCondition = `WHERE s.manual_status IS NULL AND
				(SELECT COUNT(*) FROM tasks WHERE spec_id = s.id) > 0 AND
				(SELECT COUNT(*) FROM tasks WHERE spec_id = s.id) =
				(SELECT COUNT(*) FROM tasks WHERE spec_id = s.id AND status = 'done')`
		case domain.SpecStatusActive:
			statusCondition = `WHERE s.manual_status IS NULL AND
				(SELECT COUNT(*) FROM tasks WHERE spec_id = s.id) > 0 AND
				(SELECT COUNT(*) FROM tasks WHERE spec_id = s.id) !=
				(SELECT COUNT(*) FROM tasks WHERE spec_id = s.id AND status = 'done')`
		}

		// Count with filter
		countQuery := "SELECT COUNT(*) FROM (" + baseQuery + statusCondition + ")"
		var total int
		if err := r.db.QueryRow(countQuery).Scan(&total); err != nil {
			return nil, 0, err
		}

		// Fetch with filter
		query := baseQuery + statusCondition + " ORDER BY s.created_at DESC LIMIT ? OFFSET ?"
		rows, err := r.db.Query(query, perPage, offset)
		if err != nil {
			return nil, 0, err
		}
		defer rows.Close()

		specs, err := r.scanSpecs(rows)
		if err != nil {
			return nil, 0, err
		}
		return specs, total, nil
	}

	// No filter - simple query
	countQuery := "SELECT COUNT(*) FROM specs"
	var total int
	if err := r.db.QueryRow(countQuery).Scan(&total); err != nil {
		return nil, 0, err
	}

	query := baseQuery + " ORDER BY s.created_at DESC LIMIT ? OFFSET ?"
	rows, err := r.db.Query(query, perPage, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	specs, err := r.scanSpecs(rows)
	if err != nil {
		return nil, 0, err
	}
	return specs, total, nil
}

// ListReady retrieves specs that are ready to be worked on.
// A spec is ready if it has no pending dependencies (all parent specs are done).
func (r *SpecRepository) ListReady(page, perPage int) ([]*domain.Spec, int, error) {
	offset := (page - 1) * perPage

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
	if err := r.db.QueryRow(countQuery).Scan(&total); err != nil {
		return nil, 0, err
	}

	query += " ORDER BY s.created_at ASC LIMIT ? OFFSET ?"
	rows, err := r.db.Query(query, perPage, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	specs, err := r.scanSpecs(rows)
	if err != nil {
		return nil, 0, err
	}
	return specs, total, nil
}

// Update updates a spec's fields.
func (r *SpecRepository) Update(spec *domain.Spec) error {
	query := `
		UPDATE specs
		SET title = ?, description = ?, manual_status = ?, updated_at = ?
		WHERE id = ?
	`
	result, err := r.db.Exec(query,
		spec.Title,
		spec.Description,
		spec.ManualStatus,
		spec.UpdatedAt.Format(time.RFC3339),
		spec.ID,
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

// Delete deletes a spec by ID.
func (r *SpecRepository) Delete(id string) error {
	result, err := r.db.Exec("DELETE FROM specs WHERE id = ?", id)
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

// ListTasksBySpecID returns all tasks belonging to a spec.
func (r *SpecRepository) ListTasksBySpecID(specID string, page, perPage int) ([]*domain.Task, int, error) {
	offset := (page - 1) * perPage

	countQuery := "SELECT COUNT(*) FROM tasks WHERE spec_id = ?"
	var total int
	if err := r.db.QueryRow(countQuery, specID).Scan(&total); err != nil {
		return nil, 0, err
	}

	query := `
		SELECT id, parent_id, spec_id, title, description, status, priority, claimed_by, claimed_at, created_at, updated_at
		FROM tasks
		WHERE spec_id = ?
		ORDER BY priority ASC, created_at ASC
		LIMIT ? OFFSET ?
	`

	rows, err := r.db.Query(query, specID, perPage, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var tasks []*domain.Task
	for rows.Next() {
		task, err := scanTaskRowWithSpecID(rows)
		if err != nil {
			return nil, 0, err
		}
		tasks = append(tasks, task)
	}

	return tasks, total, rows.Err()
}

func (r *SpecRepository) scanSpec(row *sql.Row) (*domain.Spec, error) {
	var spec domain.Spec
	var description, manualStatus sql.NullString
	var createdAt, updatedAt string

	err := row.Scan(
		&spec.ID,
		&spec.Title,
		&description,
		&manualStatus,
		&createdAt,
		&updatedAt,
		&spec.TaskCount,
		&spec.DoneCount,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, err
		}
		return nil, err
	}

	if description.Valid {
		spec.Description = &description.String
	}
	if manualStatus.Valid {
		spec.ManualStatus = &manualStatus.String
	}
	spec.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
	spec.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAt)

	return &spec, nil
}

func (r *SpecRepository) scanSpecs(rows *sql.Rows) ([]*domain.Spec, error) {
	var specs []*domain.Spec
	for rows.Next() {
		var spec domain.Spec
		var description, manualStatus sql.NullString
		var createdAt, updatedAt string

		err := rows.Scan(
			&spec.ID,
			&spec.Title,
			&description,
			&manualStatus,
			&createdAt,
			&updatedAt,
			&spec.TaskCount,
			&spec.DoneCount,
		)
		if err != nil {
			return nil, err
		}

		if description.Valid {
			spec.Description = &description.String
		}
		if manualStatus.Valid {
			spec.ManualStatus = &manualStatus.String
		}
		spec.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
		spec.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAt)

		specs = append(specs, &spec)
	}
	return specs, rows.Err()
}

// scanTaskRowWithSpecID scans a task row including spec_id field.
func scanTaskRowWithSpecID(rows *sql.Rows) (*domain.Task, error) {
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
