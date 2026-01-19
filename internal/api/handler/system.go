package handler

import (
	"net/http"

	"github.com/airyra/airyra/internal/api/response"
	"github.com/airyra/airyra/internal/domain"
	"github.com/airyra/airyra/internal/store"
)

// SystemHandler handles system-level operations.
type SystemHandler struct {
	manager *store.Manager
}

// NewSystemHandler creates a new SystemHandler.
func NewSystemHandler(manager *store.Manager) *SystemHandler {
	return &SystemHandler{manager: manager}
}

// Health handles GET /v1/health.
func (h *SystemHandler) Health(w http.ResponseWriter, r *http.Request) {
	response.OK(w, map[string]string{"status": "ok"})
}

// ListProjects handles GET /v1/projects.
func (h *SystemHandler) ListProjects(w http.ResponseWriter, r *http.Request) {
	projects, err := h.manager.ListProjects()
	if err != nil {
		response.Error(w, domain.NewInternalError(err))
		return
	}

	if projects == nil {
		projects = []string{}
	}

	response.OK(w, projects)
}
