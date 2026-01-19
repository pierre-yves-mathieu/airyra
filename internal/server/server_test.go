package server_test

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/airyra/airyra/internal/server"
	"github.com/airyra/airyra/internal/store"
)

func TestServer_StartAndShutdown(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "airyra-server-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	manager, err := store.NewManager(tmpDir)
	if err != nil {
		t.Fatalf("failed to create manager: %v", err)
	}
	defer manager.Close()

	srv := server.New("localhost:0", manager)

	// Start server in background
	errChan := make(chan error, 1)
	go func() {
		errChan <- srv.Start()
	}()

	// Wait a bit for server to start
	time.Sleep(100 * time.Millisecond)

	// Shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		t.Errorf("shutdown error: %v", err)
	}

	// Check if Start returned (it should after Shutdown)
	select {
	case err := <-errChan:
		// http.ErrServerClosed is expected
		if err != nil && err != http.ErrServerClosed {
			t.Errorf("unexpected error from Start: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Error("server did not stop after shutdown")
	}
}

func TestServer_GracefulShutdown(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "airyra-server-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	manager, err := store.NewManager(tmpDir)
	if err != nil {
		t.Fatalf("failed to create manager: %v", err)
	}
	defer manager.Close()

	srv := server.New("localhost:0", manager)

	// Start server in background
	go func() {
		srv.Start()
	}()

	// Wait for server to start
	time.Sleep(100 * time.Millisecond)

	// Get the address
	addr := srv.Addr()
	if addr == "" {
		t.Fatal("server address not available")
	}

	// Make a request to verify server is running
	resp, err := http.Get("http://" + addr + "/v1/health")
	if err != nil {
		t.Fatalf("failed to make request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}

	var health map[string]string
	if err := json.NewDecoder(resp.Body).Decode(&health); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if health["status"] != "ok" {
		t.Errorf("expected status 'ok', got %q", health["status"])
	}

	// Graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		t.Errorf("shutdown error: %v", err)
	}
}

func TestServer_DefaultAddress(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "airyra-server-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	manager, err := store.NewManager(tmpDir)
	if err != nil {
		t.Fatalf("failed to create manager: %v", err)
	}
	defer manager.Close()

	// Test with empty address - should use default
	srv := server.New("", manager)
	if srv.DefaultAddr() != "localhost:7432" {
		t.Errorf("expected default address 'localhost:7432', got %q", srv.DefaultAddr())
	}
}
