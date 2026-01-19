package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDiscovery_CurrentDirectory(t *testing.T) {
	// Create temp directory with airyra.toml
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "airyra.toml")
	if err := os.WriteFile(configPath, []byte(`project = "test-app"`), 0644); err != nil {
		t.Fatalf("failed to create test config: %v", err)
	}

	// Save and restore working directory
	originalWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working directory: %v", err)
	}
	defer os.Chdir(originalWd)

	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to change directory: %v", err)
	}

	cfg, err := DiscoverProjectConfig()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.Project != "test-app" {
		t.Errorf("expected project 'test-app', got '%s'", cfg.Project)
	}
}

func TestDiscovery_ParentDirectory(t *testing.T) {
	// Create temp directory structure: parent/child
	tmpDir := t.TempDir()
	childDir := filepath.Join(tmpDir, "child")
	if err := os.Mkdir(childDir, 0755); err != nil {
		t.Fatalf("failed to create child directory: %v", err)
	}

	// Put config in parent
	configPath := filepath.Join(tmpDir, "airyra.toml")
	if err := os.WriteFile(configPath, []byte(`project = "parent-app"`), 0644); err != nil {
		t.Fatalf("failed to create test config: %v", err)
	}

	// Save and restore working directory
	originalWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working directory: %v", err)
	}
	defer os.Chdir(originalWd)

	// Change to child directory
	if err := os.Chdir(childDir); err != nil {
		t.Fatalf("failed to change directory: %v", err)
	}

	cfg, err := DiscoverProjectConfig()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.Project != "parent-app" {
		t.Errorf("expected project 'parent-app', got '%s'", cfg.Project)
	}
}

func TestDiscovery_DeeplyNested(t *testing.T) {
	// Create deep directory structure: root/a/b/c/d
	tmpDir := t.TempDir()
	deepDir := filepath.Join(tmpDir, "a", "b", "c", "d")
	if err := os.MkdirAll(deepDir, 0755); err != nil {
		t.Fatalf("failed to create deep directory: %v", err)
	}

	// Put config in root
	configPath := filepath.Join(tmpDir, "airyra.toml")
	if err := os.WriteFile(configPath, []byte(`project = "deep-app"`), 0644); err != nil {
		t.Fatalf("failed to create test config: %v", err)
	}

	// Save and restore working directory
	originalWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working directory: %v", err)
	}
	defer os.Chdir(originalWd)

	// Change to deepest directory
	if err := os.Chdir(deepDir); err != nil {
		t.Fatalf("failed to change directory: %v", err)
	}

	cfg, err := DiscoverProjectConfig()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.Project != "deep-app" {
		t.Errorf("expected project 'deep-app', got '%s'", cfg.Project)
	}
}

func TestDiscovery_NotFound(t *testing.T) {
	// Create temp directory without airyra.toml
	tmpDir := t.TempDir()

	// Save and restore working directory
	originalWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working directory: %v", err)
	}
	defer os.Chdir(originalWd)

	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to change directory: %v", err)
	}

	_, err = DiscoverProjectConfig()
	if err == nil {
		t.Fatal("expected error when no config found")
	}

	expectedMsg := "No airyra.toml found. Run 'ar init <name>' to create one."
	if err.Error() != expectedMsg {
		t.Errorf("expected error message '%s', got '%s'", expectedMsg, err.Error())
	}
}

func TestParse_MinimalConfig(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "airyra.toml")

	content := `project = "minimal-app"`
	if err := os.WriteFile(configPath, []byte(content), 0644); err != nil {
		t.Fatalf("failed to create test config: %v", err)
	}

	cfg, err := ParseProjectConfig(configPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.Project != "minimal-app" {
		t.Errorf("expected project 'minimal-app', got '%s'", cfg.Project)
	}

	// Check defaults
	if cfg.ServerHost != "localhost" {
		t.Errorf("expected default host 'localhost', got '%s'", cfg.ServerHost)
	}
	if cfg.ServerPort != 7432 {
		t.Errorf("expected default port 7432, got %d", cfg.ServerPort)
	}
}

