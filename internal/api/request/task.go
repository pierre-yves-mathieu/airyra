package request

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/airyra/airyra/internal/domain"
)

// CreateTaskRequest represents a request to create a task.
type CreateTaskRequest struct {
	Title       string  `json:"title"`
	Description *string `json:"description,omitempty"`
	Priority    *int    `json:"priority,omitempty"`
	ParentID    *string `json:"parent_id,omitempty"`
	SpecID      *string `json:"spec_id,omitempty"`
}

// Validate validates the create task request.
func (r *CreateTaskRequest) Validate() []string {
	var errors []string

	if r.Title == "" {
		errors = append(errors, "title is required")
	}

	if r.Priority != nil && !domain.ValidPriority(*r.Priority) {
		errors = append(errors, "priority must be between 0 and 4")
	}

	return errors
}

// UpdateTaskRequest represents a request to update a task.
type UpdateTaskRequest struct {
	Title       *string `json:"title,omitempty"`
	Description *string `json:"description,omitempty"`
	Priority    *int    `json:"priority,omitempty"`
	ParentID    *string `json:"parent_id,omitempty"`
}

// Validate validates the update task request.
func (r *UpdateTaskRequest) Validate() []string {
	var errors []string

	if r.Title != nil && *r.Title == "" {
		errors = append(errors, "title cannot be empty")
	}

	if r.Priority != nil && !domain.ValidPriority(*r.Priority) {
		errors = append(errors, "priority must be between 0 and 4")
	}

	return errors
}

// DecodeJSON decodes JSON from request body into the given value.
func DecodeJSON(r *http.Request, v interface{}) error {
	return json.NewDecoder(r.Body).Decode(v)
}

// Pagination contains pagination parameters.
type Pagination struct {
	Page    int
	PerPage int
}

// DefaultPage is the default page number.
const DefaultPage = 1

// DefaultPerPage is the default items per page.
const DefaultPerPage = 50

// MaxPerPage is the maximum items per page.
const MaxPerPage = 100

// ParsePagination extracts pagination from query parameters.
func ParsePagination(r *http.Request) Pagination {
	page := DefaultPage
	perPage := DefaultPerPage

	if p := r.URL.Query().Get("page"); p != "" {
		if v, err := strconv.Atoi(p); err == nil && v > 0 {
			page = v
		}
	}

	if pp := r.URL.Query().Get("per_page"); pp != "" {
		if v, err := strconv.Atoi(pp); err == nil && v > 0 {
			perPage = v
		}
	}

	if perPage > MaxPerPage {
		perPage = MaxPerPage
	}

	return Pagination{Page: page, PerPage: perPage}
}

// ParseStatus extracts status filter from query parameters.
func ParseStatus(r *http.Request) *domain.TaskStatus {
	s := r.URL.Query().Get("status")
	if s == "" {
		return nil
	}

	status := domain.TaskStatus(s)
	if !status.IsValid() {
		return nil
	}
	return &status
}
