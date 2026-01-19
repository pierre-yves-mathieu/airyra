package service

import (
	"database/sql"
	"time"

	"github.com/airyra/airyra/internal/domain"
	"github.com/airyra/airyra/internal/store/sqlite"
)

// AuditService handles audit log queries.
type AuditService struct {
	auditRepo *sqlite.AuditRepository
	taskRepo  *sqlite.TaskRepository
}

// NewAuditService creates a new AuditService.
func NewAuditService(auditRepo *sqlite.AuditRepository, taskRepo *sqlite.TaskRepository) *AuditService {
	return &AuditService{
		auditRepo: auditRepo,
		taskRepo:  taskRepo,
	}
}

// GetTaskHistory returns the audit history for a specific task.
func (s *AuditService) GetTaskHistory(taskID string) ([]*domain.AuditEntry, error) {
	// Verify task exists (or existed - we still want history for deleted tasks)
	// Actually, for deleted tasks we still want to show history, so we skip this check
	// and just return what we have
	entries, err := s.auditRepo.ListByTaskID(taskID)
	if err != nil {
		return nil, domain.NewInternalError(err)
	}

	// If no entries and task doesn't exist, return not found
	if len(entries) == 0 {
		if _, err := s.taskRepo.GetByID(taskID); err != nil {
			if err == sql.ErrNoRows {
				return nil, domain.NewTaskNotFoundError(taskID)
			}
			return nil, domain.NewInternalError(err)
		}
	}

	return entries, nil
}

// QueryInput contains the input for querying the audit log.
type QueryInput struct {
	Action    *string
	AgentID   *string
	StartTime *time.Time
	EndTime   *time.Time
	Page      int
	PerPage   int
}

// Query queries the audit log with filters.
func (s *AuditService) Query(input QueryInput) ([]*domain.AuditEntry, int, error) {
	entries, total, err := s.auditRepo.Query(sqlite.AuditQueryParams{
		Action:    input.Action,
		AgentID:   input.AgentID,
		StartTime: input.StartTime,
		EndTime:   input.EndTime,
		Page:      input.Page,
		PerPage:   input.PerPage,
	})
	if err != nil {
		return nil, 0, domain.NewInternalError(err)
	}
	return entries, total, nil
}
