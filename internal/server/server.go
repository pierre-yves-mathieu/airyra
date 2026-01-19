// Package server provides the HTTP server lifecycle management for Airyra.
package server

import (
	"context"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/airyra/airyra/internal/api"
	"github.com/airyra/airyra/internal/store"
)

const (
	// DefaultAddress is the default address the server listens on.
	DefaultAddress = "localhost:7432"
	// DefaultShutdownTimeout is the default timeout for graceful shutdown.
	DefaultShutdownTimeout = 30 * time.Second
)

// Server manages the HTTP server lifecycle.
type Server struct {
	httpServer *http.Server
	manager    *store.Manager
	logger     *log.Logger
	listener   net.Listener
	addr       string
	mu         sync.Mutex
	started    bool
}

// New creates a new Server instance.
// If addr is empty, DefaultAddress ("localhost:7432") will be used.
func New(addr string, manager *store.Manager) *Server {
	if addr == "" {
		addr = DefaultAddress
	}

	router := api.NewRouter(manager)

	return &Server{
		httpServer: &http.Server{
			Addr:         addr,
			Handler:      router,
			ReadTimeout:  15 * time.Second,
			WriteTimeout: 15 * time.Second,
			IdleTimeout:  60 * time.Second,
		},
		manager: manager,
		logger:  log.New(os.Stdout, "[airyra] ", log.LstdFlags),
		addr:    addr,
	}
}

// Start starts the HTTP server and blocks until the server is shut down.
// It returns http.ErrServerClosed when the server is gracefully shut down.
func (s *Server) Start() error {
	s.mu.Lock()
	if s.started {
		s.mu.Unlock()
		return nil
	}

	// Create listener first so we know the actual address (for port 0 case)
	ln, err := net.Listen("tcp", s.httpServer.Addr)
	if err != nil {
		s.mu.Unlock()
		return err
	}

	s.listener = ln
	s.started = true
	s.mu.Unlock()

	s.logger.Printf("Server listening on %s", ln.Addr().String())

	return s.httpServer.Serve(ln)
}

// Shutdown gracefully shuts down the server without interrupting active connections.
// It waits for active connections to finish or until the context is canceled.
func (s *Server) Shutdown(ctx context.Context) error {
	s.mu.Lock()
	if !s.started {
		s.mu.Unlock()
		return nil
	}
	s.mu.Unlock()

	s.logger.Println("Shutting down server...")

	if err := s.httpServer.Shutdown(ctx); err != nil {
		return err
	}

	// Close the database manager
	if err := s.manager.Close(); err != nil {
		s.logger.Printf("Warning: error closing database manager: %v", err)
	}

	s.logger.Println("Server stopped")
	return nil
}

// Addr returns the address the server is listening on.
// Returns empty string if the server hasn't started yet.
func (s *Server) Addr() string {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.listener != nil {
		return s.listener.Addr().String()
	}
	return ""
}

// DefaultAddr returns the default address the server would use.
func (s *Server) DefaultAddr() string {
	return DefaultAddress
}

// ListenAndServe starts the server with signal handling for graceful shutdown.
// It handles SIGINT and SIGTERM signals.
// This is a convenience method that combines Start and signal handling.
func (s *Server) ListenAndServe() error {
	// Set up signal handling
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Start server in goroutine
	errChan := make(chan error, 1)
	go func() {
		errChan <- s.Start()
	}()

	// Wait for signal or error
	select {
	case err := <-errChan:
		return err
	case sig := <-sigChan:
		s.logger.Printf("Received signal: %v", sig)
	}

	// Graceful shutdown with timeout
	ctx, cancel := context.WithTimeout(context.Background(), DefaultShutdownTimeout)
	defer cancel()

	return s.Shutdown(ctx)
}

// Run is an alias for ListenAndServe for convenience.
func (s *Server) Run() error {
	return s.ListenAndServe()
}
