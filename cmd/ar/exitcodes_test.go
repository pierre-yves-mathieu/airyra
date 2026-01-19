package main

import "testing"

func TestExitCodes_Values(t *testing.T) {
	// Test that exit codes have expected values
	tests := []struct {
		name     string
		code     int
		expected int
	}{
		{"ExitSuccess", ExitSuccess, 0},
		{"ExitGeneralError", ExitGeneralError, 1},
		{"ExitServerNotRunning", ExitServerNotRunning, 2},
		{"ExitProjectNotConfigured", ExitProjectNotConfigured, 3},
		{"ExitTaskNotFound", ExitTaskNotFound, 4},
		{"ExitPermissionDenied", ExitPermissionDenied, 5},
		{"ExitConflict", ExitConflict, 6},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.code != tt.expected {
				t.Errorf("%s = %d, expected %d", tt.name, tt.code, tt.expected)
			}
		})
	}
}

func TestExitCodes_Unique(t *testing.T) {
	// Test that all exit codes are unique
	codes := []int{
		ExitSuccess,
		ExitGeneralError,
		ExitServerNotRunning,
		ExitProjectNotConfigured,
		ExitTaskNotFound,
		ExitPermissionDenied,
		ExitConflict,
	}

	seen := make(map[int]bool)
	for _, code := range codes {
		if seen[code] {
			t.Errorf("Duplicate exit code: %d", code)
		}
		seen[code] = true
	}
}
