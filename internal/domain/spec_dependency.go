package domain

// SpecDependency represents a dependency relationship between specs.
// The child spec depends on the parent spec (child is blocked until parent is done).
type SpecDependency struct {
	ChildID  string `json:"child_id"`
	ParentID string `json:"parent_id"`
}

// NewSpecDependency creates a new spec dependency relationship.
func NewSpecDependency(childID, parentID string) SpecDependency {
	return SpecDependency{
		ChildID:  childID,
		ParentID: parentID,
	}
}
