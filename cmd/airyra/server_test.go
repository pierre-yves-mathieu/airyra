package main

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

func TestServerCmd_Exists(t *testing.T) {
	if serverCmd == nil {
		t.Error("serverCmd should not be nil")
	}
}

func TestServerCmd_Use(t *testing.T) {
	if serverCmd.Use != "server" {
		t.Errorf("serverCmd.Use = %s, expected server", serverCmd.Use)
	}
}

func TestServerStartCmd_Exists(t *testing.T) {
	if serverStartCmd == nil {
		t.Error("serverStartCmd should not be nil")
	}
}

func TestServerStopCmd_Exists(t *testing.T) {
	if serverStopCmd == nil {
		t.Error("serverStopCmd should not be nil")
	}
}

func TestServerStatusCmd_Exists(t *testing.T) {
	if serverStatusCmd == nil {
		t.Error("serverStatusCmd should not be nil")
	}
}

func TestWritePIDFile(t *testing.T) {
	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "airyra-server-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	pidPath := filepath.Join(tmpDir, "test.pid")
	pid := 12345

	err = writePIDFile(pidPath, pid)
	if err != nil {
		t.Fatalf("writePIDFile failed: %v", err)
	}

	// Check file exists
	content, err := os.ReadFile(pidPath)
	if err != nil {
		t.Fatalf("Failed to read PID file: %v", err)
	}

	if strings.TrimSpace(string(content)) != strconv.Itoa(pid) {
		t.Errorf("PID file content = %s, expected %d", content, pid)
	}
}

func TestReadPIDFile(t *testing.T) {
	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "airyra-server-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	pidPath := filepath.Join(tmpDir, "test.pid")
	expectedPID := 12345

	// Write PID file
	err = os.WriteFile(pidPath, []byte(strconv.Itoa(expectedPID)), 0644)
	if err != nil {
		t.Fatalf("Failed to write PID file: %v", err)
	}

	// Read PID file
	pid, err := readPIDFile(pidPath)
	if err != nil {
		t.Fatalf("readPIDFile failed: %v", err)
	}

	if pid != expectedPID {
		t.Errorf("readPIDFile() = %d, expected %d", pid, expectedPID)
	}
}

func TestReadPIDFile_NotExists(t *testing.T) {
	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "airyra-server-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	pidPath := filepath.Join(tmpDir, "nonexistent.pid")

	pid, err := readPIDFile(pidPath)
	if err == nil {
		t.Error("readPIDFile should fail for non-existent file")
	}
	if pid != 0 {
		t.Errorf("readPIDFile() = %d, expected 0", pid)
	}
}

func TestReadPIDFile_InvalidContent(t *testing.T) {
	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "airyra-server-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	pidPath := filepath.Join(tmpDir, "invalid.pid")

	// Write invalid content
	err = os.WriteFile(pidPath, []byte("not-a-number"), 0644)
	if err != nil {
		t.Fatalf("Failed to write PID file: %v", err)
	}

	pid, err := readPIDFile(pidPath)
	if err == nil {
		t.Error("readPIDFile should fail for invalid content")
	}
	if pid != 0 {
		t.Errorf("readPIDFile() = %d, expected 0", pid)
	}
}

func TestRemovePIDFile(t *testing.T) {
	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "airyra-server-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	pidPath := filepath.Join(tmpDir, "test.pid")

	// Create PID file
	err = os.WriteFile(pidPath, []byte("12345"), 0644)
	if err != nil {
		t.Fatalf("Failed to write PID file: %v", err)
	}

	// Remove PID file
	err = removePIDFile(pidPath)
	if err != nil {
		t.Fatalf("removePIDFile failed: %v", err)
	}

	// Check file doesn't exist
	if _, err := os.Stat(pidPath); !os.IsNotExist(err) {
		t.Error("PID file should be removed")
	}
}

func TestRemovePIDFile_NotExists(t *testing.T) {
	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "airyra-server-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	pidPath := filepath.Join(tmpDir, "nonexistent.pid")

	// Should not error for non-existent file
	err = removePIDFile(pidPath)
	if err != nil {
		t.Errorf("removePIDFile should not fail for non-existent file: %v", err)
	}
}

func TestIsProcessRunning_NonExistent(t *testing.T) {
	// Use a PID that is very unlikely to exist
	// (negative PIDs are invalid, 0 is the kernel)
	running := isProcessRunning(999999999)
	if running {
		t.Error("isProcessRunning(999999999) should return false")
	}
}

func TestServerStart_HasBindFlag(t *testing.T) {
	flag := serverStartCmd.Flags().Lookup("bind")
	if flag == nil {
		t.Error("serverStartCmd should have --bind flag")
	}
}
