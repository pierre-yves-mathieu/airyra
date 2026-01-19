package domain

import (
	"strings"
	"testing"
)

func TestDomainError_Error(t *testing.T) {
	err := &DomainError{
		Code:    ErrCodeTaskNotFound,
		Message: "Test message",
		Context: map[string]interface{}{"key": "value"},
	}

	if err.Error() != "Test message" {
		t.Errorf("DomainError.Error() = %v, want %v", err.Error(), "Test message")
	}
}

func TestNewTaskNotFoundError(t *testing.T) {
	taskID := "ar-1234"
	err := NewTaskNotFoundError(taskID)

	if err.Code != ErrCodeTaskNotFound {
		t.Errorf("Code = %v, want %v", err.Code, ErrCodeTaskNotFound)
	}
	if !strings.Contains(err.Message, taskID) {
		t.Errorf("Message should contain task ID, got: %v", err.Message)
	}
	if err.Context["id"] != taskID {
		t.Errorf("Context[id] = %v, want %v", err.Context["id"], taskID)
	}
}

func TestNewAlreadyClaimedError(t *testing.T) {
	claimedBy := "agent-1"
	claimedAt := "2024-01-15T10:00:00Z"
	err := NewAlreadyClaimedError(claimedBy, claimedAt)

	if err.Code != ErrCodeAlreadyClaimed {
		t.Errorf("Code = %v, want %v", err.Code, ErrCodeAlreadyClaimed)
	}
	if err.Context["claimed_by"] != claimedBy {
		t.Errorf("Context[claimed_by] = %v, want %v", err.Context["claimed_by"], claimedBy)
	}
	if err.Context["claimed_at"] != claimedAt {
		t.Errorf("Context[claimed_at] = %v, want %v", err.Context["claimed_at"], claimedAt)
	}
}

func TestNewNotOwnerError(t *testing.T) {
	claimedBy := "agent-1"
	err := NewNotOwnerError(claimedBy)

	if err.Code != ErrCodeNotOwner {
		t.Errorf("Code = %v, want %v", err.Code, ErrCodeNotOwner)
	}
	if err.Context["claimed_by"] != claimedBy {
		t.Errorf("Context[claimed_by] = %v, want %v", err.Context["claimed_by"], claimedBy)
	}
}

func TestNewInvalidTransitionError(t *testing.T) {
	from := StatusOpen
	to := StatusDone
	err := NewInvalidTransitionError(from, to)

	if err.Code != ErrCodeInvalidTransition {
		t.Errorf("Code = %v, want %v", err.Code, ErrCodeInvalidTransition)
	}
	if !strings.Contains(err.Message, string(from)) {
		t.Errorf("Message should contain 'from' status, got: %v", err.Message)
	}
	if !strings.Contains(err.Message, string(to)) {
		t.Errorf("Message should contain 'to' status, got: %v", err.Message)
	}
	if err.Context["from"] != string(from) {
		t.Errorf("Context[from] = %v, want %v", err.Context["from"], string(from))
	}
	if err.Context["to"] != string(to) {
		t.Errorf("Context[to] = %v, want %v", err.Context["to"], string(to))
	}
}

func TestNewValidationError(t *testing.T) {
	details := []string{"field1 is required", "field2 is invalid"}
	err := NewValidationError(details)

	if err.Code != ErrCodeValidationFailed {
		t.Errorf("Code = %v, want %v", err.Code, ErrCodeValidationFailed)
	}
	contextDetails, ok := err.Context["details"].([]string)
	if !ok {
		t.Fatalf("Context[details] should be []string")
	}
	if len(contextDetails) != len(details) {
		t.Errorf("Context[details] length = %d, want %d", len(contextDetails), len(details))
	}
}

func TestNewCycleDetectedError(t *testing.T) {
	path := []string{"ar-1234", "ar-5678", "ar-1234"}
	err := NewCycleDetectedError(path)

	if err.Code != ErrCodeCycleDetected {
		t.Errorf("Code = %v, want %v", err.Code, ErrCodeCycleDetected)
	}
	contextPath, ok := err.Context["path"].([]string)
	if !ok {
		t.Fatalf("Context[path] should be []string")
	}
	if len(contextPath) != len(path) {
		t.Errorf("Context[path] length = %d, want %d", len(contextPath), len(path))
	}
}

func TestNewProjectNotFoundError(t *testing.T) {
	project := "my-project"
	err := NewProjectNotFoundError(project)

	if err.Code != ErrCodeProjectNotFound {
		t.Errorf("Code = %v, want %v", err.Code, ErrCodeProjectNotFound)
	}
	if !strings.Contains(err.Message, project) {
		t.Errorf("Message should contain project name, got: %v", err.Message)
	}
	if err.Context["project"] != project {
		t.Errorf("Context[project] = %v, want %v", err.Context["project"], project)
	}
}

func TestNewDependencyNotFoundError(t *testing.T) {
	childID := "ar-1234"
	parentID := "ar-5678"
	err := NewDependencyNotFoundError(childID, parentID)

	if err.Code != ErrCodeDependencyNotFound {
		t.Errorf("Code = %v, want %v", err.Code, ErrCodeDependencyNotFound)
	}
	if !strings.Contains(err.Message, childID) {
		t.Errorf("Message should contain child ID, got: %v", err.Message)
	}
	if !strings.Contains(err.Message, parentID) {
		t.Errorf("Message should contain parent ID, got: %v", err.Message)
	}
	if err.Context["child_id"] != childID {
		t.Errorf("Context[child_id] = %v, want %v", err.Context["child_id"], childID)
	}
	if err.Context["parent_id"] != parentID {
		t.Errorf("Context[parent_id] = %v, want %v", err.Context["parent_id"], parentID)
	}
}

func TestNewInternalError(t *testing.T) {
	err := NewInternalError(nil)

	if err.Code != ErrCodeInternalError {
		t.Errorf("Code = %v, want %v", err.Code, ErrCodeInternalError)
	}
	// Message should not expose internal details
	if strings.Contains(strings.ToLower(err.Message), "nil") {
		t.Error("Internal error message should not expose details")
	}
}

func TestErrorCodes_Unique(t *testing.T) {
	codes := []ErrorCode{
		ErrCodeTaskNotFound,
		ErrCodeAlreadyClaimed,
		ErrCodeNotOwner,
		ErrCodeInvalidTransition,
		ErrCodeValidationFailed,
		ErrCodeCycleDetected,
		ErrCodeInternalError,
		ErrCodeProjectNotFound,
		ErrCodeDependencyNotFound,
	}

	seen := make(map[ErrorCode]bool)
	for _, code := range codes {
		if seen[code] {
			t.Errorf("Duplicate error code: %v", code)
		}
		seen[code] = true
	}
}
