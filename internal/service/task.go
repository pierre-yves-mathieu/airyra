package service

import (
	"database/sql"
	"time"

	"github.com/airyra/airyra/internal/domain"
	"github.com/airyra/airyra/internal/store/sqlite"
	"github.com/airyra/airyra/pkg/idgen"
)

// TaskService handles task business logic.
type TaskService struct {
	taskRepo  *sqlite.TaskRepository
	auditRepo *sqlite.AuditRepository
}

// NewTaskService creates a new TaskService.
func NewTaskService(taskRepo *sqlite.TaskRepository, auditRepo *sqlite.AuditRepository) *TaskService {
	return &TaskService{
		taskRepo:  taskRepo,
		auditRepo: auditRepo,
	}
}

// CreateTaskInput contains the input for creating a task.
type CreateTaskInput struct {
	ParentID    *string
	SpecID      *string
	Title       string
	Description *string
	Priority    *int
}

// Create creates a new task.
func (s *TaskService) Create(input CreateTaskInput, agentID string) (*domain.Task, error) {
	id, err := idgen.Generate()
	if err != nil {
		return nil, domain.NewInternalError(err)
	}

	now := time.Now().UTC()
	priority := 2
	if input.Priority != nil {
		priority = *input.Priority
	}

	task := &domain.Task{
		ID:          id,
		ParentID:    input.ParentID,
		SpecID:      input.SpecID,
		Title:       input.Title,
		Description: input.Description,
		Status:      domain.StatusOpen,
		Priority:    priority,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if err := s.taskRepo.Create(task); err != nil {
		return nil, domain.NewInternalError(err)
	}

	// Log creation
	s.auditRepo.Log(&domain.AuditEntry{
		TaskID:    task.ID,
		Action:    "create",
		ChangedAt: now,
		ChangedBy: agentID,
	})

	return task, nil
}

// Get retrieves a task by ID.
func (s *TaskService) Get(id string) (*domain.Task, error) {
	task, err := s.taskRepo.GetByID(id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, domain.NewTaskNotFoundError(id)
		}
		return nil, domain.NewInternalError(err)
	}
	return task, nil
}

// ListTasksInput contains the input for listing tasks.
type ListTasksInput struct {
	Status  *domain.TaskStatus
	Page    int
	PerPage int
}

// List retrieves tasks with pagination.
func (s *TaskService) List(input ListTasksInput) ([]*domain.Task, int, error) {
	tasks, total, err := s.taskRepo.List(input.Status, input.Page, input.PerPage)
	if err != nil {
		return nil, 0, domain.NewInternalError(err)
	}
	return tasks, total, nil
}

// ListReady retrieves ready tasks.
func (s *TaskService) ListReady(page, perPage int) ([]*domain.Task, int, error) {
	tasks, total, err := s.taskRepo.ListReady(page, perPage)
	if err != nil {
		return nil, 0, domain.NewInternalError(err)
	}
	return tasks, total, nil
}

// UpdateTaskInput contains the input for updating a task.
type UpdateTaskInput struct {
	Title       *string
	Description *string
	Priority    *int
	ParentID    *string
}

// Update updates a task.
func (s *TaskService) Update(id string, input UpdateTaskInput, agentID string) (*domain.Task, error) {
	task, err := s.taskRepo.GetByID(id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, domain.NewTaskNotFoundError(id)
		}
		return nil, domain.NewInternalError(err)
	}

	now := time.Now().UTC()

	// Track changes for audit
	if input.Title != nil && *input.Title != task.Title {
		s.auditRepo.Log(&domain.AuditEntry{
			TaskID:    id,
			Action:    "update",
			Field:     strPtr("title"),
			OldValue:  &task.Title,
			NewValue:  input.Title,
			ChangedAt: now,
			ChangedBy: agentID,
		})
		task.Title = *input.Title
	}

	if input.Description != nil {
		oldDesc := ""
		if task.Description != nil {
			oldDesc = *task.Description
		}
		if *input.Description != oldDesc {
			s.auditRepo.Log(&domain.AuditEntry{
				TaskID:    id,
				Action:    "update",
				Field:     strPtr("description"),
				OldValue:  task.Description,
				NewValue:  input.Description,
				ChangedAt: now,
				ChangedBy: agentID,
			})
			task.Description = input.Description
		}
	}

	if input.Priority != nil && *input.Priority != task.Priority {
		oldPriority := intToStr(task.Priority)
		newPriority := intToStr(*input.Priority)
		s.auditRepo.Log(&domain.AuditEntry{
			TaskID:    id,
			Action:    "update",
			Field:     strPtr("priority"),
			OldValue:  &oldPriority,
			NewValue:  &newPriority,
			ChangedAt: now,
			ChangedBy: agentID,
		})
		task.Priority = *input.Priority
	}

	task.UpdatedAt = now

	if err := s.taskRepo.Update(task); err != nil {
		return nil, domain.NewInternalError(err)
	}

	return task, nil
}

// Delete deletes a task.
func (s *TaskService) Delete(id string, agentID string) error {
	// Check if task exists
	_, err := s.taskRepo.GetByID(id)
	if err != nil {
		if err == sql.ErrNoRows {
			return domain.NewTaskNotFoundError(id)
		}
		return domain.NewInternalError(err)
	}

	// Log deletion before deleting
	now := time.Now().UTC()
	s.auditRepo.Log(&domain.AuditEntry{
		TaskID:    id,
		Action:    "delete",
		ChangedAt: now,
		ChangedBy: agentID,
	})

	if err := s.taskRepo.Delete(id); err != nil {
		return domain.NewInternalError(err)
	}

	return nil
}

func strPtr(s string) *string {
	return &s
}

func intToStr(i int) string {
	return string(rune('0' + i))
}
