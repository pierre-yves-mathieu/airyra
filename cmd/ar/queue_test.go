package main

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/airyra/airyra/internal/client"
	"github.com/airyra/airyra/internal/domain"
)

func TestReadyCmd_Exists(t *testing.T) {
	if readyCmd == nil {
		t.Error("readyCmd should not be nil")
	}
}

func TestReadyCmd_Use(t *testing.T) {
	if readyCmd.Use != "ready" {
		t.Errorf("readyCmd.Use = %s, expected 'ready'", readyCmd.Use)
	}
}

func TestNextCmd_Exists(t *testing.T) {
	if nextCmd == nil {
		t.Error("nextCmd should not be nil")
	}
}

func TestNextCmd_Use(t *testing.T) {
	if nextCmd.Use != "next" {
		t.Errorf("nextCmd.Use = %s, expected 'next'", nextCmd.Use)
	}
}

func TestReady_Success(t *testing.T) {
	server := newMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v1/projects/testproject/tasks/ready" && r.Method == "GET" {
			resp := map[string]interface{}{
				"data": []domain.Task{
					{
						ID:        "abc123",
						Title:     "Task 1",
						Status:    domain.StatusOpen,
						Priority:  0,
						CreatedAt: time.Now(),
						UpdatedAt: time.Now(),
					},
					{
						ID:        "def456",
						Title:     "Task 2",
						Status:    domain.StatusOpen,
						Priority:  1,
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

	host, port := parseURL(server.URL)
	c := client.NewClient(host, port, "testproject", "test@host:/path")

	result, err := c.ListReadyTasks(context.Background(), 1, 50)
	if err != nil {
		t.Fatalf("ListReadyTasks failed: %v", err)
	}

	if len(result.Data) != 2 {
		t.Errorf("Expected 2 tasks, got %d", len(result.Data))
	}
}

func TestReady_Empty(t *testing.T) {
	server := newMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v1/projects/testproject/tasks/ready" && r.Method == "GET" {
			resp := map[string]interface{}{
				"data": []domain.Task{},
				"pagination": map[string]interface{}{
					"page":        1,
					"per_page":    50,
					"total":       0,
					"total_pages": 0,
				},
			}
			json.NewEncoder(w).Encode(resp)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	})
	defer server.Close()

	host, port := parseURL(server.URL)
	c := client.NewClient(host, port, "testproject", "test@host:/path")

	result, err := c.ListReadyTasks(context.Background(), 1, 50)
	if err != nil {
		t.Fatalf("ListReadyTasks failed: %v", err)
	}

	if len(result.Data) != 0 {
		t.Errorf("Expected 0 tasks, got %d", len(result.Data))
	}
}

func TestNext_Success(t *testing.T) {
	server := newMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v1/projects/testproject/tasks/ready" && r.Method == "GET" {
			resp := map[string]interface{}{
				"data": []domain.Task{
					{
						ID:        "abc123",
						Title:     "Highest priority task",
						Status:    domain.StatusOpen,
						Priority:  0,
						CreatedAt: time.Now(),
						UpdatedAt: time.Now(),
					},
				},
				"pagination": map[string]interface{}{
					"page":        1,
					"per_page":    1,
					"total":       5,
					"total_pages": 5,
				},
			}
			json.NewEncoder(w).Encode(resp)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	})
	defer server.Close()

	host, port := parseURL(server.URL)
	c := client.NewClient(host, port, "testproject", "test@host:/path")

	result, err := c.ListReadyTasks(context.Background(), 1, 1)
	if err != nil {
		t.Fatalf("ListReadyTasks failed: %v", err)
	}

	if len(result.Data) != 1 {
		t.Errorf("Expected 1 task, got %d", len(result.Data))
	}

	if result.Data[0].Priority != 0 {
		t.Errorf("Expected priority 0, got %d", result.Data[0].Priority)
	}
}

func TestNext_NoTasks(t *testing.T) {
	server := newMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v1/projects/testproject/tasks/ready" && r.Method == "GET" {
			resp := map[string]interface{}{
				"data": []domain.Task{},
				"pagination": map[string]interface{}{
					"page":        1,
					"per_page":    1,
					"total":       0,
					"total_pages": 0,
				},
			}
			json.NewEncoder(w).Encode(resp)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	})
	defer server.Close()

	host, port := parseURL(server.URL)
	c := client.NewClient(host, port, "testproject", "test@host:/path")

	result, err := c.ListReadyTasks(context.Background(), 1, 1)
	if err != nil {
		t.Fatalf("ListReadyTasks failed: %v", err)
	}

	if len(result.Data) != 0 {
		t.Errorf("Expected 0 tasks, got %d", len(result.Data))
	}
}
