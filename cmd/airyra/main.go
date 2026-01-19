package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/airyra/airyra/internal/api"
	"github.com/airyra/airyra/internal/store"
)

const (
	// DefaultAddr is the default server address.
	DefaultAddr = "localhost:7432"
	// DefaultDBPath is the default path for project databases.
	DefaultDBPath = ".airyra/projects"
)

func main() {
	// Determine database path
	homeDir, err := os.UserHomeDir()
	if err != nil {
		log.Fatalf("Failed to get home directory: %v", err)
	}
	dbPath := filepath.Join(homeDir, DefaultDBPath)

	// Create database manager
	manager, err := store.NewManager(dbPath)
	if err != nil {
		log.Fatalf("Failed to create database manager: %v", err)
	}
	defer manager.Close()

	// Create router
	router := api.NewRouter(manager)

	// Create server
	server := &http.Server{
		Addr:         DefaultAddr,
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start server in goroutine
	go func() {
		log.Printf("Starting airyra server on %s", DefaultAddr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server error: %v", err)
		}
	}()

	// Wait for shutdown signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")

	// Graceful shutdown with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Server stopped")
}
