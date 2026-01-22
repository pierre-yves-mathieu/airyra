package airyra

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
)

// CreateSpec creates a new spec.
func (c *Client) CreateSpec(ctx context.Context, opts ...CreateSpecOption) (*Spec, error) {
	cfg := &createSpecConfig{}
	for _, opt := range opts {
		opt(cfg)
	}

	if cfg.title == "" {
		return nil, fmt.Errorf("title is required")
	}

	body := createSpecRequest{
		Title: cfg.title,
	}
	if cfg.description != "" {
		body.Description = &cfg.description
	}

	req, err := c.newJSONRequest(ctx, http.MethodPost, c.projectPath("/specs"), body)
	if err != nil {
		return nil, err
	}

	resp, err := c.http.Do(req)
	if err != nil {
		if isConnectionRefused(err) {
			return nil, ErrServerNotRunning
		}
		return nil, fmt.Errorf("create spec failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		return nil, parseErrorResponse(resp)
	}

	var spec Spec
	if err := json.NewDecoder(resp.Body).Decode(&spec); err != nil {
		return nil, fmt.Errorf("failed to decode spec response: %w", err)
	}

	return &spec, nil
}

// GetSpec retrieves a spec by ID.
func (c *Client) GetSpec(ctx context.Context, id string) (*Spec, error) {
	req, err := c.newRequest(ctx, http.MethodGet, c.projectPath("/specs/"+id), nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.http.Do(req)
	if err != nil {
		if isConnectionRefused(err) {
			return nil, ErrServerNotRunning
		}
		return nil, fmt.Errorf("get spec failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, parseErrorResponse(resp)
	}

	var spec Spec
	if err := json.NewDecoder(resp.Body).Decode(&spec); err != nil {
		return nil, fmt.Errorf("failed to decode spec response: %w", err)
	}

	return &spec, nil
}

// ListSpecs lists specs with optional filtering.
func (c *Client) ListSpecs(ctx context.Context, opts ...ListSpecsOption) (*SpecList, error) {
	cfg := &listSpecsConfig{
		page:    1,
		perPage: 50,
	}
	for _, opt := range opts {
		opt(cfg)
	}

	path := c.projectPath("/specs")
	params := url.Values{}
	if cfg.status != "" {
		params.Set("status", cfg.status)
	}
	params.Set("page", strconv.Itoa(cfg.page))
	params.Set("per_page", strconv.Itoa(cfg.perPage))
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
		return nil, fmt.Errorf("list specs failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, parseErrorResponse(resp)
	}

	var paginatedResp paginatedSpecResponse
	if err := json.NewDecoder(resp.Body).Decode(&paginatedResp); err != nil {
		return nil, fmt.Errorf("failed to decode specs response: %w", err)
	}

	return &SpecList{
		Specs:      paginatedResp.Data,
		Page:       paginatedResp.Pagination.Page,
		PerPage:    paginatedResp.Pagination.PerPage,
		Total:      paginatedResp.Pagination.Total,
		TotalPages: paginatedResp.Pagination.TotalPages,
	}, nil
}

// ListReadySpecs lists specs that are ready to be worked on.
func (c *Client) ListReadySpecs(ctx context.Context, opts ...ListSpecsOption) (*SpecList, error) {
	cfg := &listSpecsConfig{
		page:    1,
		perPage: 50,
	}
	for _, opt := range opts {
		opt(cfg)
	}

	path := c.projectPath("/specs/ready")
	params := url.Values{}
	params.Set("page", strconv.Itoa(cfg.page))
	params.Set("per_page", strconv.Itoa(cfg.perPage))
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
		return nil, fmt.Errorf("list ready specs failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, parseErrorResponse(resp)
	}

	var paginatedResp paginatedSpecResponse
	if err := json.NewDecoder(resp.Body).Decode(&paginatedResp); err != nil {
		return nil, fmt.Errorf("failed to decode specs response: %w", err)
	}

	return &SpecList{
		Specs:      paginatedResp.Data,
		Page:       paginatedResp.Pagination.Page,
		PerPage:    paginatedResp.Pagination.PerPage,
		Total:      paginatedResp.Pagination.Total,
		TotalPages: paginatedResp.Pagination.TotalPages,
	}, nil
}

// UpdateSpec updates a spec.
func (c *Client) UpdateSpec(ctx context.Context, id string, opts ...UpdateSpecOption) (*Spec, error) {
	cfg := &updateSpecConfig{}
	for _, opt := range opts {
		opt(cfg)
	}

	body := updateSpecRequest{
		Title:       cfg.title,
		Description: cfg.description,
	}

	req, err := c.newJSONRequest(ctx, http.MethodPatch, c.projectPath("/specs/"+id), body)
	if err != nil {
		return nil, err
	}

	resp, err := c.http.Do(req)
	if err != nil {
		if isConnectionRefused(err) {
			return nil, ErrServerNotRunning
		}
		return nil, fmt.Errorf("update spec failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, parseErrorResponse(resp)
	}

	var spec Spec
	if err := json.NewDecoder(resp.Body).Decode(&spec); err != nil {
		return nil, fmt.Errorf("failed to decode spec response: %w", err)
	}

	return &spec, nil
}

// DeleteSpec deletes a spec.
func (c *Client) DeleteSpec(ctx context.Context, id string) error {
	req, err := c.newRequest(ctx, http.MethodDelete, c.projectPath("/specs/"+id), nil)
	if err != nil {
		return err
	}

	resp, err := c.http.Do(req)
	if err != nil {
		if isConnectionRefused(err) {
			return ErrServerNotRunning
		}
		return fmt.Errorf("delete spec failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		return parseErrorResponse(resp)
	}

	return nil
}

// CancelSpec cancels a spec.
func (c *Client) CancelSpec(ctx context.Context, id string) (*Spec, error) {
	req, err := c.newRequest(ctx, http.MethodPost, c.projectPath("/specs/"+id+"/cancel"), nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.http.Do(req)
	if err != nil {
		if isConnectionRefused(err) {
			return nil, ErrServerNotRunning
		}
		return nil, fmt.Errorf("cancel spec failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, parseErrorResponse(resp)
	}

	var spec Spec
	if err := json.NewDecoder(resp.Body).Decode(&spec); err != nil {
		return nil, fmt.Errorf("failed to decode spec response: %w", err)
	}

	return &spec, nil
}

// ReopenSpec reopens a cancelled spec.
func (c *Client) ReopenSpec(ctx context.Context, id string) (*Spec, error) {
	req, err := c.newRequest(ctx, http.MethodPost, c.projectPath("/specs/"+id+"/reopen"), nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.http.Do(req)
	if err != nil {
		if isConnectionRefused(err) {
			return nil, ErrServerNotRunning
		}
		return nil, fmt.Errorf("reopen spec failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, parseErrorResponse(resp)
	}

	var spec Spec
	if err := json.NewDecoder(resp.Body).Decode(&spec); err != nil {
		return nil, fmt.Errorf("failed to decode spec response: %w", err)
	}

	return &spec, nil
}

// ListSpecTasks lists tasks belonging to a spec.
func (c *Client) ListSpecTasks(ctx context.Context, specID string, opts ...ListTasksOption) (*TaskList, error) {
	cfg := &listTasksOptions{
		page:    1,
		perPage: 50,
	}
	for _, opt := range opts {
		opt(cfg)
	}

	path := c.projectPath("/specs/" + specID + "/tasks")
	params := url.Values{}
	params.Set("page", strconv.Itoa(cfg.page))
	params.Set("per_page", strconv.Itoa(cfg.perPage))
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
		return nil, fmt.Errorf("list spec tasks failed: %w", err)
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

// AddSpecDependency adds a dependency between specs.
func (c *Client) AddSpecDependency(ctx context.Context, childID, parentID string) error {
	body := addSpecDependencyRequest{
		ParentID: parentID,
	}

	req, err := c.newJSONRequest(ctx, http.MethodPost, c.projectPath("/specs/"+childID+"/deps"), body)
	if err != nil {
		return err
	}

	resp, err := c.http.Do(req)
	if err != nil {
		if isConnectionRefused(err) {
			return ErrServerNotRunning
		}
		return fmt.Errorf("add spec dependency failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		return parseErrorResponse(resp)
	}

	return nil
}

// RemoveSpecDependency removes a dependency between specs.
func (c *Client) RemoveSpecDependency(ctx context.Context, childID, parentID string) error {
	req, err := c.newRequest(ctx, http.MethodDelete, c.projectPath("/specs/"+childID+"/deps/"+parentID), nil)
	if err != nil {
		return err
	}

	resp, err := c.http.Do(req)
	if err != nil {
		if isConnectionRefused(err) {
			return ErrServerNotRunning
		}
		return fmt.Errorf("remove spec dependency failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		return parseErrorResponse(resp)
	}

	return nil
}

// ListSpecDependencies lists dependencies for a spec.
func (c *Client) ListSpecDependencies(ctx context.Context, specID string) ([]*SpecDependency, error) {
	req, err := c.newRequest(ctx, http.MethodGet, c.projectPath("/specs/"+specID+"/deps"), nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.http.Do(req)
	if err != nil {
		if isConnectionRefused(err) {
			return nil, ErrServerNotRunning
		}
		return nil, fmt.Errorf("list spec dependencies failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, parseErrorResponse(resp)
	}

	var deps []*SpecDependency
	if err := json.NewDecoder(resp.Body).Decode(&deps); err != nil {
		return nil, fmt.Errorf("failed to decode spec dependencies response: %w", err)
	}

	return deps, nil
}

// CreateSpec options
type CreateSpecOption func(*createSpecConfig)
type createSpecConfig struct {
	title       string
	description string
}

// WithSpecTitle sets the spec title.
func WithSpecTitle(title string) CreateSpecOption {
	return func(cfg *createSpecConfig) {
		cfg.title = title
	}
}

// WithSpecDescription sets the spec description.
func WithSpecDescription(desc string) CreateSpecOption {
	return func(cfg *createSpecConfig) {
		cfg.description = desc
	}
}

// ListSpecs options
type ListSpecsOption func(*listSpecsConfig)
type listSpecsConfig struct {
	status  string
	page    int
	perPage int
}

// WithSpecStatus filters by spec status.
func WithSpecStatus(status SpecStatus) ListSpecsOption {
	return func(cfg *listSpecsConfig) {
		cfg.status = string(status)
	}
}

// WithSpecPage sets the page number for listing.
func WithSpecPage(page int) ListSpecsOption {
	return func(cfg *listSpecsConfig) {
		cfg.page = page
	}
}

// WithSpecPerPage sets the items per page for listing.
func WithSpecPerPage(perPage int) ListSpecsOption {
	return func(cfg *listSpecsConfig) {
		cfg.perPage = perPage
	}
}

// UpdateSpec options
type UpdateSpecOption func(*updateSpecConfig)
type updateSpecConfig struct {
	title       *string
	description *string
}

// WithUpdatedSpecTitle updates the spec title.
func WithUpdatedSpecTitle(title string) UpdateSpecOption {
	return func(cfg *updateSpecConfig) {
		cfg.title = &title
	}
}

// WithUpdatedSpecDescription updates the spec description.
func WithUpdatedSpecDescription(desc string) UpdateSpecOption {
	return func(cfg *updateSpecConfig) {
		cfg.description = &desc
	}
}
