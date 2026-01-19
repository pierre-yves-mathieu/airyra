package domain

import "time"

// AuditAction represents the type of action recorded in an audit entry.
type AuditAction string

const (
	ActionCreate  AuditAction = "create"
	ActionUpdate  AuditAction = "update"
	ActionDelete  AuditAction = "delete"
	ActionClaim   AuditAction = "claim"
	ActionRelease AuditAction = "release"
)

// ValidAuditActions contains all valid audit action values.
var ValidAuditActions = []AuditAction{
	ActionCreate,
	ActionUpdate,
	ActionDelete,
	ActionClaim,
	ActionRelease,
}

// IsValid checks if the action is a valid audit action.
func (a AuditAction) IsValid() bool {
	for _, v := range ValidAuditActions {
		if a == v {
			return true
		}
	}
	return false
}

// AuditEntry represents a single change in the audit log.
type AuditEntry struct {
	ID        int64       `json:"id"`
	TaskID    string      `json:"task_id"`
	Action    AuditAction `json:"action"`
	Field     *string     `json:"field,omitempty"`
	OldValue  *string     `json:"old_value,omitempty"`
	NewValue  *string     `json:"new_value,omitempty"`
	ChangedAt time.Time   `json:"changed_at"`
	ChangedBy string      `json:"changed_by"`
}

// NewAuditEntry creates a new audit entry with the given parameters.
func NewAuditEntry(taskID string, action AuditAction, changedBy string) AuditEntry {
	return AuditEntry{
		TaskID:    taskID,
		Action:    action,
		ChangedAt: time.Now(),
		ChangedBy: changedBy,
	}
}

// WithField sets the field name for the audit entry.
func (e AuditEntry) WithField(field string) AuditEntry {
	e.Field = &field
	return e
}

// WithOldValue sets the old value for the audit entry.
func (e AuditEntry) WithOldValue(oldValue string) AuditEntry {
	e.OldValue = &oldValue
	return e
}

// WithNewValue sets the new value for the audit entry.
func (e AuditEntry) WithNewValue(newValue string) AuditEntry {
	e.NewValue = &newValue
	return e
}
