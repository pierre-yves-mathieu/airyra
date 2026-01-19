package service

import (
	"database/sql"
	"time"

	"github.com/airyra/airyra/internal/domain"
	"github.com/airyra/airyra/internal/store/sqlite"
)

// TransitionService handles task status transitions.
type TransitionService struct {
	taskRepo  *sqlite.TaskRepository
	auditRepo *sqlite.AuditRepository
}

// NewTransitionService creates a new TransitionService.
func NewTransitionService(taskRepo *sqlite.TaskRepository, auditRepo *sqlite.AuditRepository) *TransitionService {
	return &TransitionService{
		taskRepo:  taskRepo,
		auditRepo: auditRepo,
	}
}

// Claim claims a task for an agent (open -> in_progress).
func (s *TransitionService) Claim(taskID, agentID string) (*domain.Task, error) {
	now := time.Now().UTC()

	task, err := s.taskRepo.AtomicClaim(taskID, agentID, now)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, domain.NewTaskNotFoundError(taskID)
		}
		return nil, domain.NewInternalError(err)
	}

	// Check if claim succeeded (task is now in_progress and claimed by this agent)
	if task.Status != domain.StatusInProgress || task.ClaimedBy == nil || *task.ClaimedBy != agentID {
		// Task was not open, return appropriate error
		if task.Status == domain.StatusInProgress && task.ClaimedBy != nil {
			claimedAt := ""
			if task.ClaimedAt != nil {
				claimedAt = task.ClaimedAt.Format(time.RFC3339)
			}
			return nil, domain.NewAlreadyClaimedError(*task.ClaimedBy, claimedAt)
		}
		return nil, domain.NewInvalidTransitionError(task.Status, domain.StatusInProgress)
	}

	// Log the claim
	s.auditRepo.Log(&domain.AuditEntry{
		TaskID:    taskID,
		Action:    "claim",
		Field:     strPtr("status"),
		OldValue:  strPtr(string(domain.StatusOpen)),
		NewValue:  strPtr(string(domain.StatusInProgress)),
		ChangedAt: now,
		ChangedBy: agentID,
	})

	return task, nil
}

// Complete marks a task as done (in_progress -> done).
// Only the claiming agent can complete the task.
func (s *TransitionService) Complete(taskID, agentID string) (*domain.Task, error) {
	task, err := s.taskRepo.GetByID(taskID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, domain.NewTaskNotFoundError(taskID)
		}
		return nil, domain.NewInternalError(err)
	}

	// Check if transition is valid
	if task.Status != domain.StatusInProgress {
		return nil, domain.NewInvalidTransitionError(task.Status, domain.StatusDone)
	}

	// Check if agent owns the task
	if task.ClaimedBy == nil || *task.ClaimedBy != agentID {
		if task.ClaimedBy != nil {
			return nil, domain.NewNotOwnerError(*task.ClaimedBy)
		}
		return nil, domain.NewNotOwnerError("unknown")
	}

	now := time.Now().UTC()
	oldStatus := task.Status
	task.Status = domain.StatusDone
	task.UpdatedAt = now

	if err := s.taskRepo.Update(task); err != nil {
		return nil, domain.NewInternalError(err)
	}

	// Log the completion
	s.auditRepo.Log(&domain.AuditEntry{
		TaskID:    taskID,
		Action:    "done",
		Field:     strPtr("status"),
		OldValue:  strPtr(string(oldStatus)),
		NewValue:  strPtr(string(domain.StatusDone)),
		ChangedAt: now,
		ChangedBy: agentID,
	})

	return task, nil
}

// Release releases a task (in_progress -> open).
// Only the claiming agent can release unless force is true.
func (s *TransitionService) Release(taskID, agentID string, force bool) (*domain.Task, error) {
	task, err := s.taskRepo.GetByID(taskID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, domain.NewTaskNotFoundError(taskID)
		}
		return nil, domain.NewInternalError(err)
	}

	// Check if transition is valid
	if task.Status != domain.StatusInProgress {
		return nil, domain.NewInvalidTransitionError(task.Status, domain.StatusOpen)
	}

	// Check if agent owns the task (unless force)
	if !force {
		if task.ClaimedBy == nil || *task.ClaimedBy != agentID {
			if task.ClaimedBy != nil {
				return nil, domain.NewNotOwnerError(*task.ClaimedBy)
			}
			return nil, domain.NewNotOwnerError("unknown")
		}
	}

	now := time.Now().UTC()
	oldStatus := task.Status
	task.Status = domain.StatusOpen
	task.ClaimedBy = nil
	task.ClaimedAt = nil
	task.UpdatedAt = now

	if err := s.taskRepo.Update(task); err != nil {
		return nil, domain.NewInternalError(err)
	}

	// Log the release
	s.auditRepo.Log(&domain.AuditEntry{
		TaskID:    taskID,
		Action:    "release",
		Field:     strPtr("status"),
		OldValue:  strPtr(string(oldStatus)),
		NewValue:  strPtr(string(domain.StatusOpen)),
		ChangedAt: now,
		ChangedBy: agentID,
	})

	return task, nil
}

// Block blocks a task (any -> blocked).
func (s *TransitionService) Block(taskID, agentID string) (*domain.Task, error) {
	task, err := s.taskRepo.GetByID(taskID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, domain.NewTaskNotFoundError(taskID)
		}
		return nil, domain.NewInternalError(err)
	}

	// Already blocked is a no-op
	if task.Status == domain.StatusBlocked {
		return task, nil
	}

	now := time.Now().UTC()
	oldStatus := task.Status
	task.Status = domain.StatusBlocked
	task.UpdatedAt = now

	if err := s.taskRepo.Update(task); err != nil {
		return nil, domain.NewInternalError(err)
	}

	// Log the block
	s.auditRepo.Log(&domain.AuditEntry{
		TaskID:    taskID,
		Action:    "block",
		Field:     strPtr("status"),
		OldValue:  strPtr(string(oldStatus)),
		NewValue:  strPtr(string(domain.StatusBlocked)),
		ChangedAt: now,
		ChangedBy: agentID,
	})

	return task, nil
}

// Unblock unblocks a task (blocked -> open).
func (s *TransitionService) Unblock(taskID, agentID string) (*domain.Task, error) {
	task, err := s.taskRepo.GetByID(taskID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, domain.NewTaskNotFoundError(taskID)
		}
		return nil, domain.NewInternalError(err)
	}

	// Check if transition is valid
	if task.Status != domain.StatusBlocked {
		return nil, domain.NewInvalidTransitionError(task.Status, domain.StatusOpen)
	}

	now := time.Now().UTC()
	task.Status = domain.StatusOpen
	task.UpdatedAt = now

	if err := s.taskRepo.Update(task); err != nil {
		return nil, domain.NewInternalError(err)
	}

	// Log the unblock
	s.auditRepo.Log(&domain.AuditEntry{
		TaskID:    taskID,
		Action:    "unblock",
		Field:     strPtr("status"),
		OldValue:  strPtr(string(domain.StatusBlocked)),
		NewValue:  strPtr(string(domain.StatusOpen)),
		ChangedAt: now,
		ChangedBy: agentID,
	})

	return task, nil
}
