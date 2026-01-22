package request

import (
	"net/http"

	"github.com/airyra/airyra/internal/domain"
)

// CreateSpecRequest represents a request to create a spec.
type CreateSpecRequest struct {
	Title       string  `json:"title"`
	Description *string `json:"description,omitempty"`
}

// Validate validates the create spec request.
func (r *CreateSpecRequest) Validate() []string {
	var errors []string

	if r.Title == "" {
		errors = append(errors, "title is required")
	}

	return errors
}

// UpdateSpecRequest represents a request to update a spec.
type UpdateSpecRequest struct {
	Title       *string `json:"title,omitempty"`
	Description *string `json:"description,omitempty"`
}

// Validate validates the update spec request.
func (r *UpdateSpecRequest) Validate() []string {
	var errors []string

	if r.Title != nil && *r.Title == "" {
		errors = append(errors, "title cannot be empty")
	}

	return errors
}

// AddSpecDependencyRequest represents a request to add a spec dependency.
type AddSpecDependencyRequest struct {
	ParentID string `json:"parent_id"`
}

// Validate validates the add spec dependency request.
func (r *AddSpecDependencyRequest) Validate() []string {
	var errors []string

	if r.ParentID == "" {
		errors = append(errors, "parent_id is required")
	}

	return errors
}

// ParseSpecStatus extracts spec status filter from query parameters.
func ParseSpecStatus(r *http.Request) *domain.SpecStatus {
	s := r.URL.Query().Get("status")
	if s == "" {
		return nil
	}

	status := domain.SpecStatus(s)
	if !status.IsValid() {
		return nil
	}
	return &status
}