func TestParse_FullConfig(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "airyra.toml")

	content := `
project = "full-app"

[server]
host = "192.168.1.100"
port = 8080
`
	if err := os.WriteFile(configPath, []byte(content), 0644); err != nil {
		t.Fatalf("failed to create test config: %v", err)
	}

	cfg, err := ParseProjectConfig(configPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.Project != "full-app" {
		t.Errorf("expected project 'full-app', got '%s'", cfg.Project)
	}
	if cfg.ServerHost != "192.168.1.100" {
		t.Errorf("expected host '192.168.1.100', got '%s'", cfg.ServerHost)
	}
	if cfg.ServerPort != 8080 {
		t.Errorf("expected port 8080, got %d", cfg.ServerPort)
	}
}

func TestParse_MissingProject(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "airyra.toml")

	content := `
[server]
host = "localhost"
port = 8080
`
	if err := os.WriteFile(configPath, []byte(content), 0644); err != nil {
		t.Fatalf("failed to create test config: %v", err)
	}

	_, err := ParseProjectConfig(configPath)
	if err == nil {
		t.Fatal("expected error for missing project field")
	}

	if !strings.Contains(err.Error(), "project") {
		t.Errorf("error should mention 'project', got: %s", err.Error())
	}
}

func TestParse_EmptyProject(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "airyra.toml")

	content := `project = ""`
	if err := os.WriteFile(configPath, []byte(content), 0644); err != nil {
		t.Fatalf("failed to create test config: %v", err)
	}

	_, err := ParseProjectConfig(configPath)
	if err == nil {
		t.Fatal("expected error for empty project name")
	}

	if !strings.Contains(err.Error(), "empty") {
		t.Errorf("error should mention 'empty', got: %s", err.Error())
	}
}

func TestParse_InvalidPort_Zero(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "airyra.toml")

	content := `
project = "test-app"

[server]
port = 0
`
	if err := os.WriteFile(configPath, []byte(content), 0644); err != nil {
		t.Fatalf("failed to create test config: %v", err)
	}

	_, err := ParseProjectConfig(configPath)
	if err == nil {
		t.Fatal("expected error for port 0")
	}

	if !strings.Contains(err.Error(), "port") {
		t.Errorf("error should mention 'port', got: %s", err.Error())
	}
}

func TestParse_InvalidPort_Negative(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "airyra.toml")

	content := `
project = "test-app"

[server]
port = -1
`
	if err := os.WriteFile(configPath, []byte(content), 0644); err != nil {
		t.Fatalf("failed to create test config: %v", err)
	}

	_, err := ParseProjectConfig(configPath)
	if err == nil {
		t.Fatal("expected error for negative port")
	}

	if !strings.Contains(err.Error(), "port") {
		t.Errorf("error should mention 'port', got: %s", err.Error())
	}
}

func TestParse_InvalidPort_TooHigh(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "airyra.toml")

	content := `
project = "test-app"

[server]
port = 65536
`
	if err := os.WriteFile(configPath, []byte(content), 0644); err != nil {
		t.Fatalf("failed to create test config: %v", err)
	}

	_, err := ParseProjectConfig(configPath)
	if err == nil {
		t.Fatal("expected error for port > 65535")
	}

	if !strings.Contains(err.Error(), "port") {
		t.Errorf("error should mention 'port', got: %s", err.Error())
	}
}

func TestParse_InvalidTOML(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "airyra.toml")

	content := `this is not valid toml {{{`
	if err := os.WriteFile(configPath, []byte(content), 0644); err != nil {
		t.Fatalf("failed to create test config: %v", err)
	}

	_, err := ParseProjectConfig(configPath)
	if err == nil {
		t.Fatal("expected error for invalid TOML")
	}
}

func TestParse_FileNotFound(t *testing.T) {
	_, err := ParseProjectConfig("/nonexistent/path/airyra.toml")
	if err == nil {
		t.Fatal("expected error for non-existent file")
	}
}
