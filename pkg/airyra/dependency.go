package airyra

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// AddDependency adds a dependency between two tasks.
// The child task will depend on the parent task (child is blocked until parent is done).
func (c *Client) AddDependency(ctx context.Context, childID, parentID string) error {
	body := addDependencyRequest{
		ParentID: parentID,
	}

	req, err := c.newJSONRequest(ctx, http.MethodPost, c.projectPath("/tasks/"+childID+"/deps"), body)
	if err != nil {
		return err
	}

	resp, err := c.http.Do(req)
	if err != nil {
		if isConnectionRefused(err) {
			return ErrServerNotRunning
		}
		return fmt.Errorf("add dependency failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		return parseErrorResponse(resp)
	}

	// Drain and discard response body
	io.Copy(io.Discard, resp.Body)

	return nil
}

// RemoveDependency removes a dependency between two tasks.
func (c *Client) RemoveDependency(ctx context.Context, childID, parentID string) error {
	req, err := c.newRequest(ctx, http.MethodDelete, c.projectPath("/tasks/"+childID+"/deps/"+parentID), nil)
	if err != nil {
		return err
	}

	resp, err := c.http.Do(req)
	if err != nil {
		if isConnectionRefused(err) {
			return ErrServerNotRunning
		}
		return fmt.Errorf("remove dependency failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		return parseErrorResponse(resp)
	}

	return nil
}

// ListDependencies lists dependencies for a task.
func (c *Client) ListDependencies(ctx context.Context, taskID string) ([]Dependency, error) {
	req, err := c.newRequest(ctx, http.MethodGet, c.projectPath("/tasks/"+taskID+"/deps"), nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.http.Do(req)
	if err != nil {
		if isConnectionRefused(err) {
			return nil, ErrServerNotRunning
		}
		return nil, fmt.Errorf("list dependencies failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, parseErrorResponse(resp)
	}

	var deps []Dependency
	if err := json.NewDecoder(resp.Body).Decode(&deps); err != nil {
		return nil, fmt.Errorf("failed to decode dependencies response: %w", err)
	}

	return deps, nil
}
