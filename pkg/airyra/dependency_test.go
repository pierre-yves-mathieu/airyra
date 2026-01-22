package airyra

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestAddDependency(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/projects/test-project/tasks/child-task/deps" {
			t.Errorf("expected path /v1/projects/test-project/tasks/child-task/deps, got %s", r.URL.Path)
		}
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("expected Content-Type application/json, got %s", r.Header.Get("Content-Type"))
		}

		var req addDependencyRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Errorf("failed to decode request body: %v", err)
		}
		if req.ParentID != "parent-task" {
			t.Errorf("expected parent_id 'parent-task', got %s", req.ParentID)
		}

		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(Dependency{
			ChildID:  "child-task",
			ParentID: "parent-task",
		})
	}))
	defer server.Close()

	client := newTestClient(t, server)
	err := client.AddDependency(context.Background(), "child-task", "parent-task")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestAddDependencyCycleDetected(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(apiErrorResponse{
			Error: apiError{
				Code:    string(ErrCodeCycleDetected),
				Message: "Adding this dependency would create a cycle",
				Context: map[string]interface{}{
					"path": []interface{}{"task-1", "task-2", "task-3"},
				},
			},
		})
	}))
	defer server.Close()

	client := newTestClient(t, server)
	err := client.AddDependency(context.Background(), "child-task", "parent-task")
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if !IsCycleDetected(err) {
		t.Errorf("expected IsCycleDetected to be true, got false")
	}
}

func TestRemoveDependency(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/projects/test-project/tasks/child-task/deps/parent-task" {
			t.Errorf("expected path /v1/projects/test-project/tasks/child-task/deps/parent-task, got %s", r.URL.Path)
		}
		if r.Method != http.MethodDelete {
			t.Errorf("expected DELETE, got %s", r.Method)
		}

		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	client := newTestClient(t, server)
	err := client.RemoveDependency(context.Background(), "child-task", "parent-task")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRemoveDependencyNotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(apiErrorResponse{
			Error: apiError{
				Code:    string(ErrCodeDependencyNotFound),
				Message: "Dependency not found",
				Context: map[string]interface{}{
					"child_id":  "child-task",
					"parent_id": "parent-task",
				},
			},
		})
	}))
	defer server.Close()

	client := newTestClient(t, server)
	err := client.RemoveDependency(context.Background(), "child-task", "parent-task")
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if !IsDependencyNotFound(err) {
		t.Errorf("expected IsDependencyNotFound to be true, got false")
	}
}

func TestListDependencies(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/projects/test-project/tasks/task-123/deps" {
			t.Errorf("expected path /v1/projects/test-project/tasks/task-123/deps, got %s", r.URL.Path)
		}
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]Dependency{
			{ChildID: "task-123", ParentID: "parent-1"},
			{ChildID: "task-123", ParentID: "parent-2"},
		})
	}))
	defer server.Close()

	client := newTestClient(t, server)
	deps, err := client.ListDependencies(context.Background(), "task-123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(deps) != 2 {
		t.Errorf("expected 2 dependencies, got %d", len(deps))
	}
	if deps[0].ParentID != "parent-1" {
		t.Errorf("expected first dependency parent_id parent-1, got %s", deps[0].ParentID)
	}
}
