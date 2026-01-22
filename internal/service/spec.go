package service

import (
	"database/sql"
	"time"

	"github.com/airyra/airyra/internal/domain"
	"github.com/airyra/airyra/internal/store/sqlite"
	"github.com/airyra/airyra/pkg/idgen"
)

// SpecService handles spec business logic.
type SpecService struct {
	specRepo  *sqlite.SpecRepository
	auditRepo *sqlite.AuditRepository
}

// NewSpecService creates a new SpecService.
func NewSpecService(specRepo *sqlite.SpecRepository, auditRepo *sqlite.AuditRepository) *SpecService {
	return &SpecService{
		specRepo:  specRepo,
		auditRepo: auditRepo,
	}
}

// CreateSpecInput contains the input for creating a spec.
type CreateSpecInput struct {
	Title       string
	Description *string
}

// Create creates a new spec.
func (s *SpecService) Create(input CreateSpecInput, agentID string) (*domain.Spec, error) {
	id, err := idgen.GenerateWithPrefix("sp")
	if err != nil {
		return nil, domain.NewInternalError(err)
	}

	now := time.Now().UTC()
	spec := &domain.Spec{
		ID:          id,
		Title:       input.Title,
		Description: input.Description,
		TaskCount:   0,
		DoneCount:   0,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if err := s.specRepo.Create(spec); err != nil {
		return nil, domain.NewInternalError(err)
	}

	return spec, nil
}

// Get retrieves a spec by ID.
func (s *SpecService) Get(id string) (*domain.Spec, error) {
	spec, err := s.specRepo.GetByID(id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, domain.NewSpecNotFoundError(id)
		}
		return nil, domain.NewInternalError(err)
	}
	return spec, nil
}

// ListSpecsInput contains the input for listing specs.
type ListSpecsInput struct {
	Status  *domain.SpecStatus
	Page    int
	PerPage int
}

// List retrieves specs with pagination.
func (s *SpecService) List(input ListSpecsInput) ([]*domain.Spec, int, error) {
	specs, total, err := s.specRepo.List(input.Status, input.Page, input.PerPage)
	if err != nil {
		return nil, 0, domain.NewInternalError(err)
	}
	return specs, total, nil
}

// ListReady retrieves ready specs.
func (s *SpecService) ListReady(page, perPage int) ([]*domain.Spec, int, error) {
	specs, total, err := s.specRepo.ListReady(page, perPage)
	if err != nil {
		return nil, 0, domain.NewInternalError(err)
	}
	return specs, total, nil
}

// UpdateSpecInput contains the input for updating a spec.
type UpdateSpecInput struct {
	Title       *string
	Description *string
}

// Update updates a spec.
func (s *SpecService) Update(id string, input UpdateSpecInput, agentID string) (*domain.Spec, error) {
	spec, err := s.specRepo.GetByID(id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, domain.NewSpecNotFoundError(id)
		}
		return nil, domain.NewInternalError(err)
	}

	now := time.Now().UTC()

	if input.Title != nil && *input.Title != spec.Title {
		spec.Title = *input.Title
	}

	if input.Description != nil {
		spec.Description = input.Description
	}

	spec.UpdatedAt = now

	if err := s.specRepo.Update(spec); err != nil {
		return nil, domain.NewInternalError(err)
	}

	return spec, nil
}

// Cancel cancels a spec.
func (s *SpecService) Cancel(id string, agentID string) (*domain.Spec, error) {
	spec, err := s.specRepo.GetByID(id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, domain.NewSpecNotFoundError(id)
		}
		return nil, domain.NewInternalError(err)
	}

	if spec.IsCancelled() {
		return nil, domain.NewSpecAlreadyCancelledError(id)
	}

	spec.Cancel()
	spec.UpdatedAt = time.Now().UTC()

	if err := s.specRepo.Update(spec); err != nil {
		return nil, domain.NewInternalError(err)
	}

	return spec, nil
}

// Reopen reopens a cancelled spec.
func (s *SpecService) Reopen(id string, agentID string) (*domain.Spec, error) {
	spec, err := s.specRepo.GetByID(id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, domain.NewSpecNotFoundError(id)
		}
		return nil, domain.NewInternalError(err)
	}

	if !spec.IsCancelled() {
		return nil, domain.NewSpecNotCancelledError(id)
	}

	spec.Reopen()
	spec.UpdatedAt = time.Now().UTC()

	if err := s.specRepo.Update(spec); err != nil {
		return nil, domain.NewInternalError(err)
	}

	return spec, nil
}

// Delete deletes a spec.
func (s *SpecService) Delete(id string, agentID string) error {
	// Check if spec exists
	_, err := s.specRepo.GetByID(id)
	if err != nil {
		if err == sql.ErrNoRows {
			return domain.NewSpecNotFoundError(id)
		}
		return domain.NewInternalError(err)
	}

	if err := s.specRepo.Delete(id); err != nil {
		return domain.NewInternalError(err)
	}

	return nil
}

// ListTasks lists tasks belonging to a spec.
func (s *SpecService) ListTasks(specID string, page, perPage int) ([]*domain.Task, int, error) {
	// Check if spec exists
	_, err := s.specRepo.GetByID(specID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, 0, domain.NewSpecNotFoundError(specID)
		}
		return nil, 0, domain.NewInternalError(err)
	}

	tasks, total, err := s.specRepo.ListTasksBySpecID(specID, page, perPage)
	if err != nil {
		return nil, 0, domain.NewInternalError(err)
	}
	return tasks, total, nil
}
