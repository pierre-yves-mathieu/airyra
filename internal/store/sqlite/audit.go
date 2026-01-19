package sqlite

import (
	"database/sql"
	"time"

	"github.com/airyra/airyra/internal/domain"
)

// AuditRepository handles audit log persistence operations.
type AuditRepository struct {
	db *sql.DB
}

// NewAuditRepository creates a new AuditRepository.
func NewAuditRepository(db *sql.DB) *AuditRepository {
	return &AuditRepository{db: db}
}

// Log creates an audit log entry.
func (r *AuditRepository) Log(entry *domain.AuditEntry) error {
	_, err := r.db.Exec(`
		INSERT INTO audit_log (task_id, action, field, old_value, new_value, changed_at, changed_by)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`,
		entry.TaskID,
		entry.Action,
		entry.Field,
		entry.OldValue,
		entry.NewValue,
		entry.ChangedAt.Format(time.RFC3339),
		entry.ChangedBy,
	)
	return err
}

// ListByTaskID returns all audit entries for a task.
func (r *AuditRepository) ListByTaskID(taskID string) ([]*domain.AuditEntry, error) {
	rows, err := r.db.Query(`
		SELECT id, task_id, action, field, old_value, new_value, changed_at, changed_by
		FROM audit_log
		WHERE task_id = ?
		ORDER BY changed_at DESC
	`, taskID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return r.scanEntries(rows)
}

// AuditQueryParams contains parameters for querying the audit log.
type AuditQueryParams struct {
	Action    *string
	AgentID   *string
	StartTime *time.Time
	EndTime   *time.Time
	Page      int
	PerPage   int
}

// Query queries the audit log with filters and pagination.
func (r *AuditRepository) Query(params AuditQueryParams) ([]*domain.AuditEntry, int, error) {
	offset := (params.Page - 1) * params.PerPage

	// Build query with conditions
	baseQuery := "FROM audit_log WHERE 1=1"
	args := []interface{}{}

	if params.Action != nil {
		baseQuery += " AND action = ?"
		args = append(args, *params.Action)
	}
	if params.AgentID != nil {
		baseQuery += " AND changed_by = ?"
		args = append(args, *params.AgentID)
	}
	if params.StartTime != nil {
		baseQuery += " AND changed_at >= ?"
		args = append(args, params.StartTime.Format(time.RFC3339))
	}
	if params.EndTime != nil {
		baseQuery += " AND changed_at <= ?"
		args = append(args, params.EndTime.Format(time.RFC3339))
	}

	// Count total
	var total int
	countQuery := "SELECT COUNT(*) " + baseQuery
	if err := r.db.QueryRow(countQuery, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	// Fetch entries
	selectQuery := "SELECT id, task_id, action, field, old_value, new_value, changed_at, changed_by " + baseQuery
	selectQuery += " ORDER BY changed_at DESC LIMIT ? OFFSET ?"
	args = append(args, params.PerPage, offset)

	rows, err := r.db.Query(selectQuery, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	entries, err := r.scanEntries(rows)
	if err != nil {
		return nil, 0, err
	}

	return entries, total, nil
}

func (r *AuditRepository) scanEntries(rows *sql.Rows) ([]*domain.AuditEntry, error) {
	var entries []*domain.AuditEntry
	for rows.Next() {
		var entry domain.AuditEntry
		var field, oldValue, newValue sql.NullString
		var changedAt string

		err := rows.Scan(
			&entry.ID,
			&entry.TaskID,
			&entry.Action,
			&field,
			&oldValue,
			&newValue,
			&changedAt,
			&entry.ChangedBy,
		)
		if err != nil {
			return nil, err
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
		entry.ChangedAt, _ = time.Parse(time.RFC3339, changedAt)

		entries = append(entries, &entry)
	}
	return entries, rows.Err()
}
