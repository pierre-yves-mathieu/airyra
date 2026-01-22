package airyra

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestNewClient(t *testing.T) {
	tests := []struct {
		name    string
		opts    []ClientOption
		wantErr string
	}{
		{
			name:    "missing project",
			opts:    []ClientOption{WithAgentID("agent-1")},
			wantErr: "project is required",
		},
		{
			name:    "missing agent ID",
			opts:    []ClientOption{WithProject("test-project")},
			wantErr: "agent ID is required",
		},
		{
			name: "valid options",
			opts: []ClientOption{
				WithProject("test-project"),
				WithAgentID("agent-1"),
			},
			wantErr: "",
		},
		{
			name: "all options",
			opts: []ClientOption{
				WithProject("test-project"),
				WithAgentID("agent-1"),
				WithHost("example.com"),
				WithPort(8080),
			},
			wantErr: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := NewClient(tt.opts...)
			if tt.wantErr != "" {
				if err == nil {
					t.Errorf("expected error containing %q, got nil", tt.wantErr)
					return
				}
				if !strings.Contains(err.Error(), tt.wantErr) {
					t.Errorf("expected error containing %q, got %q", tt.wantErr, err.Error())
				}
				return
			}
			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}
			if client == nil {
				t.Error("expected non-nil client")
			}
		})
	}
}

func TestHealth(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		wantErr    error
	}{
		{
			name:       "healthy server",
			statusCode: http.StatusOK,
			wantErr:    nil,
		},
		{
			name:       "unhealthy server",
			statusCode: http.StatusServiceUnavailable,
			wantErr:    ErrServerUnhealthy,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/v1/health" {
					t.Errorf("expected path /v1/health, got %s", r.URL.Path)
				}
				if r.Method != http.MethodGet {
					t.Errorf("expected GET, got %s", r.Method)
				}
				w.WriteHeader(tt.statusCode)
			}))
			defer server.Close()

			client := newTestClient(t, server)
			err := client.Health(context.Background())

			if tt.wantErr != nil {
				if err != tt.wantErr {
					t.Errorf("expected error %v, got %v", tt.wantErr, err)
				}
				return
			}
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestListProjects(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/projects" {
			t.Errorf("expected path /v1/projects, got %s", r.URL.Path)
		}
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.Header.Get("X-Airyra-Agent") != "test-agent" {
			t.Errorf("expected X-Airyra-Agent header, got %s", r.Header.Get("X-Airyra-Agent"))
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]string{"project-1", "project-2"})
	}))
	defer server.Close()

	client := newTestClient(t, server)
	projects, err := client.ListProjects(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(projects) != 2 {
		t.Errorf("expected 2 projects, got %d", len(projects))
	}
	if projects[0] != "project-1" {
		t.Errorf("expected first project to be project-1, got %s", projects[0])
	}
}

// newTestClient creates a test client connected to the given test server.
func newTestClient(t *testing.T, server *httptest.Server) *Client {
	t.Helper()

	// Parse the server URL to extract host and port
	addr := server.Listener.Addr().String()
	parts := strings.Split(addr, ":")
	if len(parts) != 2 {
		t.Fatalf("failed to parse server address: %s", addr)
	}

	port := 0
	_, err := strings.NewReader(parts[1]).Read(make([]byte, 10))
	if err != nil {
		// Fallback: just use the server URL directly
	}

	// Create a client that points to the test server
	client := &Client{
		baseURL: server.URL,
		agentID: "test-agent",
		project: "test-project",
		http:    server.Client(),
	}

	_ = port // unused but kept for clarity

	return client
}
