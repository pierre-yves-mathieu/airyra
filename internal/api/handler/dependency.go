package handler

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/airyra/airyra/internal/api/middleware"
	"github.com/airyra/airyra/internal/api/request"
	"github.com/airyra/airyra/internal/api/response"
	"github.com/airyra/airyra/internal/domain"
	"github.com/airyra/airyra/internal/service"
	"github.com/airyra/airyra/internal/store/sqlite"
)

// DependencyHandler handles dependency operations.
type DependencyHandler struct{}

// NewDependencyHandler creates a new DependencyHandler.
func NewDependencyHandler() *DependencyHandler {
	return &DependencyHandler{}
}

// ListDependencies handles GET /tasks/{id}/deps.
func (h *DependencyHandler) ListDependencies(w http.ResponseWriter, r *http.Request) {
	taskID := chi.URLParam(r, "id")

	db := middleware.GetDB(r.Context())
	taskRepo := sqlite.NewTaskRepository(db)
	depRepo := sqlite.NewDependencyRepository(db)
	auditRepo := sqlite.NewAuditRepository(db)
	svc := service.NewDependencyService(depRepo, taskRepo, auditRepo)

	deps, err := svc.List(taskID)
	if err != nil {
		response.Error(w, err)
		return
	}

	if deps == nil {
		deps = []*domain.Dependency{}
	}

	response.OK(w, deps)
}

// AddDependency handles POST /tasks/{id}/deps.
func (h *DependencyHandler) AddDependency(w http.ResponseWriter, r *http.Request) {
	taskID := chi.URLParam(r, "id")

	var req request.AddDependencyRequest
	if err := request.DecodeJSON(r, &req); err != nil {
		response.Error(w, domain.NewValidationError([]string{"Invalid JSON body"}))
		return
	}

	if errors := req.Validate(); len(errors) > 0 {
		response.Error(w, domain.NewValidationError(errors))
		return
	}

	db := middleware.GetDB(r.Context())
	agentID := middleware.GetAgentID(r.Context())

	taskRepo := sqlite.NewTaskRepository(db)
	depRepo := sqlite.NewDependencyRepository(db)
	auditRepo := sqlite.NewAuditRepository(db)
	svc := service.NewDependencyService(depRepo, taskRepo, auditRepo)

	if err := svc.Add(taskID, req.ParentID, agentID); err != nil {
		response.Error(w, err)
		return
	}

	response.Created(w, map[string]string{
		"child_id":  taskID,
		"parent_id": req.ParentID,
	})
}

// RemoveDependency handles DELETE /tasks/{id}/deps/{depID}.
func (h *DependencyHandler) RemoveDependency(w http.ResponseWriter, r *http.Request) {
	taskID := chi.URLParam(r, "id")
	depID := chi.URLParam(r, "depID")

	db := middleware.GetDB(r.Context())
	agentID := middleware.GetAgentID(r.Context())

	taskRepo := sqlite.NewTaskRepository(db)
	depRepo := sqlite.NewDependencyRepository(db)
	auditRepo := sqlite.NewAuditRepository(db)
	svc := service.NewDependencyService(depRepo, taskRepo, auditRepo)

	if err := svc.Remove(taskID, depID, agentID); err != nil {
		response.Error(w, err)
		return
	}

	response.NoContent(w)
}
