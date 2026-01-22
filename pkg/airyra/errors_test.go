package airyra

import (
	"errors"
	"testing"
)

func TestErrorHelpers(t *testing.T) {
	tests := []struct {
		name    string
		err     error
		checker func(error) bool
		want    bool
	}{
		// TaskNotFound
		{
			name:    "IsTaskNotFound with task not found error",
			err:     newTaskNotFoundError("task-123"),
			checker: IsTaskNotFound,
			want:    true,
		},
		{
			name:    "IsTaskNotFound with different error",
			err:     newAlreadyClaimedError("agent", "now"),
			checker: IsTaskNotFound,
			want:    false,
		},
		{
			name:    "IsTaskNotFound with nil",
			err:     nil,
			checker: IsTaskNotFound,
			want:    false,
		},

		// AlreadyClaimed
		{
			name:    "IsAlreadyClaimed with already claimed error",
			err:     newAlreadyClaimedError("agent", "now"),
			checker: IsAlreadyClaimed,
			want:    true,
		},
		{
			name:    "IsAlreadyClaimed with different error",
			err:     newTaskNotFoundError("task-123"),
			checker: IsAlreadyClaimed,
			want:    false,
		},

		// NotOwner
		{
			name:    "IsNotOwner with not owner error",
			err:     newNotOwnerError("other-agent"),
			checker: IsNotOwner,
			want:    true,
		},
		{
			name:    "IsNotOwner with different error",
			err:     newAlreadyClaimedError("agent", "now"),
			checker: IsNotOwner,
			want:    false,
		},

		// InvalidTransition
		{
			name:    "IsInvalidTransition with invalid transition error",
			err:     newInvalidTransitionError("open", "done"),
			checker: IsInvalidTransition,
			want:    true,
		},
		{
			name:    "IsInvalidTransition with different error",
			err:     newTaskNotFoundError("task-123"),
			checker: IsInvalidTransition,
			want:    false,
		},

		// ValidationFailed
		{
			name:    "IsValidationFailed with validation error",
			err:     newValidationError([]string{"title is required"}),
			checker: IsValidationFailed,
			want:    true,
		},
		{
			name:    "IsValidationFailed with different error",
			err:     newTaskNotFoundError("task-123"),
			checker: IsValidationFailed,
			want:    false,
		},

		// CycleDetected
		{
			name:    "IsCycleDetected with cycle error",
			err:     newCycleDetectedError([]string{"a", "b", "c"}),
			checker: IsCycleDetected,
			want:    true,
		},
		{
			name:    "IsCycleDetected with different error",
			err:     newTaskNotFoundError("task-123"),
			checker: IsCycleDetected,
			want:    false,
		},

		// ProjectNotFound
		{
			name:    "IsProjectNotFound with project not found error",
			err:     newProjectNotFoundError("my-project"),
			checker: IsProjectNotFound,
			want:    true,
		},
		{
			name:    "IsProjectNotFound with different error",
			err:     newTaskNotFoundError("task-123"),
			checker: IsProjectNotFound,
			want:    false,
		},

		// DependencyNotFound
		{
			name:    "IsDependencyNotFound with dependency not found error",
			err:     newDependencyNotFoundError("child", "parent"),
			checker: IsDependencyNotFound,
			want:    true,
		},
		{
			name:    "IsDependencyNotFound with different error",
			err:     newTaskNotFoundError("task-123"),
			checker: IsDependencyNotFound,
			want:    false,
		},

		// ServerNotRunning
		{
			name:    "IsServerNotRunning with sentinel error",
			err:     ErrServerNotRunning,
			checker: IsServerNotRunning,
			want:    true,
		},
		{
			name:    "IsServerNotRunning with wrapped error",
			err:     errors.Join(ErrServerNotRunning, errors.New("connection refused")),
			checker: IsServerNotRunning,
			want:    true,
		},
		{
			name:    "IsServerNotRunning with different error",
			err:     newTaskNotFoundError("task-123"),
			checker: IsServerNotRunning,
			want:    false,
		},

		// ServerUnhealthy
		{
			name:    "IsServerUnhealthy with sentinel error",
			err:     ErrServerUnhealthy,
			checker: IsServerUnhealthy,
			want:    true,
		},
		{
			name:    "IsServerUnhealthy with different error",
			err:     ErrServerNotRunning,
			checker: IsServerUnhealthy,
			want:    false,
		},

		// Generic error
		{
			name:    "generic error with Error type",
			err:     errors.New("some error"),
			checker: IsTaskNotFound,
			want:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.checker(tt.err)
			if got != tt.want {
				t.Errorf("expected %v, got %v", tt.want, got)
			}
		})
	}
}

func TestErrorMessage(t *testing.T) {
	err := newTaskNotFoundError("task-123")
	if err.Error() != "Task task-123 not found" {
		t.Errorf("expected 'Task task-123 not found', got %q", err.Error())
	}

	err = newAlreadyClaimedError("agent-1", "2024-01-01")
	if err.Error() != "Task already claimed by another agent" {
		t.Errorf("expected 'Task already claimed by another agent', got %q", err.Error())
	}

	err = newCycleDetectedError([]string{"a", "b", "c"})
	if err.Error() != "Adding this dependency would create a cycle" {
		t.Errorf("expected 'Adding this dependency would create a cycle', got %q", err.Error())
	}
}

func TestErrorContext(t *testing.T) {
	err := newTaskNotFoundError("task-123")
	if err.Context["id"] != "task-123" {
		t.Errorf("expected context[id] = 'task-123', got %v", err.Context["id"])
	}

	err = newAlreadyClaimedError("agent-1", "2024-01-01")
	if err.Context["claimed_by"] != "agent-1" {
		t.Errorf("expected context[claimed_by] = 'agent-1', got %v", err.Context["claimed_by"])
	}
	if err.Context["claimed_at"] != "2024-01-01" {
		t.Errorf("expected context[claimed_at] = '2024-01-01', got %v", err.Context["claimed_at"])
	}

	err = newCycleDetectedError([]string{"a", "b", "c"})
	path, ok := err.Context["path"].([]string)
	if !ok {
		t.Errorf("expected context[path] to be []string, got %T", err.Context["path"])
	} else if len(path) != 3 || path[0] != "a" || path[1] != "b" || path[2] != "c" {
		t.Errorf("expected context[path] = [a, b, c], got %v", path)
	}
}
