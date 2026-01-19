package main

import (
	"errors"
	"testing"

	"github.com/airyra/airyra/internal/client"
	"github.com/airyra/airyra/internal/domain"
)

func TestMapErrorToExitCode(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected int
	}{
		{
			name:     "nil error",
			err:      nil,
			expected: ExitSuccess,
		},
		{
			name:     "server not running",
			err:      client.ErrServerNotRunning,
			expected: ExitServerNotRunning,
		},
		{
			name:     "task not found",
			err:      domain.NewTaskNotFoundError("abc123"),
			expected: ExitTaskNotFound,
		},
		{
			name:     "already claimed",
			err:      domain.NewAlreadyClaimedError("user", "2024-01-01"),
			expected: ExitConflict,
		},
		{
			name:     "not owner",
			err:      domain.NewNotOwnerError("other-user"),
			expected: ExitPermissionDenied,
		},
		{
			name:     "project not found",
			err:      domain.NewProjectNotFoundError("myproject"),
			expected: ExitProjectNotConfigured,
		},
		{
			name:     "generic error",
			err:      errors.New("something went wrong"),
			expected: ExitGeneralError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := mapErrorToExitCode(tt.err)
			if result != tt.expected {
				t.Errorf("mapErrorToExitCode() = %d, expected %d", result, tt.expected)
			}
		})
	}
}

func TestMapErrorToExitCode_DomainErrors(t *testing.T) {
	tests := []struct {
		name     string
		errCode  domain.ErrorCode
		expected int
	}{
		{
			name:     "task not found code",
			errCode:  domain.ErrCodeTaskNotFound,
			expected: ExitTaskNotFound,
		},
		{
			name:     "already claimed code",
			errCode:  domain.ErrCodeAlreadyClaimed,
			expected: ExitConflict,
		},
		{
			name:     "not owner code",
			errCode:  domain.ErrCodeNotOwner,
			expected: ExitPermissionDenied,
		},
		{
			name:     "invalid transition code",
			errCode:  domain.ErrCodeInvalidTransition,
			expected: ExitGeneralError,
		},
		{
			name:     "cycle detected code",
			errCode:  domain.ErrCodeCycleDetected,
			expected: ExitGeneralError,
		},
		{
			name:     "validation failed code",
			errCode:  domain.ErrCodeValidationFailed,
			expected: ExitGeneralError,
		},
		{
			name:     "project not found code",
			errCode:  domain.ErrCodeProjectNotFound,
			expected: ExitProjectNotConfigured,
		},
		{
			name:     "dependency not found code",
			errCode:  domain.ErrCodeDependencyNotFound,
			expected: ExitTaskNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := &domain.DomainError{Code: tt.errCode, Message: "test"}
			result := mapErrorToExitCode(err)
			if result != tt.expected {
				t.Errorf("mapErrorToExitCode() = %d, expected %d", result, tt.expected)
			}
		})
	}
}

func TestParsePriority(t *testing.T) {
	tests := []struct {
		input    string
		expected int
		hasError bool
	}{
		{"0", 0, false},
		{"1", 1, false},
		{"2", 2, false},
		{"3", 3, false},
		{"4", 4, false},
		{"critical", 0, false},
		{"high", 1, false},
		{"normal", 2, false},
		{"low", 3, false},
		{"lowest", 4, false},
		{"CRITICAL", 0, false},
		{"HIGH", 1, false},
		{"NORMAL", 2, false},
		{"-1", 0, true},
		{"5", 0, true},
		{"invalid", 0, true},
		{"", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result, err := parsePriority(tt.input)
			if tt.hasError {
				if err == nil {
					t.Error("Expected error but got nil")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if result != tt.expected {
					t.Errorf("parsePriority(%s) = %d, expected %d", tt.input, result, tt.expected)
				}
			}
		})
	}
}

func TestIsConfigNotFoundError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "nil error",
			err:      nil,
			expected: false,
		},
		{
			name:     "config not found error",
			err:      errors.New("No airyra.toml found. Run 'ar init <name>' to create one."),
			expected: true,
		},
		{
			name:     "other error",
			err:      errors.New("something else"),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isConfigNotFoundError(tt.err)
			if result != tt.expected {
				t.Errorf("isConfigNotFoundError() = %v, expected %v", result, tt.expected)
			}
		})
	}
}
