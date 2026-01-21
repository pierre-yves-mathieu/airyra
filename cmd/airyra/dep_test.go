package main

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/airyra/airyra/internal/client"
	"github.com/airyra/airyra/internal/domain"
)

func TestDepCmd_Exists(t *testing.T) {
	if depCmd == nil {
		t.Error("depCmd should not be nil")
	}
}

func TestDepCmd_Use(t *testing.T) {
	if depCmd.Use != "dep" {
		t.Errorf("depCmd.Use = %s, expected 'dep'", depCmd.Use)
	}
}

func TestDepAddCmd_Exists(t *testing.T) {
	if depAddCmd == nil {
		t.Error("depAddCmd should not be nil")
	}
}

func TestDepAddCmd_Use(t *testing.T) {
	if depAddCmd.Use != "add <child> <parent>" {
		t.Errorf("depAddCmd.Use = %s, expected 'add <child> <parent>'", depAddCmd.Use)
	}
}

func TestDepRmCmd_Exists(t *testing.T) {
	if depRmCmd == nil {
		t.Error("depRmCmd should not be nil")
	}
}

func TestDepRmCmd_Use(t *testing.T) {
	if depRmCmd.Use != "rm <child> <parent>" {
		t.Errorf("depRmCmd.Use = %s, expected 'rm <child> <parent>'", depRmCmd.Use)
	}
}

func TestDepListCmd_Exists(t *testing.T) {
	if depListCmd == nil {
		t.Error("depListCmd should not be nil")
	}
}

func TestDepListCmd_Use(t *testing.T) {
	if depListCmd.Use != "list <id>" {
		t.Errorf("depListCmd.Use = %s, expected 'list <id>'", depListCmd.Use)
	}
}

func TestDepAdd_Success(t *testing.T) {
	server := newMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v1/projects/testproject/tasks/child123/deps" && r.Method == "POST" {
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"child_id":  "child123",
				"parent_id": "parent456",
			})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	})
	defer server.Close()

	host, port := parseURL(server.URL)
	c := client.NewClient(host, port, "testproject", "test@host:/path")

	err := c.AddDependency(context.Background(), "child123", "parent456")
	if err != nil {
		t.Fatalf("AddDependency failed: %v", err)
	}
}

func TestDepAdd_CycleDetected(t *testing.T) {
	server := newMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error": map[string]interface{}{
				"code":    "CYCLE_DETECTED",
				"message": "Adding this dependency would create a cycle",
				"context": map[string]interface{}{
					"path": []string{"abc", "def", "abc"},
				},
			},
		})
	})
	defer server.Close()

	host, port := parseURL(server.URL)
	c := client.NewClient(host, port, "testproject", "test@host:/path")

	err := c.AddDependency(context.Background(), "child123", "parent456")
	if err == nil {
		t.Error("AddDependency should fail with cycle detected")
	}

	domainErr, ok := err.(*domain.DomainError)
	if !ok {
		t.Errorf("Expected DomainError, got %T", err)
		return
	}

	if domainErr.Code != domain.ErrCodeCycleDetected {
		t.Errorf("Expected CYCLE_DETECTED error code, got %s", domainErr.Code)
	}
}

func TestDepRm_Success(t *testing.T) {
	server := newMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v1/projects/testproject/tasks/child123/deps/parent456" && r.Method == "DELETE" {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	})
	defer server.Close()

	host, port := parseURL(server.URL)
	c := client.NewClient(host, port, "testproject", "test@host:/path")

	err := c.RemoveDependency(context.Background(), "child123", "parent456")
	if err != nil {
		t.Fatalf("RemoveDependency failed: %v", err)
	}
}

func TestDepRm_NotFound(t *testing.T) {
	server := newMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error": map[string]interface{}{
				"code":    "DEPENDENCY_NOT_FOUND",
				"message": "Dependency from child123 to parent456 not found",
				"context": map[string]interface{}{
					"child_id":  "child123",
					"parent_id": "parent456",
				},
			},
		})
	})
	defer server.Close()

	host, port := parseURL(server.URL)
	c := client.NewClient(host, port, "testproject", "test@host:/path")

	err := c.RemoveDependency(context.Background(), "child123", "parent456")
	if err == nil {
		t.Error("RemoveDependency should fail for non-existent dependency")
	}
}

func TestDepList_Success(t *testing.T) {
	server := newMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v1/projects/testproject/tasks/abc123/deps" && r.Method == "GET" {
			deps := []domain.Dependency{
				{ChildID: "abc123", ParentID: "parent1"},
				{ChildID: "abc123", ParentID: "parent2"},
			}
			json.NewEncoder(w).Encode(deps)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	})
	defer server.Close()

	host, port := parseURL(server.URL)
	c := client.NewClient(host, port, "testproject", "test@host:/path")

	deps, err := c.ListDependencies(context.Background(), "abc123")
	if err != nil {
		t.Fatalf("ListDependencies failed: %v", err)
	}

	if len(deps) != 2 {
		t.Errorf("Expected 2 dependencies, got %d", len(deps))
	}
}

func TestDepList_Empty(t *testing.T) {
	server := newMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v1/projects/testproject/tasks/abc123/deps" && r.Method == "GET" {
			json.NewEncoder(w).Encode([]domain.Dependency{})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	})
	defer server.Close()

	host, port := parseURL(server.URL)
	c := client.NewClient(host, port, "testproject", "test@host:/path")

	deps, err := c.ListDependencies(context.Background(), "abc123")
	if err != nil {
		t.Fatalf("ListDependencies failed: %v", err)
	}

	if len(deps) != 0 {
		t.Errorf("Expected 0 dependencies, got %d", len(deps))
	}
}
