package domain

import (
	"testing"
	"time"
)

func TestAuditAction_IsValid(t *testing.T) {
	tests := []struct {
		name   string
		action AuditAction
		want   bool
	}{
		{"ActionCreate is valid", ActionCreate, true},
		{"ActionUpdate is valid", ActionUpdate, true},
		{"ActionDelete is valid", ActionDelete, true},
		{"ActionClaim is valid", ActionClaim, true},
		{"ActionRelease is valid", ActionRelease, true},
		{"empty string is invalid", AuditAction(""), false},
		{"random string is invalid", AuditAction("random"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.action.IsValid(); got != tt.want {
				t.Errorf("AuditAction.IsValid() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestValidAuditActions_ContainsAllActions(t *testing.T) {
	expected := []AuditAction{ActionCreate, ActionUpdate, ActionDelete, ActionClaim, ActionRelease}
	if len(ValidAuditActions) != len(expected) {
		t.Errorf("ValidAuditActions has %d items, want %d", len(ValidAuditActions), len(expected))
	}
	for _, a := range expected {
		found := false
		for _, v := range ValidAuditActions {
			if v == a {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("ValidAuditActions does not contain %s", a)
		}
	}
}

func TestNewAuditEntry(t *testing.T) {
	taskID := "ar-1234"
	action := ActionCreate
	changedBy := "agent-1"

	entry := NewAuditEntry(taskID, action, changedBy)

	if entry.TaskID != taskID {
		t.Errorf("NewAuditEntry() TaskID = %v, want %v", entry.TaskID, taskID)
	}
	if entry.Action != action {
		t.Errorf("NewAuditEntry() Action = %v, want %v", entry.Action, action)
	}
	if entry.ChangedBy != changedBy {
		t.Errorf("NewAuditEntry() ChangedBy = %v, want %v", entry.ChangedBy, changedBy)
	}

	// ChangedAt should be recent
	now := time.Now()
	if now.Sub(entry.ChangedAt) > time.Second {
		t.Error("NewAuditEntry() ChangedAt should be recent")
	}

	// Optional fields should be nil
	if entry.Field != nil {
		t.Error("NewAuditEntry() Field should be nil")
	}
	if entry.OldValue != nil {
		t.Error("NewAuditEntry() OldValue should be nil")
	}
	if entry.NewValue != nil {
		t.Error("NewAuditEntry() NewValue should be nil")
	}
}

func TestAuditEntry_WithField(t *testing.T) {
	entry := NewAuditEntry("ar-1234", ActionUpdate, "agent-1")
	field := "status"

	updated := entry.WithField(field)

	if updated.Field == nil {
		t.Fatal("WithField() should set Field")
	}
	if *updated.Field != field {
		t.Errorf("WithField() Field = %v, want %v", *updated.Field, field)
	}
}

func TestAuditEntry_WithOldValue(t *testing.T) {
	entry := NewAuditEntry("ar-1234", ActionUpdate, "agent-1")
	oldValue := "open"

	updated := entry.WithOldValue(oldValue)

	if updated.OldValue == nil {
		t.Fatal("WithOldValue() should set OldValue")
	}
	if *updated.OldValue != oldValue {
		t.Errorf("WithOldValue() OldValue = %v, want %v", *updated.OldValue, oldValue)
	}
}

func TestAuditEntry_WithNewValue(t *testing.T) {
	entry := NewAuditEntry("ar-1234", ActionUpdate, "agent-1")
	newValue := "done"

	updated := entry.WithNewValue(newValue)

	if updated.NewValue == nil {
		t.Fatal("WithNewValue() should set NewValue")
	}
	if *updated.NewValue != newValue {
		t.Errorf("WithNewValue() NewValue = %v, want %v", *updated.NewValue, newValue)
	}
}

func TestAuditEntry_Chaining(t *testing.T) {
	entry := NewAuditEntry("ar-1234", ActionUpdate, "agent-1").
		WithField("status").
		WithOldValue("open").
		WithNewValue("done")

	if entry.Field == nil || *entry.Field != "status" {
		t.Error("Chaining should work for Field")
	}
	if entry.OldValue == nil || *entry.OldValue != "open" {
		t.Error("Chaining should work for OldValue")
	}
	if entry.NewValue == nil || *entry.NewValue != "done" {
		t.Error("Chaining should work for NewValue")
	}
}
