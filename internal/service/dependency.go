package service

import (
	"database/sql"
	"time"

	"github.com/airyra/airyra/internal/domain"
	"github.com/airyra/airyra/internal/store/sqlite"
)

// DependencyService handles dependency business logic.
type DependencyService struct {
	depRepo   *sqlite.DependencyRepository
	taskRepo  *sqlite.TaskRepository
	auditRepo *sqlite.AuditRepository
}

// NewDependencyService creates a new DependencyService.
func NewDependencyService(depRepo *sqlite.DependencyRepository, taskRepo *sqlite.TaskRepository, auditRepo *sqlite.AuditRepository) *DependencyService {
	return &DependencyService{
		depRepo:   depRepo,
		taskRepo:  taskRepo,
		auditRepo: auditRepo,
	}
}

// Add adds a dependency (childID depends on parentID).
func (s *DependencyService) Add(childID, parentID, agentID string) error {
	// Validate child task exists
	if _, err := s.taskRepo.GetByID(childID); err != nil {
		if err == sql.ErrNoRows {
			return domain.NewTaskNotFoundError(childID)
		}
		return domain.NewInternalError(err)
	}

	// Validate parent task exists
	if _, err := s.taskRepo.GetByID(parentID); err != nil {
		if err == sql.ErrNoRows {
			return domain.NewTaskNotFoundError(parentID)
		}
		return domain.NewInternalError(err)
	}

	// Check for self-dependency
	if childID == parentID {
		return domain.NewValidationError([]string{"Cannot add self-dependency"})
	}

	// Check if dependency already exists
	exists, err := s.depRepo.Exists(childID, parentID)
	if err != nil {
		return domain.NewInternalError(err)
	}
	if exists {
		return nil // Idempotent - already exists
	}

	// Check for cycle
	cyclePath, err := s.depRepo.WouldCreateCycle(childID, parentID)
	if err != nil {
		return domain.NewInternalError(err)
	}
	if cyclePath != nil {
		return domain.NewCycleDetectedError(cyclePath)
	}

	// Add the dependency
	if err := s.depRepo.Add(childID, parentID); err != nil {
		return domain.NewInternalError(err)
	}

	// Log the action
	now := time.Now().UTC()
	s.auditRepo.Log(&domain.AuditEntry{
		TaskID:    childID,
		Action:    "add_dependency",
		NewValue:  &parentID,
		ChangedAt: now,
		ChangedBy: agentID,
	})

	return nil
}

// Remove removes a dependency.
func (s *DependencyService) Remove(childID, parentID, agentID string) error {
	if err := s.depRepo.Remove(childID, parentID); err != nil {
		if err == sql.ErrNoRows {
			return domain.NewDependencyNotFoundError(childID, parentID)
		}
		return domain.NewInternalError(err)
	}

	// Log the action
	now := time.Now().UTC()
	s.auditRepo.Log(&domain.AuditEntry{
		TaskID:    childID,
		Action:    "remove_dependency",
		OldValue:  &parentID,
		ChangedAt: now,
		ChangedBy: agentID,
	})

	return nil
}

// List lists all dependencies for a task.
func (s *DependencyService) List(taskID string) ([]*domain.Dependency, error) {
	// Verify task exists
	if _, err := s.taskRepo.GetByID(taskID); err != nil {
		if err == sql.ErrNoRows {
			return nil, domain.NewTaskNotFoundError(taskID)
		}
		return nil, domain.NewInternalError(err)
	}

	deps, err := s.depRepo.ListByChild(taskID)
	if err != nil {
		return nil, domain.NewInternalError(err)
	}
	return deps, nil
}
