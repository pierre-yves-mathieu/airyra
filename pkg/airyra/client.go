package airyra

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

// Client is an HTTP client for the Airyra server API.
type Client struct {
	baseURL string
	agentID string
	project string
	http    *http.Client
}

// NewClient creates a new Airyra API client.
//
// Required options:
//   - WithProject: sets the project name
//   - WithAgentID: sets the agent ID for task ownership
//
// Optional options:
//   - WithHost: sets the server host (default: localhost)
//   - WithPort: sets the server port (default: 7432)
//   - WithTimeout: sets the HTTP client timeout (default: 30s)
//
// Example:
//
//	client, err := airyra.NewClient(
//	    airyra.WithProject("my-project"),
//	    airyra.WithAgentID("agent-001"),
//	)
func NewClient(opts ...ClientOption) (*Client, error) {
	cfg := defaultConfig()
	for _, opt := range opts {
		opt(cfg)
	}

	if cfg.project == "" {
		return nil, fmt.Errorf("project is required: use WithProject option")
	}
	if cfg.agentID == "" {
		return nil, fmt.Errorf("agent ID is required: use WithAgentID option")
	}

	return &Client{
		baseURL: fmt.Sprintf("http://%s:%d", cfg.host, cfg.port),
		agentID: cfg.agentID,
		project: cfg.project,
		http: &http.Client{
			Timeout: cfg.timeout,
		},
	}, nil
}

// Health checks if the server is healthy.
func (c *Client) Health(ctx context.Context) error {
	req, err := c.newRequest(ctx, http.MethodGet, "/v1/health", nil)
	if err != nil {
		return err
	}

	resp, err := c.http.Do(req)
	if err != nil {
		if isConnectionRefused(err) {
			return ErrServerNotRunning
		}
		return fmt.Errorf("health check failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return ErrServerUnhealthy
	}

	return nil
}

// ListProjects returns a list of all project names.
func (c *Client) ListProjects(ctx context.Context) ([]string, error) {
	req, err := c.newRequest(ctx, http.MethodGet, "/v1/projects", nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.http.Do(req)
	if err != nil {
		if isConnectionRefused(err) {
			return nil, ErrServerNotRunning
		}
		return nil, fmt.Errorf("list projects failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, parseErrorResponse(resp)
	}

	var projects []string
	if err := json.NewDecoder(resp.Body).Decode(&projects); err != nil {
		return nil, fmt.Errorf("failed to decode projects response: %w", err)
	}

	return projects, nil
}
