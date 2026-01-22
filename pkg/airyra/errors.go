package airyra

import (
	"errors"
	"fmt"
)

// Sentinel errors for connection-related issues.
var (
	// ErrServerNotRunning indicates the server is not reachable.
	ErrServerNotRunning = errors.New("server is not running or unreachable")
	// ErrServerUnhealthy indicates the health check failed.
	ErrServerUnhealthy = errors.New("server health check failed")
)

// ErrorCode represents a domain error code from the API.
type ErrorCode string

const (
	ErrCodeTaskNotFound           ErrorCode = "TASK_NOT_FOUND"
	ErrCodeAlreadyClaimed         ErrorCode = "ALREADY_CLAIMED"
	ErrCodeNotOwner               ErrorCode = "NOT_OWNER"
	ErrCodeInvalidTransition      ErrorCode = "INVALID_TRANSITION"
	ErrCodeValidationFailed       ErrorCode = "VALIDATION_FAILED"
	ErrCodeCycleDetected          ErrorCode = "CYCLE_DETECTED"
	ErrCodeInternalError          ErrorCode = "INTERNAL_ERROR"
	ErrCodeProjectNotFound        ErrorCode = "PROJECT_NOT_FOUND"
	ErrCodeDependencyNotFound     ErrorCode = "DEPENDENCY_NOT_FOUND"
	ErrCodeSpecNotFound           ErrorCode = "SPEC_NOT_FOUND"
	ErrCodeSpecAlreadyCancelled   ErrorCode = "SPEC_ALREADY_CANCELLED"
	ErrCodeSpecNotCancelled       ErrorCode = "SPEC_NOT_CANCELLED"
	ErrCodeSpecDepNotFound        ErrorCode = "SPEC_DEPENDENCY_NOT_FOUND"
)

// Error represents an error response from the Airyra API.
type Error struct {
	Code    ErrorCode
	Message string
	Context map[string]interface{}
}

func (e *Error) Error() string {
	return e.Message
}

// apiErrorResponse wraps the error in the API response format.
type apiErrorResponse struct {
	Error apiError `json:"error"`
}

// apiError is the JSON structure for an API error.
type apiError struct {
	Code    string                 `json:"code"`
	Message string                 `json:"message"`
	Context map[string]interface{} `json:"context,omitempty"`
}

// Helper functions to check error types.

// IsTaskNotFound returns true if the error indicates a task was not found.
func IsTaskNotFound(err error) bool {
	return hasErrorCode(err, ErrCodeTaskNotFound)
}

// IsAlreadyClaimed returns true if the error indicates a task is already claimed.
func IsAlreadyClaimed(err error) bool {
	return hasErrorCode(err, ErrCodeAlreadyClaimed)
}

// IsNotOwner returns true if the error indicates the agent is not the task owner.
func IsNotOwner(err error) bool {
	return hasErrorCode(err, ErrCodeNotOwner)
}

// IsInvalidTransition returns true if the error indicates an invalid status transition.
func IsInvalidTransition(err error) bool {
	return hasErrorCode(err, ErrCodeInvalidTransition)
}

// IsValidationFailed returns true if the error indicates validation failed.
func IsValidationFailed(err error) bool {
	return hasErrorCode(err, ErrCodeValidationFailed)
}

// IsCycleDetected returns true if the error indicates a dependency cycle was detected.
func IsCycleDetected(err error) bool {
	return hasErrorCode(err, ErrCodeCycleDetected)
}

// IsProjectNotFound returns true if the error indicates a project was not found.
func IsProjectNotFound(err error) bool {
	return hasErrorCode(err, ErrCodeProjectNotFound)
}

// IsDependencyNotFound returns true if the error indicates a dependency was not found.
func IsDependencyNotFound(err error) bool {
	return hasErrorCode(err, ErrCodeDependencyNotFound)
}

