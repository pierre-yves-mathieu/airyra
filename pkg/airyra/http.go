package airyra

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// newRequest creates a new HTTP request with common headers.
func (c *Client) newRequest(ctx context.Context, method, path string, body io.Reader) (*http.Request, error) {
	reqURL := c.baseURL + path

	req, err := http.NewRequestWithContext(ctx, method, reqURL, body)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("X-Airyra-Agent", c.agentID)

	return req, nil
}

// newJSONRequest creates a new HTTP request with JSON body.
func (c *Client) newJSONRequest(ctx context.Context, method, path string, body interface{}) (*http.Request, error) {
	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(body); err != nil {
		return nil, fmt.Errorf("failed to encode request body: %w", err)
	}

	req, err := c.newRequest(ctx, method, path, &buf)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")

	return req, nil
}

// projectPath constructs a URL path with the project prefix.
func (c *Client) projectPath(path string) string {
	return "/v1/projects/" + c.project + path
}

// parseErrorResponse parses an error response from the API and returns the
// appropriate error type.
func parseErrorResponse(resp *http.Response) error {
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read error response: %w", err)
	}

	var apiErr apiErrorResponse
	if err := json.Unmarshal(body, &apiErr); err != nil {
		return fmt.Errorf("server error: %s", string(body))
	}

	return mapAPIErrorToSDK(resp.StatusCode, &apiErr.Error)
}

// mapAPIErrorToSDK maps an API error to the appropriate SDK error.
func mapAPIErrorToSDK(statusCode int, apiErr *apiError) error {
	switch {
	case statusCode == http.StatusNotFound && apiErr.Code == string(ErrCodeTaskNotFound):
		taskID, _ := apiErr.Context["id"].(string)
		return newTaskNotFoundError(taskID)

	case statusCode == http.StatusConflict && apiErr.Code == string(ErrCodeAlreadyClaimed):
		claimedBy, _ := apiErr.Context["claimed_by"].(string)
		claimedAt, _ := apiErr.Context["claimed_at"].(string)
		return newAlreadyClaimedError(claimedBy, claimedAt)

	case statusCode == http.StatusForbidden && apiErr.Code == string(ErrCodeNotOwner):
		claimedBy, _ := apiErr.Context["claimed_by"].(string)
		return newNotOwnerError(claimedBy)

	case statusCode == http.StatusBadRequest && apiErr.Code == string(ErrCodeInvalidTransition):
		from, _ := apiErr.Context["from"].(string)
		to, _ := apiErr.Context["to"].(string)
		return newInvalidTransitionError(from, to)

	case statusCode == http.StatusBadRequest && apiErr.Code == string(ErrCodeCycleDetected):
		path := extractStringSlice(apiErr.Context, "path")
		return newCycleDetectedError(path)

	case statusCode == http.StatusBadRequest && apiErr.Code == string(ErrCodeValidationFailed):
		details := extractStringSlice(apiErr.Context, "details")
		return newValidationError(details)

	case statusCode == http.StatusNotFound && apiErr.Code == string(ErrCodeProjectNotFound):
		project, _ := apiErr.Context["project"].(string)
		return newProjectNotFoundError(project)

	case statusCode == http.StatusNotFound && apiErr.Code == string(ErrCodeDependencyNotFound):
		childID, _ := apiErr.Context["child_id"].(string)
		parentID, _ := apiErr.Context["parent_id"].(string)
		return newDependencyNotFoundError(childID, parentID)

	default:
		return &Error{
			Code:    ErrorCode(apiErr.Code),
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
	errStr := err.Error()
	return strings.Contains(errStr, "connection refused") ||
		(strings.Contains(errStr, "dial tcp") && strings.Contains(errStr, "refused"))
}
