package domain

import "fmt"

// ErrorCode represents a domain error code.
type ErrorCode string

const (
	ErrCodeTaskNotFound       ErrorCode = "TASK_NOT_FOUND"
	ErrCodeAlreadyClaimed     ErrorCode = "ALREADY_CLAIMED"
	ErrCodeNotOwner           ErrorCode = "NOT_OWNER"
	ErrCodeInvalidTransition  ErrorCode = "INVALID_TRANSITION"
	ErrCodeValidationFailed   ErrorCode = "VALIDATION_FAILED"
	ErrCodeCycleDetected      ErrorCode = "CYCLE_DETECTED"
	ErrCodeInternalError      ErrorCode = "INTERNAL_ERROR"
	ErrCodeProjectNotFound    ErrorCode = "PROJECT_NOT_FOUND"
	ErrCodeDependencyNotFound ErrorCode = "DEPENDENCY_NOT_FOUND"
)

// DomainError represents an error in the domain layer with context.
type DomainError struct {
	Code    ErrorCode
	Message string
	Context map[string]interface{}
}

func (e *DomainError) Error() string {
	return e.Message
}

// NewTaskNotFoundError creates a task not found error.
func NewTaskNotFoundError(taskID string) *DomainError {
	return &DomainError{
		Code:    ErrCodeTaskNotFound,
		Message: fmt.Sprintf("Task %s not found", taskID),
		Context: map[string]interface{}{"id": taskID},
	}
}

// NewAlreadyClaimedError creates an already claimed error.
func NewAlreadyClaimedError(claimedBy string, claimedAt string) *DomainError {
	return &DomainError{
		Code:    ErrCodeAlreadyClaimed,
		Message: "Task already claimed by another agent",
		Context: map[string]interface{}{
			"claimed_by": claimedBy,
			"claimed_at": claimedAt,
		},
	}
}

// NewNotOwnerError creates a not owner error.
func NewNotOwnerError(claimedBy string) *DomainError {
	return &DomainError{
		Code:    ErrCodeNotOwner,
		Message: "Task is claimed by another agent",
		Context: map[string]interface{}{"claimed_by": claimedBy},
	}
}

// NewInvalidTransitionError creates an invalid status transition error.
func NewInvalidTransitionError(from, to TaskStatus) *DomainError {
	return &DomainError{
		Code:    ErrCodeInvalidTransition,
		Message: fmt.Sprintf("Cannot transition from %s to %s", from, to),
		Context: map[string]interface{}{
			"from": string(from),
			"to":   string(to),
		},
	}
}

// NewValidationError creates a validation error.
func NewValidationError(details []string) *DomainError {
	return &DomainError{
		Code:    ErrCodeValidationFailed,
		Message: "Validation failed",
		Context: map[string]interface{}{"details": details},
	}
}

// NewCycleDetectedError creates a cycle detected error.
func NewCycleDetectedError(path []string) *DomainError {
	return &DomainError{
		Code:    ErrCodeCycleDetected,
		Message: "Adding this dependency would create a cycle",
		Context: map[string]interface{}{"path": path},
	}
}

// NewProjectNotFoundError creates a project not found error.
func NewProjectNotFoundError(project string) *DomainError {
	return &DomainError{
		Code:    ErrCodeProjectNotFound,
		Message: fmt.Sprintf("Project %s not found", project),
		Context: map[string]interface{}{"project": project},
	}
}

// NewDependencyNotFoundError creates a dependency not found error.
func NewDependencyNotFoundError(childID, parentID string) *DomainError {
	return &DomainError{
		Code:    ErrCodeDependencyNotFound,
		Message: fmt.Sprintf("Dependency from %s to %s not found", childID, parentID),
		Context: map[string]interface{}{
			"child_id":  childID,
			"parent_id": parentID,
		},
	}
}

// NewInternalError creates an internal error.
func NewInternalError(err error) *DomainError {
	return &DomainError{
		Code:    ErrCodeInternalError,
		Message: "An internal error occurred",
		Context: map[string]interface{}{},
	}
}
