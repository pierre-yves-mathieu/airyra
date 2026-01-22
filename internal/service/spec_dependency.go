package service

import (
	"database/sql"
	"time"

	"github.com/airyra/airyra/internal/domain"
	"github.com/airyra/airyra/internal/store/sqlite"
)

// SpecDependencyService handles spec dependency business logic.
type SpecDependencyService struct {
	depRepo   *sqlite.SpecDependencyRepository
	specRepo  *sqlite.SpecRepository
	auditRepo *sqlite.AuditRepository
}

// NewSpecDependencyService creates a new SpecDependencyService.
func NewSpecDependencyService(depRepo *sqlite.SpecDependencyRepository, specRepo *sqlite.SpecRepository, auditRepo *sqlite.AuditRepository) *SpecDependencyService {
	return &SpecDependencyService{
		depRepo:   depRepo,
		specRepo:  specRepo,
		auditRepo: auditRepo,
	}
}

// Add adds a dependency (childID depends on parentID).
func (s *SpecDependencyService) Add(childID, parentID, agentID string) error {
	// Validate child spec exists
	if _, err := s.specRepo.GetByID(childID); err != nil {
		if err == sql.ErrNoRows {
			return domain.NewSpecNotFoundError(childID)
		}
		return domain.NewInternalError(err)
	}

	// Validate parent spec exists
	if _, err := s.specRepo.GetByID(parentID); err != nil {
		if err == sql.ErrNoRows {
			return domain.NewSpecNotFoundError(parentID)
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

	// Log the action (optional - using audit for spec changes)
	now := time.Now().UTC()
	_ = now
	_ = agentID

	return nil
}

// Remove removes a dependency.
func (s *SpecDependencyService) Remove(childID, parentID, agentID string) error {
	if err := s.depRepo.Remove(childID, parentID); err != nil {
		if err == sql.ErrNoRows {
			return domain.NewSpecDependencyNotFoundError(childID, parentID)
		}
		return domain.NewInternalError(err)
	}

	return nil
}

// List lists all dependencies for a spec.
func (s *SpecDependencyService) List(specID string) ([]*domain.SpecDependency, error) {
	// Verify spec exists
	if _, err := s.specRepo.GetByID(specID); err != nil {
		if err == sql.ErrNoRows {
			return nil, domain.NewSpecNotFoundError(specID)
		}
		return nil, domain.NewInternalError(err)
	}

	deps, err := s.depRepo.ListByChild(specID)
	if err != nil {
		return nil, domain.NewInternalError(err)
	}
	return deps, nil
}
