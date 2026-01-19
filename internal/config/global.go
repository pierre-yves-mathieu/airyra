package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

const (
	// GlobalConfigDir is the name of the global config directory in home
	GlobalConfigDir = ".airyra"

	// GlobalConfigFileName is the name of the global config file
	GlobalConfigFileName = "config.toml"
)

// GlobalConfig represents the user-level configuration from ~/.airyra/config.toml
type GlobalConfig struct {
	ServerHost string
	ServerPort int
}

// globalConfigFile represents the raw TOML structure for global config
type globalConfigFile struct {
	Server serverConfig `toml:"server"`
}

// LoadGlobalConfig loads the global configuration from ~/.airyra/config.toml.
// Returns an empty config (not an error) if the file doesn't exist.
func LoadGlobalConfig() (*GlobalConfig, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %w", err)
	}

	return LoadGlobalConfigFromDir(homeDir)
}

// LoadGlobalConfigFromDir loads global config using the specified directory as home.
// This is useful for testing.
func LoadGlobalConfigFromDir(homeDir string) (*GlobalConfig, error) {
	configPath := filepath.Join(homeDir, GlobalConfigDir, GlobalConfigFileName)

	// Check if file exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		// Return empty config if file doesn't exist
		return &GlobalConfig{}, nil
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read global config: %w", err)
	}

	var rawConfig globalConfigFile
	if _, err := toml.Decode(string(data), &rawConfig); err != nil {
		return nil, fmt.Errorf("failed to parse global config TOML: %w", err)
	}

	cfg := &GlobalConfig{
		ServerHost: rawConfig.Server.Host,
	}

	if rawConfig.Server.Port != nil {
		cfg.ServerPort = *rawConfig.Server.Port
	}

	return cfg, nil
}
