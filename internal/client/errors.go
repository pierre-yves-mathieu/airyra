package client

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/airyra/airyra/internal/domain"
)

// Client-specific errors.
var (
	// ErrServerNotRunning indicates the server is not reachable.
	ErrServerNotRunning = errors.New("server is not running or unreachable")
	// ErrServerUnhealthy indicates the health check failed.
	ErrServerUnhealthy = errors.New("server health check failed")
)

// APIError represents an error response from the API.
type APIError struct {
	Code    string                 `json:"code"`
	Message string                 `json:"message"`
	Context map[string]interface{} `json:"context,omitempty"`
}

// apiErrorResponse wraps the error in the API response format.
type apiErrorResponse struct {
	Error APIError `json:"error"`
}

// parseErrorResponse parses an error response from the API and returns the
// appropriate domain error or a generic API error.
func parseErrorResponse(resp *http.Response) error {
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read error response: %w", err)
	}

	var apiErr apiErrorResponse
	if err := json.Unmarshal(body, &apiErr); err != nil {
		return fmt.Errorf("server error: %s", string(body))
	}

	return mapAPIErrorToDomain(resp.StatusCode, &apiErr.Error)
}

// mapAPIErrorToDomain maps an API error to the appropriate domain error.
func mapAPIErrorToDomain(statusCode int, apiErr *APIError) error {
	switch {
	case statusCode == http.StatusNotFound && apiErr.Code == string(domain.ErrCodeTaskNotFound):
		taskID, _ := apiErr.Context["id"].(string)
		return domain.NewTaskNotFoundError(taskID)

	case statusCode == http.StatusConflict && apiErr.Code == string(domain.ErrCodeAlreadyClaimed):
		claimedBy, _ := apiErr.Context["claimed_by"].(string)
		claimedAt, _ := apiErr.Context["claimed_at"].(string)
		return domain.NewAlreadyClaimedError(claimedBy, claimedAt)

	case statusCode == http.StatusForbidden && apiErr.Code == string(domain.ErrCodeNotOwner):
		claimedBy, _ := apiErr.Context["claimed_by"].(string)
		return domain.NewNotOwnerError(claimedBy)

	case statusCode == http.StatusBadRequest && apiErr.Code == string(domain.ErrCodeInvalidTransition):
		from, _ := apiErr.Context["from"].(string)
		to, _ := apiErr.Context["to"].(string)
		return domain.NewInvalidTransitionError(domain.TaskStatus(from), domain.TaskStatus(to))

	case statusCode == http.StatusBadRequest && apiErr.Code == string(domain.ErrCodeCycleDetected):
		path := extractStringSlice(apiErr.Context, "path")
		return domain.NewCycleDetectedError(path)

	case statusCode == http.StatusBadRequest && apiErr.Code == string(domain.ErrCodeValidationFailed):
		details := extractStringSlice(apiErr.Context, "details")
		return domain.NewValidationError(details)

	case statusCode == http.StatusNotFound && apiErr.Code == string(domain.ErrCodeProjectNotFound):
		project, _ := apiErr.Context["project"].(string)
		return domain.NewProjectNotFoundError(project)

	case statusCode == http.StatusNotFound && apiErr.Code == string(domain.ErrCodeDependencyNotFound):
		childID, _ := apiErr.Context["child_id"].(string)
		parentID, _ := apiErr.Context["parent_id"].(string)
		return domain.NewDependencyNotFoundError(childID, parentID)

	default:
		return &domain.DomainError{
			Code:    domain.ErrorCode(apiErr.Code),
			Message: apiErr.Message,
			Context: apiErr.Context,
		}
	}
}

// extractStringSlice extracts a string slice from a context map.
func extractStringSlice(ctx map[string]interface{}, key string) []string {
	val, ok := ctx[key]
	if !ok {
		return nil
	}

	// JSON unmarshals arrays as []interface{}
	slice, ok := val.([]interface{})
	if !ok {
		return nil
	}

	result := make([]string, 0, len(slice))
	for _, v := range slice {
		if s, ok := v.(string); ok {
			result = append(result, s)
		}
	}
	return result
}

// isConnectionRefused checks if the error is a connection refused error.
func isConnectionRefused(err error) bool {
	if err == nil {
		return false
	}
	// Check for common connection refused patterns in the error string
	errStr := err.Error()
	return contains(errStr, "connection refused") ||
		contains(errStr, "connect: connection refused") ||
		contains(errStr, "dial tcp") && contains(errStr, "refused")
}

// contains checks if s contains substr (case-insensitive).
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsImpl(s, substr))
}

func containsImpl(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
