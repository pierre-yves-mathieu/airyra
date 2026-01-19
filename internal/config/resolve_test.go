package config

import (
	"os"
	"path/filepath"
	"testing"
)

// Helper to set up test environment with project and global configs
type testEnv struct {
	projectDir string
	homeDir    string
	originalWd string
}

func setupTestEnv(t *testing.T) *testEnv {
	t.Helper()
	env := &testEnv{
		projectDir: t.TempDir(),
		homeDir:    t.TempDir(),
	}

	var err error
	env.originalWd, err = os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working directory: %v", err)
	}

	if err := os.Chdir(env.projectDir); err != nil {
		t.Fatalf("failed to change directory: %v", err)
	}

	return env
}

func (e *testEnv) cleanup(t *testing.T) {
	t.Helper()
	if err := os.Chdir(e.originalWd); err != nil {
		t.Fatalf("failed to restore working directory: %v", err)
	}
}

func (e *testEnv) writeProjectConfig(t *testing.T, content string) {
	t.Helper()
	configPath := filepath.Join(e.projectDir, "airyra.toml")
	if err := os.WriteFile(configPath, []byte(content), 0644); err != nil {
		t.Fatalf("failed to create project config: %v", err)
	}
}

func (e *testEnv) writeGlobalConfig(t *testing.T, content string) {
	t.Helper()
	airyraDir := filepath.Join(e.homeDir, ".airyra")
	if err := os.MkdirAll(airyraDir, 0755); err != nil {
		t.Fatalf("failed to create .airyra directory: %v", err)
	}
	configPath := filepath.Join(airyraDir, "config.toml")
	if err := os.WriteFile(configPath, []byte(content), 0644); err != nil {
		t.Fatalf("failed to create global config: %v", err)
	}
}

func TestPrecedence_ProjectOverridesGlobal(t *testing.T) {
	env := setupTestEnv(t)
	defer env.cleanup(t)

	// Global config with host and port
	env.writeGlobalConfig(t, `
[server]
host = "global-host"
port = 1111
`)

	// Project config overrides both
	env.writeProjectConfig(t, `
project = "test-app"

[server]
host = "project-host"
port = 2222
`)

	cfg, err := ResolveConfigWithHome(env.homeDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.Project != "test-app" {
		t.Errorf("expected project 'test-app', got '%s'", cfg.Project)
	}
	if cfg.ServerHost != "project-host" {
		t.Errorf("expected host 'project-host', got '%s'", cfg.ServerHost)
	}
	if cfg.ServerPort != 2222 {
		t.Errorf("expected port 2222, got %d", cfg.ServerPort)
	}
}

func TestPrecedence_GlobalOverridesDefaults(t *testing.T) {
	env := setupTestEnv(t)
	defer env.cleanup(t)

	// Global config with custom values
	env.writeGlobalConfig(t, `
[server]
host = "custom-global-host"
port = 5555
`)

	// Project config without server settings
	env.writeProjectConfig(t, `project = "test-app"`)

	cfg, err := ResolveConfigWithHome(env.homeDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.Project != "test-app" {
		t.Errorf("expected project 'test-app', got '%s'", cfg.Project)
	}
	// Should use global values, not defaults
	if cfg.ServerHost != "custom-global-host" {
		t.Errorf("expected host 'custom-global-host', got '%s'", cfg.ServerHost)
	}
	if cfg.ServerPort != 5555 {
		t.Errorf("expected port 5555, got %d", cfg.ServerPort)
	}
}

func TestPrecedence_DefaultsUsed(t *testing.T) {
	env := setupTestEnv(t)
	defer env.cleanup(t)

	// No global config

	// Project config without server settings
	env.writeProjectConfig(t, `project = "test-app"`)

	cfg, err := ResolveConfigWithHome(env.homeDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.Project != "test-app" {
		t.Errorf("expected project 'test-app', got '%s'", cfg.Project)
	}
	// Should use built-in defaults
	if cfg.ServerHost != "localhost" {
		t.Errorf("expected default host 'localhost', got '%s'", cfg.ServerHost)
	}
	if cfg.ServerPort != 7432 {
		t.Errorf("expected default port 7432, got %d", cfg.ServerPort)
	}
}

func TestPrecedence_MixedSources(t *testing.T) {
	env := setupTestEnv(t)
	defer env.cleanup(t)

	// Global config with only port
	env.writeGlobalConfig(t, `
[server]
port = 3333
`)

	// Project config with only host
	env.writeProjectConfig(t, `
project = "test-app"

[server]
host = "project-only-host"
`)

	cfg, err := ResolveConfigWithHome(env.homeDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.Project != "test-app" {
		t.Errorf("expected project 'test-app', got '%s'", cfg.Project)
	}
	// Host from project config
	if cfg.ServerHost != "project-only-host" {
		t.Errorf("expected host 'project-only-host', got '%s'", cfg.ServerHost)
	}
	// Port from global config
	if cfg.ServerPort != 3333 {
		t.Errorf("expected port 3333, got %d", cfg.ServerPort)
	}
}

func TestPrecedence_MixedSources_GlobalHostProjectPort(t *testing.T) {
	env := setupTestEnv(t)
	defer env.cleanup(t)

	// Global config with only host
	env.writeGlobalConfig(t, `
[server]
host = "global-only-host"
`)

	// Project config with only port
	env.writeProjectConfig(t, `
project = "test-app"

[server]
port = 4444
`)

	cfg, err := ResolveConfigWithHome(env.homeDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.Project != "test-app" {
		t.Errorf("expected project 'test-app', got '%s'", cfg.Project)
	}
	// Host from global config
	if cfg.ServerHost != "global-only-host" {
		t.Errorf("expected host 'global-only-host', got '%s'", cfg.ServerHost)
	}
	// Port from project config
	if cfg.ServerPort != 4444 {
		t.Errorf("expected port 4444, got %d", cfg.ServerPort)
	}
}

func TestResolve_NoProjectConfig(t *testing.T) {
	env := setupTestEnv(t)
	defer env.cleanup(t)

	// No project config, no global config

	_, err := ResolveConfigWithHome(env.homeDir)
	if err == nil {
		t.Fatal("expected error when no project config found")
	}
}

func TestResolve_GlobalConfigInvalid(t *testing.T) {
	env := setupTestEnv(t)
	defer env.cleanup(t)

	// Invalid global config
	airyraDir := filepath.Join(env.homeDir, ".airyra")
	if err := os.MkdirAll(airyraDir, 0755); err != nil {
		t.Fatalf("failed to create .airyra directory: %v", err)
	}
	configPath := filepath.Join(airyraDir, "config.toml")
	if err := os.WriteFile(configPath, []byte(`invalid {{{ toml`), 0644); err != nil {
		t.Fatalf("failed to create global config: %v", err)
	}

	// Valid project config
	env.writeProjectConfig(t, `project = "test-app"`)

	_, err := ResolveConfigWithHome(env.homeDir)
	if err == nil {
		t.Fatal("expected error when global config is invalid")
	}
}
