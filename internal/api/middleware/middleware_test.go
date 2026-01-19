package middleware_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/airyra/airyra/internal/api/middleware"
	"github.com/airyra/airyra/internal/api/response"
)

func TestRecovery_PanicReturns500(t *testing.T) {
	// Handler that panics
	panicHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("something went wrong!")
	})

	// Wrap with recovery middleware
	handler := middleware.Recovery(panicHandler)

	req := httptest.NewRequest("GET", "/test", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Errorf("expected status 500, got %d", rr.Code)
	}

	var resp response.ErrorResponse
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp.Error.Code != "INTERNAL_ERROR" {
		t.Errorf("expected code 'INTERNAL_ERROR', got %q", resp.Error.Code)
	}
}

func TestAgentHeader_Extracted(t *testing.T) {
	var extractedAgent string

	// Handler that captures the agent ID
	captureHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		extractedAgent = middleware.GetAgentID(r.Context())
		w.WriteHeader(http.StatusOK)
	})

	// Wrap with agent middleware
	handler := middleware.AgentID(captureHandler)

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set(middleware.AgentHeader, "my-custom-agent")
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if extractedAgent != "my-custom-agent" {
		t.Errorf("expected agent 'my-custom-agent', got %q", extractedAgent)
	}
}

func TestAgentHeader_DefaultsToAnonymous(t *testing.T) {
	var extractedAgent string

	// Handler that captures the agent ID
	captureHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		extractedAgent = middleware.GetAgentID(r.Context())
		w.WriteHeader(http.StatusOK)
	})

	// Wrap with agent middleware
	handler := middleware.AgentID(captureHandler)

	req := httptest.NewRequest("GET", "/test", nil)
	// No agent header set
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if extractedAgent != middleware.DefaultAgentID {
		t.Errorf("expected default agent %q, got %q", middleware.DefaultAgentID, extractedAgent)
	}
}

func TestLogging_CapturesStatus(t *testing.T) {
	// Handler that returns 201
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
	})

	// Wrap with logging middleware
	wrapped := middleware.Logging(handler)

	req := httptest.NewRequest("GET", "/test", nil)
	rr := httptest.NewRecorder()

	wrapped.ServeHTTP(rr, req)

	// Just verify it doesn't panic and returns correct status
	if rr.Code != http.StatusCreated {
		t.Errorf("expected status 201, got %d", rr.Code)
	}
}
