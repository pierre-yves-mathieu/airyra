package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/airyra/airyra/internal/config"
)

func TestInit_CreatesConfig(t *testing.T) {
	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "airyra-init-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Change to temp directory
	oldWd, _ := os.Getwd()
	defer os.Chdir(oldWd)
	os.Chdir(tmpDir)

	// Run init command
	err = runInit("myproject", "", 0)
	if err != nil {
		t.Fatalf("runInit failed: %v", err)
	}

	// Check config file was created
	configPath := filepath.Join(tmpDir, config.ConfigFileName)
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Error("Config file was not created")
	}

	// Check content
	content, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("Failed to read config: %v", err)
	}

	if !strings.Contains(string(content), `project = "myproject"`) {
		t.Error("Config file should contain project name")
	}
}

func TestInit_ConfigAlreadyExists(t *testing.T) {
	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "airyra-init-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create existing config
	configPath := filepath.Join(tmpDir, config.ConfigFileName)
	err = os.WriteFile(configPath, []byte(`project = "existing"`), 0644)
	if err != nil {
		t.Fatalf("Failed to create existing config: %v", err)
	}

	// Change to temp directory
	oldWd, _ := os.Getwd()
	defer os.Chdir(oldWd)
	os.Chdir(tmpDir)

	// Run init command - should fail
	err = runInit("newproject", "", 0)
	if err == nil {
		t.Error("runInit should fail when config already exists")
	}

	if !strings.Contains(err.Error(), "already exists") {
		t.Errorf("Error message should mention 'already exists', got: %v", err)
	}
}

func TestInit_WithHostAndPort(t *testing.T) {
	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "airyra-init-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Change to temp directory
	oldWd, _ := os.Getwd()
	defer os.Chdir(oldWd)
	os.Chdir(tmpDir)

	// Run init command with host and port
	err = runInit("myproject", "example.com", 8080)
	if err != nil {
		t.Fatalf("runInit failed: %v", err)
	}

	// Check content
	configPath := filepath.Join(tmpDir, config.ConfigFileName)
	content, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("Failed to read config: %v", err)
	}

	contentStr := string(content)
	if !strings.Contains(contentStr, `project = "myproject"`) {
		t.Error("Config file should contain project name")
	}
	if !strings.Contains(contentStr, `host = "example.com"`) {
		t.Error("Config file should contain host")
	}
	if !strings.Contains(contentStr, `port = 8080`) {
		t.Error("Config file should contain port")
	}
}

func TestInit_WithHostOnly(t *testing.T) {
	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "airyra-init-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Change to temp directory
	oldWd, _ := os.Getwd()
	defer os.Chdir(oldWd)
	os.Chdir(tmpDir)

	// Run init command with host only
	err = runInit("myproject", "example.com", 0)
	if err != nil {
		t.Fatalf("runInit failed: %v", err)
	}

	// Check content
	configPath := filepath.Join(tmpDir, config.ConfigFileName)
	content, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("Failed to read config: %v", err)
	}

	contentStr := string(content)
	if !strings.Contains(contentStr, `host = "example.com"`) {
		t.Error("Config file should contain host")
	}
	// Should not contain port if not specified
	if strings.Contains(contentStr, "port =") {
		t.Error("Config file should not contain port when not specified")
	}
}

func TestInit_WithPortOnly(t *testing.T) {
	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "airyra-init-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Change to temp directory
	oldWd, _ := os.Getwd()
	defer os.Chdir(oldWd)
	os.Chdir(tmpDir)

	// Run init command with port only
	err = runInit("myproject", "", 8080)
	if err != nil {
		t.Fatalf("runInit failed: %v", err)
	}

	// Check content
	configPath := filepath.Join(tmpDir, config.ConfigFileName)
	content, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("Failed to read config: %v", err)
	}

	contentStr := string(content)
	if !strings.Contains(contentStr, `port = 8080`) {
		t.Error("Config file should contain port")
	}
	// Should not contain host if not specified
	if strings.Contains(contentStr, "host =") {
		t.Error("Config file should not contain host when not specified")
	}
}

func TestInit_MissingName(t *testing.T) {
	err := runInit("", "", 0)
	if err == nil {
		t.Error("runInit should fail with empty project name")
	}

	if !strings.Contains(err.Error(), "project name") {
		t.Errorf("Error message should mention 'project name', got: %v", err)
	}
}

func TestInitCmd_Exists(t *testing.T) {
	if initCmd == nil {
		t.Error("initCmd should not be nil")
	}
}

func TestInitCmd_Use(t *testing.T) {
	if initCmd.Use != "init <name>" {
		t.Errorf("initCmd.Use = %s, expected 'init <name>'", initCmd.Use)
	}
}

func TestInitCmd_HasHostFlag(t *testing.T) {
	flag := initCmd.Flags().Lookup("host")
	if flag == nil {
		t.Error("initCmd should have --host flag")
	}
}

func TestInitCmd_HasPortFlag(t *testing.T) {
	flag := initCmd.Flags().Lookup("port")
	if flag == nil {
		t.Error("initCmd should have --port flag")
	}
}
