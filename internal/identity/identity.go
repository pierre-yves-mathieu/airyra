// Package identity provides agent identity generation for the Airyra CLI.
// The agent identity is used for HTTP request headers (X-Airyra-Agent),
// audit logging, and claim ownership verification.
package identity

import (
	"fmt"
	"os"
	"os/user"
)

const (
	// FallbackUser is used when the user cannot be determined
	FallbackUser = "unknown"
	// FallbackHostname is used when the hostname cannot be determined
	FallbackHostname = "localhost"
	// FallbackCwd is used when the current working directory cannot be determined
	FallbackCwd = "."
)

// Generate returns the agent identity string in the format: user@hostname:cwd
// It attempts to determine the user, hostname, and current working directory
// from the system, falling back to default values if any cannot be determined.
//
// The identity is used for:
//   - X-Airyra-Agent HTTP header on all requests
//   - Audit log entries
//   - Claim ownership verification
//
// Examples:
//   - alice@macbook:/Users/alice/projects/myapp
//   - dev@server:/home/dev/backend
func Generate() string {
	return GenerateWithOverrides(getUser(), getHostname(), getCwd())
}

// GenerateWithOverrides returns the agent identity string using the provided
// values, applying fallbacks for any empty values. This function is primarily
// intended for testing purposes where controlled inputs are needed.
func GenerateWithOverrides(usr, hostname, cwd string) string {
	if usr == "" {
		usr = FallbackUser
	}
	if hostname == "" {
		hostname = FallbackHostname
	}
	if cwd == "" {
		cwd = FallbackCwd
	}

	return fmt.Sprintf("%s@%s:%s", usr, hostname, cwd)
}

// getUser returns the current user's username.
// It first checks the USER environment variable, then falls back to user.Current().
func getUser() string {
	// First try USER environment variable
	if usr := os.Getenv("USER"); usr != "" {
		return usr
	}

	// Fall back to user.Current()
	if u, err := user.Current(); err == nil && u.Username != "" {
		return u.Username
	}

	return ""
}

// getHostname returns the system hostname.
func getHostname() string {
	if hostname, err := os.Hostname(); err == nil && hostname != "" {
		return hostname
	}
	return ""
}

// getCwd returns the current working directory.
func getCwd() string {
	if cwd, err := os.Getwd(); err == nil && cwd != "" {
		return cwd
	}
	return ""
}