// IsSpecNotFound returns true if the error indicates a spec was not found.
func IsSpecNotFound(err error) bool {
	return hasErrorCode(err, ErrCodeSpecNotFound)
}

// IsSpecAlreadyCancelled returns true if the error indicates a spec is already cancelled.
func IsSpecAlreadyCancelled(err error) bool {
	return hasErrorCode(err, ErrCodeSpecAlreadyCancelled)
}

// IsSpecNotCancelled returns true if the error indicates a spec is not cancelled.
func IsSpecNotCancelled(err error) bool {
	return hasErrorCode(err, ErrCodeSpecNotCancelled)
}

// IsSpecDependencyNotFound returns true if the error indicates a spec dependency was not found.
func IsSpecDependencyNotFound(err error) bool {
	return hasErrorCode(err, ErrCodeSpecDepNotFound)
}

// IsServerNotRunning returns true if the error indicates the server is not running.
func IsServerNotRunning(err error) bool {
	return errors.Is(err, ErrServerNotRunning)
}

// IsServerUnhealthy returns true if the error indicates the server is unhealthy.
func IsServerUnhealthy(err error) bool {
	return errors.Is(err, ErrServerUnhealthy)
}

// hasErrorCode checks if the error has the given error code.
func hasErrorCode(err error, code ErrorCode) bool {
	if err == nil {
		return false
	}
	var apiErr *Error
	if errors.As(err, &apiErr) {
		return apiErr.Code == code
	}
	return false
}

// newTaskNotFoundError creates a task not found error.
func newTaskNotFoundError(taskID string) *Error {
	return &Error{
		Code:    ErrCodeTaskNotFound,
		Message: fmt.Sprintf("Task %s not found", taskID),
		Context: map[string]interface{}{"id": taskID},
	}
}

// newAlreadyClaimedError creates an already claimed error.
func newAlreadyClaimedError(claimedBy, claimedAt string) *Error {
	return &Error{
		Code:    ErrCodeAlreadyClaimed,
		Message: "Task already claimed by another agent",
		Context: map[string]interface{}{
			"claimed_by": claimedBy,
			"claimed_at": claimedAt,
		},
	}
}

// newNotOwnerError creates a not owner error.
func newNotOwnerError(claimedBy string) *Error {
	return &Error{
		Code:    ErrCodeNotOwner,
		Message: "Task is claimed by another agent",
		Context: map[string]interface{}{"claimed_by": claimedBy},
	}
}

// newInvalidTransitionError creates an invalid status transition error.
func newInvalidTransitionError(from, to string) *Error {
	return &Error{
		Code:    ErrCodeInvalidTransition,
		Message: fmt.Sprintf("Cannot transition from %s to %s", from, to),
		Context: map[string]interface{}{
			"from": from,
			"to":   to,
		},
	}
}

// newValidationError creates a validation error.
func newValidationError(details []string) *Error {
	return &Error{
		Code:    ErrCodeValidationFailed,
		Message: "Validation failed",
		Context: map[string]interface{}{"details": details},
	}
}

// newCycleDetectedError creates a cycle detected error.
func newCycleDetectedError(path []string) *Error {
	return &Error{
		Code:    ErrCodeCycleDetected,
		Message: "Adding this dependency would create a cycle",
		Context: map[string]interface{}{"path": path},
	}
}

// newProjectNotFoundError creates a project not found error.
func newProjectNotFoundError(project string) *Error {
	return &Error{
		Code:    ErrCodeProjectNotFound,
		Message: fmt.Sprintf("Project %s not found", project),
		Context: map[string]interface{}{"project": project},
	}
}

// newDependencyNotFoundError creates a dependency not found error.
func newDependencyNotFoundError(childID, parentID string) *Error {
	return &Error{
		Code:    ErrCodeDependencyNotFound,
		Message: fmt.Sprintf("Dependency from %s to %s not found", childID, parentID),
		Context: map[string]interface{}{
			"child_id":  childID,
			"parent_id": parentID,
		},
	}
}
