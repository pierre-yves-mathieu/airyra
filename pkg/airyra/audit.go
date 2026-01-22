package airyra

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

// GetTaskHistory retrieves the audit history for a task.
func (c *Client) GetTaskHistory(ctx context.Context, taskID string) ([]AuditEntry, error) {
	req, err := c.newRequest(ctx, http.MethodGet, c.projectPath("/tasks/"+taskID+"/history"), nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.http.Do(req)
	if err != nil {
		if isConnectionRefused(err) {
			return nil, ErrServerNotRunning
		}
		return nil, fmt.Errorf("get task history failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, parseErrorResponse(resp)
	}

	var entries []AuditEntry
	if err := json.NewDecoder(resp.Body).Decode(&entries); err != nil {
		return nil, fmt.Errorf("failed to decode audit entries response: %w", err)
	}

	return entries, nil
}
