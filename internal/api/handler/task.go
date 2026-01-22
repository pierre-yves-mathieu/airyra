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

// TaskHandler handles task CRUD operations.
type TaskHandler struct{}

// NewTaskHandler creates a new TaskHandler.
func NewTaskHandler() *TaskHandler {
	return &TaskHandler{}
}

// CreateTask handles POST /tasks.
func (h *TaskHandler) CreateTask(w http.ResponseWriter, r *http.Request) {
	var req request.CreateTaskRequest
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
	auditRepo := sqlite.NewAuditRepository(db)
	svc := service.NewTaskService(taskRepo, auditRepo)

	task, err := svc.Create(service.CreateTaskInput{
		Title:       req.Title,
		Description: req.Description,
		Priority:    req.Priority,
		ParentID:    req.ParentID,
		SpecID:      req.SpecID,
	}, agentID)
	if err != nil {
		response.Error(w, err)
		return
	}

	response.Created(w, task)
}

// GetTask handles GET /tasks/{id}.
func (h *TaskHandler) GetTask(w http.ResponseWriter, r *http.Request) {
	taskID := chi.URLParam(r, "id")

	db := middleware.GetDB(r.Context())
	taskRepo := sqlite.NewTaskRepository(db)
	auditRepo := sqlite.NewAuditRepository(db)
	svc := service.NewTaskService(taskRepo, auditRepo)

	task, err := svc.Get(taskID)
	if err != nil {
		response.Error(w, err)
		return
	}

	response.OK(w, task)
}

// ListTasks handles GET /tasks.
func (h *TaskHandler) ListTasks(w http.ResponseWriter, r *http.Request) {
	pagination := request.ParsePagination(r)
	status := request.ParseStatus(r)

	db := middleware.GetDB(r.Context())
	taskRepo := sqlite.NewTaskRepository(db)
	auditRepo := sqlite.NewAuditRepository(db)
	svc := service.NewTaskService(taskRepo, auditRepo)

	tasks, total, err := svc.List(service.ListTasksInput{
		Status:  status,
		Page:    pagination.Page,
		PerPage: pagination.PerPage,
	})
	if err != nil {
		response.Error(w, err)
		return
	}

	if tasks == nil {
		tasks = []*domain.Task{}
	}

	response.Paginated(w, tasks, pagination.Page, pagination.PerPage, total)
}

// ListReadyTasks handles GET /tasks/ready.
func (h *TaskHandler) ListReadyTasks(w http.ResponseWriter, r *http.Request) {
	pagination := request.ParsePagination(r)

	db := middleware.GetDB(r.Context())
	taskRepo := sqlite.NewTaskRepository(db)
	auditRepo := sqlite.NewAuditRepository(db)
	svc := service.NewTaskService(taskRepo, auditRepo)

	tasks, total, err := svc.ListReady(pagination.Page, pagination.PerPage)
	if err != nil {
		response.Error(w, err)
		return
	}

	if tasks == nil {
		tasks = []*domain.Task{}
	}

	response.Paginated(w, tasks, pagination.Page, pagination.PerPage, total)
}

// UpdateTask handles PATCH /tasks/{id}.
func (h *TaskHandler) UpdateTask(w http.ResponseWriter, r *http.Request) {
	taskID := chi.URLParam(r, "id")

	var req request.UpdateTaskRequest
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
	auditRepo := sqlite.NewAuditRepository(db)
	svc := service.NewTaskService(taskRepo, auditRepo)

	task, err := svc.Update(taskID, service.UpdateTaskInput{
		Title:       req.Title,
		Description: req.Description,
		Priority:    req.Priority,
		ParentID:    req.ParentID,
	}, agentID)
	if err != nil {
		response.Error(w, err)
		return
	}

	response.OK(w, task)
}

// DeleteTask handles DELETE /tasks/{id}.
func (h *TaskHandler) DeleteTask(w http.ResponseWriter, r *http.Request) {
	taskID := chi.URLParam(r, "id")

	db := middleware.GetDB(r.Context())
	agentID := middleware.GetAgentID(r.Context())

	taskRepo := sqlite.NewTaskRepository(db)
	auditRepo := sqlite.NewAuditRepository(db)
	svc := service.NewTaskService(taskRepo, auditRepo)

	if err := svc.Delete(taskID, agentID); err != nil {
		response.Error(w, err)
		return
	}

	response.NoContent(w)
}
