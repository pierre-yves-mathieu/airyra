package request

// AddDependencyRequest represents a request to add a dependency.
type AddDependencyRequest struct {
	ParentID string `json:"parent_id"`
}

// Validate validates the add dependency request.
func (r *AddDependencyRequest) Validate() []string {
	var errors []string

	if r.ParentID == "" {
		errors = append(errors, "parent_id is required")
	}

	return errors
}
