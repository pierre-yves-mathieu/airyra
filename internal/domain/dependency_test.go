package domain

import "testing"

func TestNewDependency(t *testing.T) {
	childID := "ar-1234"
	parentID := "ar-5678"

	dep := NewDependency(childID, parentID)

	if dep.ChildID != childID {
		t.Errorf("NewDependency() ChildID = %v, want %v", dep.ChildID, childID)
	}
	if dep.ParentID != parentID {
		t.Errorf("NewDependency() ParentID = %v, want %v", dep.ParentID, parentID)
	}
}

func TestDependency_Fields(t *testing.T) {
	dep := Dependency{
		ChildID:  "ar-aaaa",
		ParentID: "ar-bbbb",
	}

	if dep.ChildID != "ar-aaaa" {
		t.Errorf("Dependency.ChildID = %v, want %v", dep.ChildID, "ar-aaaa")
	}
	if dep.ParentID != "ar-bbbb" {
		t.Errorf("Dependency.ParentID = %v, want %v", dep.ParentID, "ar-bbbb")
	}
}
