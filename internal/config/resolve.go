package config

import "os"

// ResolvedConfig represents the final merged configuration with all
// precedence rules applied. Precedence order (highest to lowest):
// 1. Project config (airyra.toml)
// 2. Global config (~/.airyra/config.toml)
// 3. Built-in defaults (localhost:7432)
type ResolvedConfig struct {
	Project    string
	ServerHost string
	ServerPort int
}

// ResolveConfig discovers the project config, loads the global config,
// and merges them according to precedence rules.
func ResolveConfig() (*ResolvedConfig, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	return ResolveConfigWithHome(homeDir)
}

// ResolveConfigWithHome resolves config using a specified home directory.
// This is useful for testing.
func ResolveConfigWithHome(homeDir string) (*ResolvedConfig, error) {
	// Step 1: Discover project config (required)
	projectCfg, err := DiscoverProjectConfig()
	if err != nil {
		return nil, err
	}

	// Step 2: Load global config (optional, errors are not ignored for invalid files)
	globalCfg, err := LoadGlobalConfigFromDir(homeDir)
	if err != nil {
		return nil, err
	}

	// Step 3: Merge with precedence (defaults -> global -> project)
	resolved := &ResolvedConfig{
		Project:    projectCfg.Project,
		ServerHost: DefaultServerHost,
		ServerPort: DefaultServerPort,
	}

	// Apply global config (overrides defaults)
	if globalCfg.ServerHost != "" {
		resolved.ServerHost = globalCfg.ServerHost
	}
	if globalCfg.ServerPort != 0 {
		resolved.ServerPort = globalCfg.ServerPort
	}

	// Apply project config (overrides global and defaults, only if explicitly set)
	if projectCfg.HostExplicitlySet() {
		resolved.ServerHost = projectCfg.ServerHost
	}
	if projectCfg.PortExplicitlySet() {
		resolved.ServerPort = projectCfg.ServerPort
	}

	return resolved, nil
}
