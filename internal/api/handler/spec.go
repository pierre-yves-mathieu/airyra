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

// SpecHandler handles spec CRUD operations.
type SpecHandler struct{}

// NewSpecHandler creates a new SpecHandler.
func NewSpecHandler() *SpecHandler {
	return &SpecHandler{}
}

// CreateSpec handles POST /specs.
func (h *SpecHandler) CreateSpec(w http.ResponseWriter, r *http.Request) {
	var req request.CreateSpecRequest
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

	specRepo := sqlite.NewSpecRepository(db)
	auditRepo := sqlite.NewAuditRepository(db)
	svc := service.NewSpecService(specRepo, auditRepo)

	spec, err := svc.Create(service.CreateSpecInput{
		Title:       req.Title,
		Description: req.Description,
	}, agentID)
	if err != nil {
		response.Error(w, err)
		return
	}

	response.Created(w, specWithStatus(spec))
}

// GetSpec handles GET /specs/{id}.
func (h *SpecHandler) GetSpec(w http.ResponseWriter, r *http.Request) {
	specID := chi.URLParam(r, "id")

	db := middleware.GetDB(r.Context())
	specRepo := sqlite.NewSpecRepository(db)
	auditRepo := sqlite.NewAuditRepository(db)
	svc := service.NewSpecService(specRepo, auditRepo)

	spec, err := svc.Get(specID)
	if err != nil {
		response.Error(w, err)
		return
	}

	response.OK(w, specWithStatus(spec))
}

// ListSpecs handles GET /specs.
func (h *SpecHandler) ListSpecs(w http.ResponseWriter, r *http.Request) {
	pagination := request.ParsePagination(r)
	status := request.ParseSpecStatus(r)

	db := middleware.GetDB(r.Context())
	specRepo := sqlite.NewSpecRepository(db)
	auditRepo := sqlite.NewAuditRepository(db)
	svc := service.NewSpecService(specRepo, auditRepo)

	specs, total, err := svc.List(service.ListSpecsInput{
		Status:  status,
		Page:    pagination.Page,
		PerPage: pagination.PerPage,
	})
	if err != nil {
		response.Error(w, err)
		return
	}

	specsWithStatus := make([]SpecResponse, 0, len(specs))
	for _, spec := range specs {
		specsWithStatus = append(specsWithStatus, specWithStatus(spec))
	}

	response.Paginated(w, specsWithStatus, pagination.Page, pagination.PerPage, total)
}

// ListReadySpecs handles GET /specs/ready.
func (h *SpecHandler) ListReadySpecs(w http.ResponseWriter, r *http.Request) {
	pagination := request.ParsePagination(r)

	db := middleware.GetDB(r.Context())
	specRepo := sqlite.NewSpecRepository(db)
	auditRepo := sqlite.NewAuditRepository(db)
	svc := service.NewSpecService(specRepo, auditRepo)

	specs, total, err := svc.ListReady(pagination.Page, pagination.PerPage)
	if err != nil {
		response.Error(w, err)
		return
	}

	specsWithStatus := make([]SpecResponse, 0, len(specs))
	for _, spec := range specs {
		specsWithStatus = append(specsWithStatus, specWithStatus(spec))
	}

	response.Paginated(w, specsWithStatus, pagination.Page, pagination.PerPage, total)
}

// UpdateSpec handles PATCH /specs/{id}.
func (h *SpecHandler) UpdateSpec(w http.ResponseWriter, r *http.Request) {
	specID := chi.URLParam(r, "id")

	var req request.UpdateSpecRequest
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

	specRepo := sqlite.NewSpecRepository(db)
	auditRepo := sqlite.NewAuditRepository(db)
	svc := service.NewSpecService(specRepo, auditRepo)

	spec, err := svc.Update(specID, service.UpdateSpecInput{
		Title:       req.Title,
		Description: req.Description,
	}, agentID)
	if err != nil {
		response.Error(w, err)
		return
	}

	response.OK(w, specWithStatus(spec))
}

// DeleteSpec handles DELETE /specs/{id}.
func (h *SpecHandler) DeleteSpec(w http.ResponseWriter, r *http.Request) {
	specID := chi.URLParam(r, "id")

	db := middleware.GetDB(r.Context())
	agentID := middleware.GetAgentID(r.Context())

	specRepo := sqlite.NewSpecRepository(db)
	auditRepo := sqlite.NewAuditRepository(db)
	svc := service.NewSpecService(specRepo, auditRepo)

	if err := svc.Delete(specID, agentID); err != nil {
		response.Error(w, err)
		return
	}

	response.NoContent(w)
}

