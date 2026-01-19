package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"

	"github.com/airyra/airyra/internal/config"
	"github.com/spf13/cobra"
)

var serverCmd = &cobra.Command{
	Use:   "server",
	Short: "Manage the airyra server",
	Long:  `Commands for starting, stopping, and checking the status of the airyra server.`,
}

var serverStartCmd = &cobra.Command{
	Use:   "start",
	Short: "Start the airyra server",
	Long:  `Start the airyra server as a background process.`,
	Run: func(cmd *cobra.Command, args []string) {
		bind, _ := cmd.Flags().GetString("bind")

		if err := runServerStart(bind); err != nil {
			handleError(err)
		}
	},
}

var serverStopCmd = &cobra.Command{
	Use:   "stop",
	Short: "Stop the airyra server",
	Long:  `Stop the running airyra server.`,
	Run: func(cmd *cobra.Command, args []string) {
		if err := runServerStop(); err != nil {
			handleError(err)
		}
	},
}

var serverStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Check airyra server status",
	Long:  `Check if the airyra server is running.`,
	Run: func(cmd *cobra.Command, args []string) {
		if err := runServerStatus(); err != nil {
			handleError(err)
		}
	},
}

func init() {
	rootCmd.AddCommand(serverCmd)

	serverCmd.AddCommand(serverStartCmd)
	serverCmd.AddCommand(serverStopCmd)
	serverCmd.AddCommand(serverStatusCmd)

	serverStartCmd.Flags().String("bind", "localhost:7432", "Address to bind the server to")
}

// runServerStart starts the airyra server
func runServerStart(bind string) error {
	pidPath, err := pidFilePath()
	if err != nil {
		return err
	}

	// Check if already running
	if pid, err := readPIDFile(pidPath); err == nil && isProcessRunning(pid) {
		return fmt.Errorf("server is already running (PID: %d)", pid)
	}

	// Find airyra binary
	airyraPath, err := exec.LookPath("airyra")
	if err != nil {
		// Try to find it relative to ar binary
		arPath, err := os.Executable()
		if err != nil {
			return fmt.Errorf("failed to find airyra binary: %w", err)
		}
		airyraPath = filepath.Join(filepath.Dir(arPath), "airyra")
		if _, err := os.Stat(airyraPath); os.IsNotExist(err) {
			return fmt.Errorf("airyra binary not found. Install it first.")
		}
	}

	// Start server in background
	cmd := exec.Command(airyraPath)
	cmd.Env = append(os.Environ(), fmt.Sprintf("AIRYRA_BIND=%s", bind))
	cmd.Stdout = nil
	cmd.Stderr = nil
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setsid: true, // Create new session to detach from terminal
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start server: %w", err)
	}

	// Write PID file
	if err := writePIDFile(pidPath, cmd.Process.Pid); err != nil {
		return fmt.Errorf("failed to write PID file: %w", err)
	}

	printSuccess(os.Stdout, fmt.Sprintf("Server started (PID: %d)", cmd.Process.Pid), jsonOutput)
	return nil
}

// runServerStop stops the airyra server
func runServerStop() error {
	pidPath, err := pidFilePath()
	if err != nil {
		return err
	}

	pid, err := readPIDFile(pidPath)
	if err != nil {
		return fmt.Errorf("server is not running (no PID file found)")
	}

	if !isProcessRunning(pid) {
		// Clean up stale PID file
		removePIDFile(pidPath)
		return fmt.Errorf("server is not running (stale PID file)")
	}

	// Send SIGTERM
	process, err := os.FindProcess(pid)
	if err != nil {
		return fmt.Errorf("failed to find process: %w", err)
	}

	if err := process.Signal(syscall.SIGTERM); err != nil {
		return fmt.Errorf("failed to stop server: %w", err)
	}

	// Remove PID file
	removePIDFile(pidPath)

	printSuccess(os.Stdout, fmt.Sprintf("Server stopped (PID: %d)", pid), jsonOutput)
	return nil
}

// runServerStatus checks if the server is running
func runServerStatus() error {
	pidPath, err := pidFilePath()
	if err != nil {
		return err
	}

	pid, err := readPIDFile(pidPath)
	if err != nil {
		printSuccess(os.Stdout, "Server is not running", jsonOutput)
		os.Exit(ExitServerNotRunning)
		return nil
	}

	if !isProcessRunning(pid) {
		// Clean up stale PID file
		removePIDFile(pidPath)
		printSuccess(os.Stdout, "Server is not running (stale PID file removed)", jsonOutput)
		os.Exit(ExitServerNotRunning)
		return nil
	}

	printSuccess(os.Stdout, fmt.Sprintf("Server is running (PID: %d)", pid), jsonOutput)
	return nil
}

// writePIDFile writes the process ID to a file
func writePIDFile(path string, pid int) error {
	// Ensure directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	return os.WriteFile(path, []byte(strconv.Itoa(pid)), 0644)
}

// readPIDFile reads the process ID from a file
func readPIDFile(path string) (int, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return 0, err
	}

	pid, err := strconv.Atoi(strings.TrimSpace(string(content)))
	if err != nil {
		return 0, fmt.Errorf("invalid PID file content: %w", err)
	}

	return pid, nil
}

// removePIDFile removes the PID file
func removePIDFile(path string) error {
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

// isProcessRunning checks if a process is running
func isProcessRunning(pid int) bool {
	process, err := os.FindProcess(pid)
	if err != nil {
		return false
	}

	// On Unix, FindProcess always succeeds, so we need to send signal 0
	// to check if the process exists
	err = process.Signal(syscall.Signal(0))
	return err == nil
}

// ensureAiryraDir ensures the ~/.airyra directory exists
func ensureAiryraDir() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}

	airyraDir := filepath.Join(homeDir, config.GlobalConfigDir)
	if err := os.MkdirAll(airyraDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create airyra directory: %w", err)
	}

	return airyraDir, nil
}
