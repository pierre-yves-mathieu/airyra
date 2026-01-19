package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestGlobal_FileExists(t *testing.T) {
	// Create temp directory to act as home
	tmpDir := t.TempDir()
	airyraDir := filepath.Join(tmpDir, ".airyra")
	if err := os.Mkdir(airyraDir, 0755); err != nil {
		t.Fatalf("failed to create .airyra directory: %v", err)
	}

	configPath := filepath.Join(airyraDir, "config.toml")
	content := `
[server]
host = "global-host.example.com"
port = 9999
`
	if err := os.WriteFile(configPath, []byte(content), 0644); err != nil {
		t.Fatalf("failed to create test config: %v", err)
	}

	cfg, err := LoadGlobalConfigFromDir(tmpDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.ServerHost != "global-host.example.com" {
		t.Errorf("expected host 'global-host.example.com', got '%s'", cfg.ServerHost)
	}
	if cfg.ServerPort != 9999 {
		t.Errorf("expected port 9999, got %d", cfg.ServerPort)
	}
}

func TestGlobal_FileNotExists(t *testing.T) {
	// Create empty temp directory (no config file)
	tmpDir := t.TempDir()

	cfg, err := LoadGlobalConfigFromDir(tmpDir)
	if err != nil {
		t.Fatalf("expected no error when config doesn't exist, got: %v", err)
	}

	// Should return empty config (zero values)
	if cfg.ServerHost != "" {
		t.Errorf("expected empty host, got '%s'", cfg.ServerHost)
	}
	if cfg.ServerPort != 0 {
		t.Errorf("expected zero port, got %d", cfg.ServerPort)
	}
}

func TestGlobal_InvalidTOML(t *testing.T) {
	// Create temp directory with invalid TOML
	tmpDir := t.TempDir()
	airyraDir := filepath.Join(tmpDir, ".airyra")
	if err := os.Mkdir(airyraDir, 0755); err != nil {
		t.Fatalf("failed to create .airyra directory: %v", err)
	}

	configPath := filepath.Join(airyraDir, "config.toml")
	content := `this is not valid toml {{{`
	if err := os.WriteFile(configPath, []byte(content), 0644); err != nil {
		t.Fatalf("failed to create test config: %v", err)
	}

	_, err := LoadGlobalConfigFromDir(tmpDir)
	if err == nil {
		t.Fatal("expected error for invalid TOML")
	}
}

func TestGlobal_PartialConfig(t *testing.T) {
	// Create temp directory with only host set
	tmpDir := t.TempDir()
	airyraDir := filepath.Join(tmpDir, ".airyra")
	if err := os.Mkdir(airyraDir, 0755); err != nil {
		t.Fatalf("failed to create .airyra directory: %v", err)
	}

	configPath := filepath.Join(airyraDir, "config.toml")
	content := `
[server]
host = "partial-host.example.com"
`
	if err := os.WriteFile(configPath, []byte(content), 0644); err != nil {
		t.Fatalf("failed to create test config: %v", err)
	}

	cfg, err := LoadGlobalConfigFromDir(tmpDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.ServerHost != "partial-host.example.com" {
		t.Errorf("expected host 'partial-host.example.com', got '%s'", cfg.ServerHost)
	}
	// Port should be zero (not set)
	if cfg.ServerPort != 0 {
		t.Errorf("expected zero port, got %d", cfg.ServerPort)
	}
}

func TestGlobal_EmptyFile(t *testing.T) {
	// Create temp directory with empty config file
	tmpDir := t.TempDir()
	airyraDir := filepath.Join(tmpDir, ".airyra")
	if err := os.Mkdir(airyraDir, 0755); err != nil {
		t.Fatalf("failed to create .airyra directory: %v", err)
	}

	configPath := filepath.Join(airyraDir, "config.toml")
	if err := os.WriteFile(configPath, []byte(""), 0644); err != nil {
		t.Fatalf("failed to create test config: %v", err)
	}

	cfg, err := LoadGlobalConfigFromDir(tmpDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should return empty config (zero values)
	if cfg.ServerHost != "" {
		t.Errorf("expected empty host, got '%s'", cfg.ServerHost)
	}
	if cfg.ServerPort != 0 {
		t.Errorf("expected zero port, got %d", cfg.ServerPort)
	}
}
