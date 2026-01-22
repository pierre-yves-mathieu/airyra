package main

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/airyra/airyra/internal/client"
	"github.com/airyra/airyra/internal/domain"
)

func TestCreateCmd_Exists(t *testing.T) {
	if createCmd == nil {
		t.Error("createCmd should not be nil")
	}
}

func TestCreateCmd_Use(t *testing.T) {
	if createCmd.Use != "create <title>" {
		t.Errorf("createCmd.Use = %s, expected 'create <title>'", createCmd.Use)
	}
}

func TestCreateCmd_HasPriorityFlag(t *testing.T) {
	flag := createCmd.Flags().Lookup("priority")
	if flag == nil {
		t.Error("createCmd should have --priority flag")
	}
}

func TestCreateCmd_HasPriorityShortFlag(t *testing.T) {
	flag := createCmd.Flags().ShorthandLookup("p")
	if flag == nil {
		t.Error("createCmd should have -p flag")
	}
}

func TestCreateCmd_HasDescriptionFlag(t *testing.T) {
	flag := createCmd.Flags().Lookup("description")
	if flag == nil {
		t.Error("createCmd should have --description flag")
	}
}

func TestCreateCmd_HasDescriptionShortFlag(t *testing.T) {
	flag := createCmd.Flags().ShorthandLookup("d")
	if flag == nil {
		t.Error("createCmd should have -d flag")
	}
}

func TestCreateCmd_HasParentFlag(t *testing.T) {
	flag := createCmd.Flags().Lookup("parent")
	if flag == nil {
		t.Error("createCmd should have --parent flag")
	}
}

func TestListCmd_Exists(t *testing.T) {
	if listCmd == nil {
		t.Error("listCmd should not be nil")
	}
}

func TestListCmd_HasStatusFlag(t *testing.T) {
	flag := listCmd.Flags().Lookup("status")
	if flag == nil {
		t.Error("listCmd should have --status flag")
	}
}

func TestListCmd_HasPageFlag(t *testing.T) {
	flag := listCmd.Flags().Lookup("page")
	if flag == nil {
		t.Error("listCmd should have --page flag")
	}
}

func TestListCmd_HasPerPageFlag(t *testing.T) {
	flag := listCmd.Flags().Lookup("per-page")
	if flag == nil {
		t.Error("listCmd should have --per-page flag")
	}
}

func TestShowCmd_Exists(t *testing.T) {
	if showCmd == nil {
		t.Error("showCmd should not be nil")
	}
}

func TestShowCmd_Use(t *testing.T) {
	if showCmd.Use != "show <id>" {
		t.Errorf("showCmd.Use = %s, expected 'show <id>'", showCmd.Use)
	}
}

func TestEditCmd_Exists(t *testing.T) {
	if editCmd == nil {
		t.Error("editCmd should not be nil")
	}
}

func TestEditCmd_HasTitleFlag(t *testing.T) {
	flag := editCmd.Flags().Lookup("title")
	if flag == nil {
		t.Error("editCmd should have --title flag")
	}
}

func TestDeleteCmd_Exists(t *testing.T) {
	if deleteCmd == nil {
		t.Error("deleteCmd should not be nil")
	}
}

// Mock server for integration tests

func newMockServer(t *testing.T, handler http.HandlerFunc) *httptest.Server {
	return httptest.NewServer(handler)
}

func TestCreate_Success(t *testing.T) {
	// Create mock server
	server := newMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v1/projects/testproject/tasks" && r.Method == "POST" {
			task := domain.Task{
				ID:        "abc123",
				Title:     "Test task",
				Status:    domain.StatusOpen,
				Priority:  2,
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			}
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(task)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	})
	defer server.Close()

	// Parse server URL to get host and port
	host, port := parseURL(server.URL)

	// Create client
	c := client.NewClient(host, port, "testproject", "test@host:/path")

	// Run create
	var buf bytes.Buffer
	task, err := c.CreateTask(context.Background(), "Test task", "", 2, "", "")
	if err != nil {
		t.Fatalf("CreateTask failed: %v", err)
	}

	printTask(&buf, task, false)
	output := buf.String()

	if !strings.Contains(output, "abc123") {
		t.Error("Output should contain task ID")
	}
	if !strings.Contains(output, "Test task") {
		t.Error("Output should contain task title")
	}
}

