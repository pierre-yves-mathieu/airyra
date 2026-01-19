package client

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/airyra/airyra/internal/domain"
)

// Client is an HTTP client for the Airyra server API.
type Client struct {
	baseURL string       // http://host:port
	agentID string       // X-Airyra-Agent header value
	project string       // Project name for URL paths
	http    *http.Client // HTTP client
}

// NewClient creates a new Airyra API client.
func NewClient(host string, port int, project string, agentID string) *Client {
	return &Client{
		baseURL: fmt.Sprintf("http://%s:%d", host, port),
		agentID: agentID,
		project: project,
		http: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// =============================================================================
// Health
// =============================================================================

// Health checks if the server is healthy.
func (c *Client) Health(ctx context.Context) error {
	req, err := c.newRequest(ctx, http.MethodGet, "/v1/health", nil)
	if err != nil {
		return err
	}

	resp, err := c.http.Do(req)
	if err != nil {
		if isConnectionRefused(err) {
			return ErrServerNotRunning
		}
		return fmt.Errorf("health check failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return ErrServerUnhealthy
	}

	return nil
}

// ListProjects returns a list of all project names.
func (c *Client) ListProjects(ctx context.Context) ([]string, error) {
	req, err := c.newRequest(ctx, http.MethodGet, "/v1/projects", nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.http.Do(req)
	if err != nil {
		if isConnectionRefused(err) {
			return nil, ErrServerNotRunning
		}
		return nil, fmt.Errorf("list projects failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, parseErrorResponse(resp)
	}

	var projects []string
	if err := json.NewDecoder(resp.Body).Decode(&projects); err != nil {
		return nil, fmt.Errorf("failed to decode projects response: %w", err)
	}

	return projects, nil
}

// =============================================================================
// Task CRUD
// =============================================================================

// CreateTask creates a new task.
func (c *Client) CreateTask(ctx context.Context, title, description string, priority int, parentID string) (*domain.Task, error) {
	body := createTaskRequest{
		Title: title,
	}
	if description != "" {
		body.Description = &description
	}
	body.Priority = &priority
	if parentID != "" {
		body.ParentID = &parentID
	}

	req, err := c.newJSONRequest(ctx, http.MethodPost, c.projectPath("/tasks"), body)
	if err != nil {
		return nil, err
	}

	resp, err := c.http.Do(req)
	if err != nil {
		if isConnectionRefused(err) {
			return nil, ErrServerNotRunning
		}
		return nil, fmt.Errorf("create task failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		return nil, parseErrorResponse(resp)
	}

	var task domain.Task
	if err := json.NewDecoder(resp.Body).Decode(&task); err != nil {
		return nil, fmt.Errorf("failed to decode task response: %w", err)
	}

	return &task, nil
}

// GetTask retrieves a task by ID.
func (c *Client) GetTask(ctx context.Context, id string) (*domain.Task, error) {
	req, err := c.newRequest(ctx, http.MethodGet, c.projectPath("/tasks/"+id), nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.http.Do(req)
	if err != nil {
		if isConnectionRefused(err) {
			return nil, ErrServerNotRunning
		}
		return nil, fmt.Errorf("get task failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, parseErrorResponse(resp)
	}

	var task domain.Task
	if err := json.NewDecoder(resp.Body).Decode(&task); err != nil {
		return nil, fmt.Errorf("failed to decode task response: %w", err)
	}

	return &task, nil
}

// ListTasks lists tasks with optional filtering.
func (c *Client) ListTasks(ctx context.Context, status string, page, perPage int) (*TaskListResponse, error) {
	path := c.projectPath("/tasks")

	// Build query parameters
	params := url.Values{}
	if status != "" {
		params.Set("status", status)
	}
	params.Set("page", strconv.Itoa(page))
	params.Set("per_page", strconv.Itoa(perPage))

	if len(params) > 0 {
		path = path + "?" + params.Encode()
	}

	req, err := c.newRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.http.Do(req)
	if err != nil {
		if isConnectionRefused(err) {
			return nil, ErrServerNotRunning
		}
		return nil, fmt.Errorf("list tasks failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, parseErrorResponse(resp)
	}

	var paginatedResp paginatedTaskResponse
	if err := json.NewDecoder(resp.Body).Decode(&paginatedResp); err != nil {
		return nil, fmt.Errorf("failed to decode tasks response: %w", err)
	}

	return &TaskListResponse{
		Data: paginatedResp.Data,
		Pagination: &Pagination{
			Page:       paginatedResp.Pagination.Page,
			PerPage:    paginatedResp.Pagination.PerPage,
			Total:      paginatedResp.Pagination.Total,
			TotalPages: paginatedResp.Pagination.TotalPages,
		},
	}, nil
}

// ListReadyTasks lists tasks that are ready to be worked on.
func (c *Client) ListReadyTasks(ctx context.Context, page, perPage int) (*TaskListResponse, error) {
	path := c.projectPath("/tasks/ready")

	params := url.Values{}
	params.Set("page", strconv.Itoa(page))
	params.Set("per_page", strconv.Itoa(perPage))
	path = path + "?" + params.Encode()

	req, err := c.newRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.http.Do(req)
	if err != nil {
		if isConnectionRefused(err) {
			return nil, ErrServerNotRunning
		}
		return nil, fmt.Errorf("list ready tasks failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, parseErrorResponse(resp)
	}

	var paginatedResp paginatedTaskResponse
	if err := json.NewDecoder(resp.Body).Decode(&paginatedResp); err != nil {
		return nil, fmt.Errorf("failed to decode tasks response: %w", err)
	}

	return &TaskListResponse{
		Data: paginatedResp.Data,
		Pagination: &Pagination{
			Page:       paginatedResp.Pagination.Page,
			PerPage:    paginatedResp.Pagination.PerPage,
			Total:      paginatedResp.Pagination.Total,
			TotalPages: paginatedResp.Pagination.TotalPages,
		},
	}, nil
}

// UpdateTask updates a task.
func (c *Client) UpdateTask(ctx context.Context, id string, updates TaskUpdates) (*domain.Task, error) {
	body := updateTaskRequest{
		Title:       updates.Title,
		Description: updates.Description,
		Priority:    updates.Priority,
	}

	req, err := c.newJSONRequest(ctx, http.MethodPatch, c.projectPath("/tasks/"+id), body)
	if err != nil {
		return nil, err
	}

	resp, err := c.http.Do(req)
	if err != nil {
		if isConnectionRefused(err) {
			return nil, ErrServerNotRunning
		}
		return nil, fmt.Errorf("update task failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, parseErrorResponse(resp)
	}

	var task domain.Task
	if err := json.NewDecoder(resp.Body).Decode(&task); err != nil {
		return nil, fmt.Errorf("failed to decode task response: %w", err)
	}

	return &task, nil
}

// DeleteTask deletes a task.
func (c *Client) DeleteTask(ctx context.Context, id string) error {
	req, err := c.newRequest(ctx, http.MethodDelete, c.projectPath("/tasks/"+id), nil)
	if err != nil {
		return err
	}

	resp, err := c.http.Do(req)
	if err != nil {
		if isConnectionRefused(err) {
			return ErrServerNotRunning
		}
		return fmt.Errorf("delete task failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		return parseErrorResponse(resp)
	}

	return nil
}

// =============================================================================
// Status Transitions
// =============================================================================

// ClaimTask claims a task for the current agent.
func (c *Client) ClaimTask(ctx context.Context, id string) (*domain.Task, error) {
	return c.doTransition(ctx, id, "claim")
}

// CompleteTask marks a task as complete.
func (c *Client) CompleteTask(ctx context.Context, id string) (*domain.Task, error) {
	return c.doTransition(ctx, id, "done")
}

// ReleaseTask releases a claimed task.
func (c *Client) ReleaseTask(ctx context.Context, id string, force bool) (*domain.Task, error) {
	path := c.projectPath("/tasks/" + id + "/release")
	if force {
		path = path + "?force=true"
	}

	req, err := c.newRequest(ctx, http.MethodPost, path, nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.http.Do(req)
	if err != nil {
		if isConnectionRefused(err) {
			return nil, ErrServerNotRunning
		}
		return nil, fmt.Errorf("release task failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, parseErrorResponse(resp)
	}

	var task domain.Task
	if err := json.NewDecoder(resp.Body).Decode(&task); err != nil {
		return nil, fmt.Errorf("failed to decode task response: %w", err)
	}

	return &task, nil
}

// BlockTask marks a task as blocked.
func (c *Client) BlockTask(ctx context.Context, id string) (*domain.Task, error) {
	return c.doTransition(ctx, id, "block")
}

// UnblockTask unblocks a blocked task.
func (c *Client) UnblockTask(ctx context.Context, id string) (*domain.Task, error) {
	return c.doTransition(ctx, id, "unblock")
}

// doTransition performs a status transition on a task.
func (c *Client) doTransition(ctx context.Context, id, action string) (*domain.Task, error) {
	req, err := c.newRequest(ctx, http.MethodPost, c.projectPath("/tasks/"+id+"/"+action), nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.http.Do(req)
	if err != nil {
		if isConnectionRefused(err) {
			return nil, ErrServerNotRunning
		}
		return nil, fmt.Errorf("%s task failed: %w", action, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, parseErrorResponse(resp)
	}

	var task domain.Task
	if err := json.NewDecoder(resp.Body).Decode(&task); err != nil {
		return nil, fmt.Errorf("failed to decode task response: %w", err)
	}

	return &task, nil
}

// =============================================================================
// Dependencies
// =============================================================================

// AddDependency adds a dependency between two tasks.
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
func (c *Client) ListDependencies(ctx context.Context, taskID string) ([]domain.Dependency, error) {
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

	var deps []domain.Dependency
	if err := json.NewDecoder(resp.Body).Decode(&deps); err != nil {
		return nil, fmt.Errorf("failed to decode dependencies response: %w", err)
	}

	return deps, nil
}

// =============================================================================
// Audit
// =============================================================================

// GetTaskHistory retrieves the audit history for a task.
func (c *Client) GetTaskHistory(ctx context.Context, taskID string) ([]domain.AuditEntry, error) {
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

	var entries []domain.AuditEntry
	if err := json.NewDecoder(resp.Body).Decode(&entries); err != nil {
		return nil, fmt.Errorf("failed to decode audit entries response: %w", err)
	}

	return entries, nil
}

// =============================================================================
// Helper Methods
// =============================================================================

// projectPath constructs a URL path with the project prefix.
func (c *Client) projectPath(path string) string {
	return "/v1/projects/" + c.project + path
}

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

// Ensure Client implements expected interface at compile time.
var _ interface {
	Health(ctx context.Context) error
	ListProjects(ctx context.Context) ([]string, error)
	CreateTask(ctx context.Context, title, description string, priority int, parentID string) (*domain.Task, error)
	GetTask(ctx context.Context, id string) (*domain.Task, error)
	ListTasks(ctx context.Context, status string, page, perPage int) (*TaskListResponse, error)
	ListReadyTasks(ctx context.Context, page, perPage int) (*TaskListResponse, error)
	UpdateTask(ctx context.Context, id string, updates TaskUpdates) (*domain.Task, error)
	DeleteTask(ctx context.Context, id string) error
	ClaimTask(ctx context.Context, id string) (*domain.Task, error)
	CompleteTask(ctx context.Context, id string) (*domain.Task, error)
	ReleaseTask(ctx context.Context, id string, force bool) (*domain.Task, error)
	BlockTask(ctx context.Context, id string) (*domain.Task, error)
	UnblockTask(ctx context.Context, id string) (*domain.Task, error)
	AddDependency(ctx context.Context, childID, parentID string) error
	RemoveDependency(ctx context.Context, childID, parentID string) error
	ListDependencies(ctx context.Context, taskID string) ([]domain.Dependency, error)
	GetTaskHistory(ctx context.Context, taskID string) ([]domain.AuditEntry, error)
} = (*Client)(nil)

// wrapConnectionError wraps connection errors with ErrServerNotRunning.
func wrapConnectionError(err error) error {
	if err == nil {
		return nil
	}
	if isConnectionRefused(err) {
		return errors.Join(ErrServerNotRunning, err)
	}
	return err
}
