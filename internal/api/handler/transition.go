package handler

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/airyra/airyra/internal/api/middleware"
	"github.com/airyra/airyra/internal/api/response"
	"github.com/airyra/airyra/internal/service"
	"github.com/airyra/airyra/internal/store/sqlite"
)

// TransitionHandler handles task status transitions.
type TransitionHandler struct{}

// NewTransitionHandler creates a new TransitionHandler.
func NewTransitionHandler() *TransitionHandler {
	return &TransitionHandler{}
}

// ClaimTask handles POST /tasks/{id}/claim.
func (h *TransitionHandler) ClaimTask(w http.ResponseWriter, r *http.Request) {
	taskID := chi.URLParam(r, "id")

	db := middleware.GetDB(r.Context())
	agentID := middleware.GetAgentID(r.Context())

	taskRepo := sqlite.NewTaskRepository(db)
	auditRepo := sqlite.NewAuditRepository(db)
	svc := service.NewTransitionService(taskRepo, auditRepo)

	task, err := svc.Claim(taskID, agentID)
	if err != nil {
		response.Error(w, err)
		return
	}

	response.OK(w, task)
}

// CompleteTask handles POST /tasks/{id}/done.
func (h *TransitionHandler) CompleteTask(w http.ResponseWriter, r *http.Request) {
	taskID := chi.URLParam(r, "id")

	db := middleware.GetDB(r.Context())
	agentID := middleware.GetAgentID(r.Context())

	taskRepo := sqlite.NewTaskRepository(db)
	auditRepo := sqlite.NewAuditRepository(db)
	svc := service.NewTransitionService(taskRepo, auditRepo)

	task, err := svc.Complete(taskID, agentID)
	if err != nil {
		response.Error(w, err)
		return
	}

	response.OK(w, task)
}

// ReleaseTask handles POST /tasks/{id}/release.
func (h *TransitionHandler) ReleaseTask(w http.ResponseWriter, r *http.Request) {
	taskID := chi.URLParam(r, "id")

	db := middleware.GetDB(r.Context())
	agentID := middleware.GetAgentID(r.Context())

	// Check for force parameter
	force := r.URL.Query().Get("force") == "true"

	taskRepo := sqlite.NewTaskRepository(db)
	auditRepo := sqlite.NewAuditRepository(db)
	svc := service.NewTransitionService(taskRepo, auditRepo)

	task, err := svc.Release(taskID, agentID, force)
	if err != nil {
		response.Error(w, err)
		return
	}

	response.OK(w, task)
}

// BlockTask handles POST /tasks/{id}/block.
func (h *TransitionHandler) BlockTask(w http.ResponseWriter, r *http.Request) {
	taskID := chi.URLParam(r, "id")

	db := middleware.GetDB(r.Context())
	agentID := middleware.GetAgentID(r.Context())

	taskRepo := sqlite.NewTaskRepository(db)
	auditRepo := sqlite.NewAuditRepository(db)
	svc := service.NewTransitionService(taskRepo, auditRepo)

	task, err := svc.Block(taskID, agentID)
	if err != nil {
		response.Error(w, err)
		return
	}

	response.OK(w, task)
}

// UnblockTask handles POST /tasks/{id}/unblock.
func (h *TransitionHandler) UnblockTask(w http.ResponseWriter, r *http.Request) {
	taskID := chi.URLParam(r, "id")

	db := middleware.GetDB(r.Context())
	agentID := middleware.GetAgentID(r.Context())

	taskRepo := sqlite.NewTaskRepository(db)
	auditRepo := sqlite.NewAuditRepository(db)
	svc := service.NewTransitionService(taskRepo, auditRepo)

	task, err := svc.Unblock(taskID, agentID)
	if err != nil {
		response.Error(w, err)
		return
	}

	response.OK(w, task)
}
