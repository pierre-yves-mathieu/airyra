package airyra

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
)

// CreateTask creates a new task with the given title.
func (c *Client) CreateTask(ctx context.Context, title string, opts ...CreateTaskOption) (*Task, error) {
	options := &createTaskOptions{}
	for _, opt := range opts {
		opt(options)
	}

	body := createTaskRequest{
		Title:       title,
		Description: options.description,
		Priority:    options.priority,
		ParentID:    options.parentID,
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

	var task Task
	if err := json.NewDecoder(resp.Body).Decode(&task); err != nil {
		return nil, fmt.Errorf("failed to decode task response: %w", err)
	}

	return &task, nil
}

// GetTask retrieves a task by ID.
func (c *Client) GetTask(ctx context.Context, id string) (*Task, error) {
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

	var task Task
	if err := json.NewDecoder(resp.Body).Decode(&task); err != nil {
		return nil, fmt.Errorf("failed to decode task response: %w", err)
	}

	return &task, nil
}

// ListTasks lists tasks with optional filtering.
func (c *Client) ListTasks(ctx context.Context, opts ...ListTasksOption) (*TaskList, error) {
	options := defaultListTasksOptions()
	for _, opt := range opts {
		opt(options)
	}

	path := c.projectPath("/tasks")

	params := url.Values{}
	if options.status != "" {
		params.Set("status", options.status)
	}
	params.Set("page", strconv.Itoa(options.page))
	params.Set("per_page", strconv.Itoa(options.perPage))

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

	return &TaskList{
		Tasks:      paginatedResp.Data,
		Page:       paginatedResp.Pagination.Page,
		PerPage:    paginatedResp.Pagination.PerPage,
		Total:      paginatedResp.Pagination.Total,
		TotalPages: paginatedResp.Pagination.TotalPages,
	}, nil
}

// ListReadyTasks lists tasks that are ready to be worked on.
func (c *Client) ListReadyTasks(ctx context.Context, opts ...ListTasksOption) (*TaskList, error) {
	options := defaultListTasksOptions()
	for _, opt := range opts {
		opt(options)
	}

	path := c.projectPath("/tasks/ready")

	params := url.Values{}
	params.Set("page", strconv.Itoa(options.page))
	params.Set("per_page", strconv.Itoa(options.perPage))
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

	return &TaskList{
		Tasks:      paginatedResp.Data,
		Page:       paginatedResp.Pagination.Page,
		PerPage:    paginatedResp.Pagination.PerPage,
		Total:      paginatedResp.Pagination.Total,
		TotalPages: paginatedResp.Pagination.TotalPages,
	}, nil
}

// UpdateTask updates a task.
func (c *Client) UpdateTask(ctx context.Context, id string, opts ...UpdateTaskOption) (*Task, error) {
	options := &updateTaskOptions{}
	for _, opt := range opts {
		opt(options)
	}

	body := updateTaskRequest{
		Title:       options.title,
		Description: options.description,
		Priority:    options.priority,
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

	var task Task
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

// ClaimTask claims a task for the current agent.
func (c *Client) ClaimTask(ctx context.Context, id string) (*Task, error) {
	return c.doTransition(ctx, id, "claim")
}

// CompleteTask marks a task as complete.
func (c *Client) CompleteTask(ctx context.Context, id string) (*Task, error) {
	return c.doTransition(ctx, id, "done")
}

// ReleaseTask releases a claimed task.
func (c *Client) ReleaseTask(ctx context.Context, id string, force bool) (*Task, error) {
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

	var task Task
	if err := json.NewDecoder(resp.Body).Decode(&task); err != nil {
		return nil, fmt.Errorf("failed to decode task response: %w", err)
	}

	return &task, nil
}

// BlockTask marks a task as blocked.
func (c *Client) BlockTask(ctx context.Context, id string) (*Task, error) {
	return c.doTransition(ctx, id, "block")
}

// UnblockTask unblocks a blocked task.
func (c *Client) UnblockTask(ctx context.Context, id string) (*Task, error) {
	return c.doTransition(ctx, id, "unblock")
}

// doTransition performs a status transition on a task.
func (c *Client) doTransition(ctx context.Context, id, action string) (*Task, error) {
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

	var task Task
	if err := json.NewDecoder(resp.Body).Decode(&task); err != nil {
		return nil, fmt.Errorf("failed to decode task response: %w", err)
	}

	return &task, nil
}
