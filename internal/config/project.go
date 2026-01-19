package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

const (
	// ConfigFileName is the name of the project configuration file
	ConfigFileName = "airyra.toml"

	// DefaultServerHost is the default server host
	DefaultServerHost = "localhost"

	// DefaultServerPort is the default server port
	DefaultServerPort = 7432
)

// ProjectConfig represents the project-level configuration from airyra.toml
type ProjectConfig struct {
	Project    string `toml:"project"`
	ServerHost string `toml:"-"`
	ServerPort int    `toml:"-"`

	// Track whether values were explicitly set in config file
	hostExplicitlySet bool
	portExplicitlySet bool
}

// projectConfigFile represents the raw TOML structure
type projectConfigFile struct {
	Project string       `toml:"project"`
	Server  serverConfig `toml:"server"`
}

// serverConfig represents the [server] section in TOML
type serverConfig struct {
	Host string `toml:"host"`
	Port *int   `toml:"port"`
}

// DiscoverProjectConfig finds and parses the airyra.toml file by traversing
// up the directory tree from the current working directory.
func DiscoverProjectConfig() (*ProjectConfig, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("failed to get current directory: %w", err)
	}

	return discoverProjectConfigFrom(cwd)
}

// discoverProjectConfigFrom searches for airyra.toml starting from the given directory
func discoverProjectConfigFrom(startDir string) (*ProjectConfig, error) {
	dir := startDir

	for {
		configPath := filepath.Join(dir, ConfigFileName)
		if _, err := os.Stat(configPath); err == nil {
			return ParseProjectConfig(configPath)
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			// Reached filesystem root
			return nil, errors.New("No airyra.toml found. Run 'ar init <name>' to create one.")
		}
		dir = parent
	}
}

// ParseProjectConfig parses the airyra.toml file at the given path
func ParseProjectConfig(path string) (*ProjectConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var rawConfig projectConfigFile
	if _, err := toml.Decode(string(data), &rawConfig); err != nil {
		return nil, fmt.Errorf("failed to parse TOML: %w", err)
	}

	// Validate required fields
	if rawConfig.Project == "" {
		return nil, errors.New("project name cannot be empty")
	}

	// Validate port if explicitly specified in config
	if rawConfig.Server.Port != nil {
		if err := validatePort(*rawConfig.Server.Port); err != nil {
			return nil, err
		}
	}

	// Apply defaults
	cfg := &ProjectConfig{
		Project:    rawConfig.Project,
		ServerHost: DefaultServerHost,
		ServerPort: DefaultServerPort,
	}

	// Override with values from config if specified
	if rawConfig.Server.Host != "" {
		cfg.ServerHost = rawConfig.Server.Host
		cfg.hostExplicitlySet = true
	}
	if rawConfig.Server.Port != nil {
		cfg.ServerPort = *rawConfig.Server.Port
		cfg.portExplicitlySet = true
	}

	return cfg, nil
}

// HostExplicitlySet returns true if the host was explicitly set in the config file
func (c *ProjectConfig) HostExplicitlySet() bool {
	return c.hostExplicitlySet
}

// PortExplicitlySet returns true if the port was explicitly set in the config file
func (c *ProjectConfig) PortExplicitlySet() bool {
	return c.portExplicitlySet
}

// validatePort checks if the port is in the valid range (1-65535)
func validatePort(port int) error {
	if port < 1 || port > 65535 {
		return fmt.Errorf("invalid port %d: must be between 1 and 65535", port)
	}
	return nil
}
