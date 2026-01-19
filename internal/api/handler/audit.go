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

// AuditHandler handles audit log operations.
type AuditHandler struct{}

// NewAuditHandler creates a new AuditHandler.
func NewAuditHandler() *AuditHandler {
	return &AuditHandler{}
}

// GetTaskHistory handles GET /tasks/{id}/history.
func (h *AuditHandler) GetTaskHistory(w http.ResponseWriter, r *http.Request) {
	taskID := chi.URLParam(r, "id")

	db := middleware.GetDB(r.Context())
	taskRepo := sqlite.NewTaskRepository(db)
	auditRepo := sqlite.NewAuditRepository(db)
	svc := service.NewAuditService(auditRepo, taskRepo)

	entries, err := svc.GetTaskHistory(taskID)
	if err != nil {
		response.Error(w, err)
		return
	}

	if entries == nil {
		entries = []*domain.AuditEntry{}
	}

	response.OK(w, entries)
}

// QueryAuditLog handles GET /audit.
func (h *AuditHandler) QueryAuditLog(w http.ResponseWriter, r *http.Request) {
	pagination := request.ParsePagination(r)
	queryParams := request.ParseAuditQuery(r)

	db := middleware.GetDB(r.Context())
	taskRepo := sqlite.NewTaskRepository(db)
	auditRepo := sqlite.NewAuditRepository(db)
	svc := service.NewAuditService(auditRepo, taskRepo)

	entries, total, err := svc.Query(service.QueryInput{
		Action:    queryParams.Action,
		AgentID:   queryParams.AgentID,
		StartTime: queryParams.StartTime,
		EndTime:   queryParams.EndTime,
		Page:      pagination.Page,
		PerPage:   pagination.PerPage,
	})
	if err != nil {
		response.Error(w, err)
		return
	}

	if entries == nil {
		entries = []*domain.AuditEntry{}
	}

	response.Paginated(w, entries, pagination.Page, pagination.PerPage, total)
}
