package main

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/airyra/airyra/internal/client"
	"github.com/airyra/airyra/internal/config"
	"github.com/airyra/airyra/internal/domain"
	"github.com/airyra/airyra/internal/identity"
)

// getClient creates a client from the resolved config and identity
func getClient() (*client.Client, error) {
	cfg, err := config.ResolveConfig()
	if err != nil {
		return nil, err
	}

	agentID := identity.Generate()
	return client.NewClient(cfg.ServerHost, cfg.ServerPort, cfg.Project, agentID), nil
}

// mapErrorToExitCode maps an error to the appropriate exit code
func mapErrorToExitCode(err error) int {
	if err == nil {
		return ExitSuccess
	}

	// Check for client errors
	if errors.Is(err, client.ErrServerNotRunning) {
		return ExitServerNotRunning
	}

	// Check for domain errors
	var domainErr *domain.DomainError
	if errors.As(err, &domainErr) {
		switch domainErr.Code {
		case domain.ErrCodeTaskNotFound:
			return ExitTaskNotFound
		case domain.ErrCodeAlreadyClaimed:
			return ExitConflict
		case domain.ErrCodeNotOwner:
			return ExitPermissionDenied
		case domain.ErrCodeProjectNotFound:
			return ExitProjectNotConfigured
		case domain.ErrCodeDependencyNotFound:
			return ExitTaskNotFound
		default:
			return ExitGeneralError
		}
	}

	// Check for config errors
	if isConfigNotFoundError(err) {
		return ExitProjectNotConfigured
	}

	return ExitGeneralError
}

// isConfigNotFoundError checks if the error is a config not found error
func isConfigNotFoundError(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(err.Error(), "No airyra.toml found")
}

// handleError handles an error by printing it and exiting with the appropriate code
func handleError(err error) {
	if err == nil {
		return
	}

	printError(os.Stderr, err, jsonOutput)
	os.Exit(mapErrorToExitCode(err))
}

// parsePriority parses a priority string (name or number) into an int
func parsePriority(s string) (int, error) {
	// Try parsing as number first
	if n, err := strconv.Atoi(s); err == nil {
		if n < 0 || n > 4 {
			return 0, fmt.Errorf("priority must be between 0-4, got %d", n)
		}
		return n, nil
	}

	// Try parsing as name
	switch strings.ToLower(s) {
	case "critical":
		return 0, nil
	case "high":
		return 1, nil
	case "normal":
		return 2, nil
	case "low":
		return 3, nil
	case "lowest":
		return 4, nil
	default:
		return 0, fmt.Errorf("invalid priority: %s (use 0-4 or critical/high/normal/low/lowest)", s)
	}
}

// pidFilePath returns the path to the PID file
func pidFilePath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}
	return homeDir + "/" + config.GlobalConfigDir + "/airyra.pid", nil
}