func TestCreate_MissingTitle(t *testing.T) {
	err := validateCreateArgs([]string{})
	if err == nil {
		t.Error("validateCreateArgs should fail with no arguments")
	}
}

func TestList_Success(t *testing.T) {
	// Create mock server
	server := newMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v1/projects/testproject/tasks" && r.Method == "GET" {
			resp := map[string]interface{}{
				"data": []domain.Task{
					{
						ID:        "abc123",
						Title:     "Task 1",
						Status:    domain.StatusOpen,
						Priority:  1,
						CreatedAt: time.Now(),
						UpdatedAt: time.Now(),
					},
					{
						ID:        "def456",
						Title:     "Task 2",
						Status:    domain.StatusInProgress,
						Priority:  2,
						CreatedAt: time.Now(),
						UpdatedAt: time.Now(),
					},
				},
				"pagination": map[string]interface{}{
					"page":        1,
					"per_page":    50,
					"total":       2,
					"total_pages": 1,
				},
			}
			json.NewEncoder(w).Encode(resp)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	})
	defer server.Close()

	// Parse server URL to get host and port
	host, port := parseURL(server.URL)

	// Create client
	c := client.NewClient(host, port, "testproject", "test@host:/path")

	// Run list
	result, err := c.ListTasks(context.Background(), "", 1, 50)
	if err != nil {
		t.Fatalf("ListTasks failed: %v", err)
	}

	if len(result.Data) != 2 {
		t.Errorf("Expected 2 tasks, got %d", len(result.Data))
	}
}

func TestShow_Success(t *testing.T) {
	// Create mock server
	server := newMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v1/projects/testproject/tasks/abc123" && r.Method == "GET" {
			task := domain.Task{
				ID:        "abc123",
				Title:     "Test task",
				Status:    domain.StatusOpen,
				Priority:  2,
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			}
			json.NewEncoder(w).Encode(task)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	})
	defer server.Close()

	// Parse server URL to get host and port
	host, port := parseURL(server.URL)

	// Create client
	c := client.NewClient(host, port, "testproject", "test@host:/path")

	// Run show
	task, err := c.GetTask(context.Background(), "abc123")
	if err != nil {
		t.Fatalf("GetTask failed: %v", err)
	}

	if task.ID != "abc123" {
		t.Errorf("Expected task ID abc123, got %s", task.ID)
	}
}

func TestShow_NotFound(t *testing.T) {
	// Create mock server
	server := newMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error": map[string]interface{}{
				"code":    "TASK_NOT_FOUND",
				"message": "Task notfound not found",
				"context": map[string]interface{}{"id": "notfound"},
			},
		})
	})
	defer server.Close()

	// Parse server URL to get host and port
	host, port := parseURL(server.URL)

	// Create client
	c := client.NewClient(host, port, "testproject", "test@host:/path")

	// Run show
	_, err := c.GetTask(context.Background(), "notfound")
	if err == nil {
		t.Error("GetTask should fail for non-existent task")
	}

	// Check it's a domain error
	domainErr, ok := err.(*domain.DomainError)
	if !ok {
		t.Errorf("Expected DomainError, got %T", err)
		return
	}

	if domainErr.Code != domain.ErrCodeTaskNotFound {
		t.Errorf("Expected TASK_NOT_FOUND error code, got %s", domainErr.Code)
	}
}

// Helper function to parse URL into host and port
func parseURL(urlStr string) (string, int) {
	// Remove http:// prefix
	urlStr = strings.TrimPrefix(urlStr, "http://")
	parts := strings.Split(urlStr, ":")
	host := parts[0]
	port := 80
	if len(parts) > 1 {
		var err error
		_, err = strings.NewReader(parts[1]).Read(make([]byte, 10))
		if err == nil {
			var p int
			for _, c := range parts[1] {
				if c >= '0' && c <= '9' {
					p = p*10 + int(c-'0')
				} else {
					break
				}
			}
			port = p
		}
	}
	return host, port
}

func TestValidateCreateArgs(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		wantErr bool
	}{
		{"no args", []string{}, true},
		{"one arg", []string{"title"}, false},
		{"empty title", []string{""}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateCreateArgs(tt.args)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateCreateArgs() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
