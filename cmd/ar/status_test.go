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

func TestClaimCmd_Exists(t *testing.T) {
	if claimCmd == nil {
		t.Error("claimCmd should not be nil")
	}
}

func TestClaimCmd_Use(t *testing.T) {
	if claimCmd.Use != "claim <id>" {
		t.Errorf("claimCmd.Use = %s, expected 'claim <id>'", claimCmd.Use)
	}
}

func TestDoneCmd_Exists(t *testing.T) {
	if doneCmd == nil {
		t.Error("doneCmd should not be nil")
	}
}

func TestDoneCmd_Use(t *testing.T) {
	if doneCmd.Use != "done <id>" {
		t.Errorf("doneCmd.Use = %s, expected 'done <id>'", doneCmd.Use)
	}
}

func TestReleaseCmd_Exists(t *testing.T) {
	if releaseCmd == nil {
		t.Error("releaseCmd should not be nil")
	}
}

func TestReleaseCmd_HasForceFlag(t *testing.T) {
	flag := releaseCmd.Flags().Lookup("force")
	if flag == nil {
		t.Error("releaseCmd should have --force flag")
	}
}

func TestBlockCmd_Exists(t *testing.T) {
	if blockCmd == nil {
		t.Error("blockCmd should not be nil")
	}
}

func TestUnblockCmd_Exists(t *testing.T) {
	if unblockCmd == nil {
		t.Error("unblockCmd should not be nil")
	}
}

func TestClaim_Success(t *testing.T) {
	claimedBy := "test@host:/path"
	claimedAt := time.Now()

	server := newMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v1/projects/testproject/tasks/abc123/claim" && r.Method == "POST" {
			task := domain.Task{
				ID:        "abc123",
				Title:     "Test task",
				Status:    domain.StatusInProgress,
				Priority:  2,
				ClaimedBy: &claimedBy,
				ClaimedAt: &claimedAt,
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			}
			json.NewEncoder(w).Encode(task)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	})
	defer server.Close()

	host, port := parseURL(server.URL)
	c := client.NewClient(host, port, "testproject", "test@host:/path")

	task, err := c.ClaimTask(context.Background(), "abc123")
	if err != nil {
		t.Fatalf("ClaimTask failed: %v", err)
	}

	if task.Status != domain.StatusInProgress {
		t.Errorf("Expected status in_progress, got %s", task.Status)
	}
	if task.ClaimedBy == nil || *task.ClaimedBy != claimedBy {
		t.Error("Expected task to be claimed by test@host:/path")
	}
}

func TestClaim_AlreadyClaimed(t *testing.T) {
	server := newMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusConflict)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error": map[string]interface{}{
				"code":    "ALREADY_CLAIMED",
				"message": "Task already claimed by another agent",
				"context": map[string]interface{}{
					"claimed_by": "other@host:/path",
					"claimed_at": "2024-01-15T10:30:00Z",
				},
			},
		})
	})
	defer server.Close()

	host, port := parseURL(server.URL)
	c := client.NewClient(host, port, "testproject", "test@host:/path")

	_, err := c.ClaimTask(context.Background(), "abc123")
	if err == nil {
		t.Error("ClaimTask should fail for already claimed task")
	}

	domainErr, ok := err.(*domain.DomainError)
	if !ok {
		t.Errorf("Expected DomainError, got %T", err)
		return
	}

	if domainErr.Code != domain.ErrCodeAlreadyClaimed {
		t.Errorf("Expected ALREADY_CLAIMED error code, got %s", domainErr.Code)
	}
}

func TestDone_Success(t *testing.T) {
	server := newMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v1/projects/testproject/tasks/abc123/done" && r.Method == "POST" {
			task := domain.Task{
				ID:        "abc123",
				Title:     "Test task",
				Status:    domain.StatusDone,
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

	host, port := parseURL(server.URL)
	c := client.NewClient(host, port, "testproject", "test@host:/path")

	task, err := c.CompleteTask(context.Background(), "abc123")
	if err != nil {
		t.Fatalf("CompleteTask failed: %v", err)
	}

	if task.Status != domain.StatusDone {
		t.Errorf("Expected status done, got %s", task.Status)
	}
}

func TestRelease_Success(t *testing.T) {
	server := newMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v1/projects/testproject/tasks/abc123/release" && r.Method == "POST" {
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

	host, port := parseURL(server.URL)
	c := client.NewClient(host, port, "testproject", "test@host:/path")

	task, err := c.ReleaseTask(context.Background(), "abc123", false)
	if err != nil {
		t.Fatalf("ReleaseTask failed: %v", err)
	}

	if task.Status != domain.StatusOpen {
		t.Errorf("Expected status open, got %s", task.Status)
	}
}

func TestRelease_WithForce(t *testing.T) {
	forceCalled := false
	server := newMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v1/projects/testproject/tasks/abc123/release" && r.Method == "POST" {
			if r.URL.Query().Get("force") == "true" {
				forceCalled = true
			}
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

	host, port := parseURL(server.URL)
	c := client.NewClient(host, port, "testproject", "test@host:/path")

	_, err := c.ReleaseTask(context.Background(), "abc123", true)
	if err != nil {
		t.Fatalf("ReleaseTask failed: %v", err)
	}

	if !forceCalled {
		t.Error("Expected force=true query parameter")
	}
}

func TestBlock_Success(t *testing.T) {
	server := newMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v1/projects/testproject/tasks/abc123/block" && r.Method == "POST" {
			task := domain.Task{
				ID:        "abc123",
				Title:     "Test task",
				Status:    domain.StatusBlocked,
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

	host, port := parseURL(server.URL)
	c := client.NewClient(host, port, "testproject", "test@host:/path")

	task, err := c.BlockTask(context.Background(), "abc123")
	if err != nil {
		t.Fatalf("BlockTask failed: %v", err)
	}

	if task.Status != domain.StatusBlocked {
		t.Errorf("Expected status blocked, got %s", task.Status)
	}
}

func TestUnblock_Success(t *testing.T) {
	server := newMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v1/projects/testproject/tasks/abc123/unblock" && r.Method == "POST" {
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

	host, port := parseURL(server.URL)
	c := client.NewClient(host, port, "testproject", "test@host:/path")

	task, err := c.UnblockTask(context.Background(), "abc123")
	if err != nil {
		t.Fatalf("UnblockTask failed: %v", err)
	}

	if task.Status != domain.StatusOpen {
		t.Errorf("Expected status open, got %s", task.Status)
	}
}
