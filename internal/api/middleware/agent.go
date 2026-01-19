package middleware

import (
	"context"
	"net/http"
)

type contextKey string

const (
	// AgentIDKey is the context key for the agent ID.
	AgentIDKey contextKey = "agentID"
	// AgentHeader is the HTTP header name for the agent ID.
	AgentHeader = "X-Airyra-Agent"
	// DefaultAgentID is used when no agent header is provided.
	DefaultAgentID = "anonymous"
)

// AgentID middleware extracts the X-Airyra-Agent header and adds it to context.
func AgentID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		agentID := r.Header.Get(AgentHeader)
		if agentID == "" {
			agentID = DefaultAgentID
		}

		ctx := context.WithValue(r.Context(), AgentIDKey, agentID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// GetAgentID retrieves the agent ID from context.
func GetAgentID(ctx context.Context) string {
	if agentID, ok := ctx.Value(AgentIDKey).(string); ok {
		return agentID
	}
	return DefaultAgentID
}
