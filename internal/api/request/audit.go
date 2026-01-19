package request

import (
	"net/http"
	"time"
)

// AuditQueryParams contains query parameters for audit log queries.
type AuditQueryParams struct {
	Action    *string
	AgentID   *string
	StartTime *time.Time
	EndTime   *time.Time
}

// ParseAuditQuery extracts audit query parameters from the request.
func ParseAuditQuery(r *http.Request) AuditQueryParams {
	params := AuditQueryParams{}

	if action := r.URL.Query().Get("action"); action != "" {
		params.Action = &action
	}

	if agentID := r.URL.Query().Get("agent"); agentID != "" {
		params.AgentID = &agentID
	}

	if startStr := r.URL.Query().Get("start"); startStr != "" {
		if t, err := time.Parse(time.RFC3339, startStr); err == nil {
			params.StartTime = &t
		}
	}

	if endStr := r.URL.Query().Get("end"); endStr != "" {
		if t, err := time.Parse(time.RFC3339, endStr); err == nil {
			params.EndTime = &t
		}
	}

	return params
}
