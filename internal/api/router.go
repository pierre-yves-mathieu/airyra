package api

import (
	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"

	"github.com/airyra/airyra/internal/api/handler"
	"github.com/airyra/airyra/internal/api/middleware"
	"github.com/airyra/airyra/internal/store"
)

// NewRouter creates and configures the HTTP router.
func NewRouter(manager *store.Manager) *chi.Mux {
	r := chi.NewRouter()

	// Global middleware chain
	r.Use(middleware.Recovery)
	r.Use(middleware.Logging)
	r.Use(chimiddleware.RealIP)
	r.Use(middleware.AgentID)

	// Initialize handlers
	systemHandler := handler.NewSystemHandler(manager)
	taskHandler := handler.NewTaskHandler()
	transitionHandler := handler.NewTransitionHandler()
	dependencyHandler := handler.NewDependencyHandler()
	auditHandler := handler.NewAuditHandler()

	// System routes (no project context needed)
	r.Get("/v1/health", systemHandler.Health)
	r.Get("/v1/projects", systemHandler.ListProjects)

	// Project-scoped routes
	r.Route("/v1/projects/{project}", func(r chi.Router) {
		// Apply project context middleware
		r.Use(middleware.ProjectContext(manager))

		// Task CRUD
		r.Get("/tasks", taskHandler.ListTasks)
		r.Post("/tasks", taskHandler.CreateTask)
		r.Get("/tasks/ready", taskHandler.ListReadyTasks)
		r.Get("/tasks/{id}", taskHandler.GetTask)
		r.Patch("/tasks/{id}", taskHandler.UpdateTask)
		r.Delete("/tasks/{id}", taskHandler.DeleteTask)

		// Status transitions
		r.Post("/tasks/{id}/claim", transitionHandler.ClaimTask)
		r.Post("/tasks/{id}/done", transitionHandler.CompleteTask)
		r.Post("/tasks/{id}/release", transitionHandler.ReleaseTask)
		r.Post("/tasks/{id}/block", transitionHandler.BlockTask)
		r.Post("/tasks/{id}/unblock", transitionHandler.UnblockTask)

		// Dependencies
		r.Get("/tasks/{id}/deps", dependencyHandler.ListDependencies)
		r.Post("/tasks/{id}/deps", dependencyHandler.AddDependency)
		r.Delete("/tasks/{id}/deps/{depID}", dependencyHandler.RemoveDependency)

		// Audit
		r.Get("/tasks/{id}/history", auditHandler.GetTaskHistory)
		r.Get("/audit", auditHandler.QueryAuditLog)
	})

	return r
}
