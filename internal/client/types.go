package client

import "github.com/airyra/airyra/internal/domain"

// TaskListResponse represents a paginated list of tasks.
type TaskListResponse struct {
	Data       []*domain.Task
	Pagination *Pagination
}

// Pagination contains pagination metadata from API responses.
type Pagination struct {
	Page       int
	PerPage    int
	Total      int
	TotalPages int
}

// TaskUpdates contains optional fields for updating a task.
type TaskUpdates struct {
	Title       *string
	Description *string
	Priority    *int
}

// paginatedTaskResponse is the raw JSON structure for paginated task responses.
type paginatedTaskResponse struct {
	Data       []*domain.Task     `json:"data"`
	Pagination paginationResponse `json:"pagination"`
}

// paginationResponse is the raw JSON structure for pagination metadata.
type paginationResponse struct {
	Page       int `json:"page"`
	PerPage    int `json:"per_page"`
	Total      int `json:"total"`
	TotalPages int `json:"total_pages"`
}

// createTaskRequest is the JSON request body for creating a task.
type createTaskRequest struct {
	Title       string  `json:"title"`
	Description *string `json:"description,omitempty"`
	Priority    *int    `json:"priority,omitempty"`
	ParentID    *string `json:"parent_id,omitempty"`
	SpecID      *string `json:"spec_id,omitempty"`
}

// Spec represents an epic-like entity for grouping related tasks.
type Spec struct {
	ID          string  `json:"id"`
	Title       string  `json:"title"`
	Description *string `json:"description,omitempty"`
	Status      string  `json:"status"`
	TaskCount   int     `json:"task_count"`
	DoneCount   int     `json:"done_count"`
	CreatedAt   string  `json:"created_at"`
	UpdatedAt   string  `json:"updated_at"`
}

// SpecListResponse represents a paginated list of specs.
type SpecListResponse struct {
	Data       []*Spec
	Pagination *Pagination
}

// SpecUpdates contains optional fields for updating a spec.
type SpecUpdates struct {
	Title       *string
	Description *string
}

// SpecDependency represents a dependency relationship between specs.
type SpecDependency struct {
	ChildID  string `json:"child_id"`
	ParentID string `json:"parent_id"`
}

// paginatedSpecResponse is the raw JSON structure for paginated spec responses.
type paginatedSpecResponse struct {
	Data       []*Spec            `json:"data"`
	Pagination paginationResponse `json:"pagination"`
}

// createSpecRequest is the JSON request body for creating a spec.
type createSpecRequest struct {
	Title       string  `json:"title"`
	Description *string `json:"description,omitempty"`
}

// updateSpecRequest is the JSON request body for updating a spec.
type updateSpecRequest struct {
	Title       *string `json:"title,omitempty"`
	Description *string `json:"description,omitempty"`
}

// addSpecDependencyRequest is the JSON request body for adding a spec dependency.
type addSpecDependencyRequest struct {
	ParentID string `json:"parent_id"`
}

// updateTaskRequest is the JSON request body for updating a task.
type updateTaskRequest struct {
	Title       *string `json:"title,omitempty"`
	Description *string `json:"description,omitempty"`
	Priority    *int    `json:"priority,omitempty"`
}

// addDependencyRequest is the JSON request body for adding a dependency.
type addDependencyRequest struct {
	ParentID string `json:"parent_id"`
}

// healthResponse is the JSON response for the health endpoint.
type healthResponse struct {
	Status string `json:"status"`
}
