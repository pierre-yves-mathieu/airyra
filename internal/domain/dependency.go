package domain

// Dependency represents a dependency relationship between tasks.
// The child task depends on the parent task (child is blocked until parent is done).
type Dependency struct {
	ChildID  string `json:"child_id"`
	ParentID string `json:"parent_id"`
}

// NewDependency creates a new dependency relationship.
func NewDependency(childID, parentID string) Dependency {
	return Dependency{
		ChildID:  childID,
		ParentID: parentID,
	}
}