// CancelSpec handles POST /specs/{id}/cancel.
func (h *SpecHandler) CancelSpec(w http.ResponseWriter, r *http.Request) {
	specID := chi.URLParam(r, "id")

	db := middleware.GetDB(r.Context())
	agentID := middleware.GetAgentID(r.Context())

	specRepo := sqlite.NewSpecRepository(db)
	auditRepo := sqlite.NewAuditRepository(db)
	svc := service.NewSpecService(specRepo, auditRepo)

	spec, err := svc.Cancel(specID, agentID)
	if err != nil {
		response.Error(w, err)
		return
	}

	response.OK(w, specWithStatus(spec))
}

// ReopenSpec handles POST /specs/{id}/reopen.
func (h *SpecHandler) ReopenSpec(w http.ResponseWriter, r *http.Request) {
	specID := chi.URLParam(r, "id")

	db := middleware.GetDB(r.Context())
	agentID := middleware.GetAgentID(r.Context())

	specRepo := sqlite.NewSpecRepository(db)
	auditRepo := sqlite.NewAuditRepository(db)
	svc := service.NewSpecService(specRepo, auditRepo)

	spec, err := svc.Reopen(specID, agentID)
	if err != nil {
		response.Error(w, err)
		return
	}

	response.OK(w, specWithStatus(spec))
}

// ListSpecTasks handles GET /specs/{id}/tasks.
func (h *SpecHandler) ListSpecTasks(w http.ResponseWriter, r *http.Request) {
	specID := chi.URLParam(r, "id")
	pagination := request.ParsePagination(r)

	db := middleware.GetDB(r.Context())
	specRepo := sqlite.NewSpecRepository(db)
	auditRepo := sqlite.NewAuditRepository(db)
	svc := service.NewSpecService(specRepo, auditRepo)

	tasks, total, err := svc.ListTasks(specID, pagination.Page, pagination.PerPage)
	if err != nil {
		response.Error(w, err)
		return
	}

	if tasks == nil {
		tasks = []*domain.Task{}
	}

	response.Paginated(w, tasks, pagination.Page, pagination.PerPage, total)
}

// ListSpecDependencies handles GET /specs/{id}/deps.
func (h *SpecHandler) ListSpecDependencies(w http.ResponseWriter, r *http.Request) {
	specID := chi.URLParam(r, "id")

	db := middleware.GetDB(r.Context())
	specRepo := sqlite.NewSpecRepository(db)
	specDepRepo := sqlite.NewSpecDependencyRepository(db)
	auditRepo := sqlite.NewAuditRepository(db)
	svc := service.NewSpecDependencyService(specDepRepo, specRepo, auditRepo)

	deps, err := svc.List(specID)
	if err != nil {
		response.Error(w, err)
		return
	}

	if deps == nil {
		deps = []*domain.SpecDependency{}
	}

	response.OK(w, deps)
}

// AddSpecDependency handles POST /specs/{id}/deps.
func (h *SpecHandler) AddSpecDependency(w http.ResponseWriter, r *http.Request) {
	childID := chi.URLParam(r, "id")

	var req request.AddSpecDependencyRequest
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

	specRepo := sqlite.NewSpecRepository(db)
	specDepRepo := sqlite.NewSpecDependencyRepository(db)
	auditRepo := sqlite.NewAuditRepository(db)
	svc := service.NewSpecDependencyService(specDepRepo, specRepo, auditRepo)

	if err := svc.Add(childID, req.ParentID, agentID); err != nil {
		response.Error(w, err)
		return
	}

	response.NoContent(w)
}

// RemoveSpecDependency handles DELETE /specs/{id}/deps/{parentID}.
func (h *SpecHandler) RemoveSpecDependency(w http.ResponseWriter, r *http.Request) {
	childID := chi.URLParam(r, "id")
	parentID := chi.URLParam(r, "parentID")

	db := middleware.GetDB(r.Context())
	agentID := middleware.GetAgentID(r.Context())

	specRepo := sqlite.NewSpecRepository(db)
	specDepRepo := sqlite.NewSpecDependencyRepository(db)
	auditRepo := sqlite.NewAuditRepository(db)
	svc := service.NewSpecDependencyService(specDepRepo, specRepo, auditRepo)

	if err := svc.Remove(childID, parentID, agentID); err != nil {
		response.Error(w, err)
		return
	}

	response.NoContent(w)
}

// SpecResponse is the API response for a spec including computed status.
type SpecResponse struct {
	ID          string  `json:"id"`
	Title       string  `json:"title"`
	Description *string `json:"description,omitempty"`
	Status      string  `json:"status"`
	TaskCount   int     `json:"task_count"`
	DoneCount   int     `json:"done_count"`
	CreatedAt   string  `json:"created_at"`
	UpdatedAt   string  `json:"updated_at"`
}

func specWithStatus(spec *domain.Spec) SpecResponse {
	return SpecResponse{
		ID:          spec.ID,
		Title:       spec.Title,
		Description: spec.Description,
		Status:      string(spec.ComputeStatus()),
		TaskCount:   spec.TaskCount,
		DoneCount:   spec.DoneCount,
		CreatedAt:   spec.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt:   spec.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
}
