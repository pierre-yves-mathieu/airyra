package response

import (
	"encoding/json"
	"net/http"

	"github.com/airyra/airyra/internal/domain"
)

// ErrorResponse represents an error response.
type ErrorResponse struct {
	Error ErrorBody `json:"error"`
}

// ErrorBody contains error details.
type ErrorBody struct {
	Code    string                 `json:"code"`
	Message string                 `json:"message"`
	Context map[string]interface{} `json:"context,omitempty"`
}

// PaginationMeta contains pagination metadata.
type PaginationMeta struct {
	Page       int `json:"page"`
	PerPage    int `json:"per_page"`
	Total      int `json:"total"`
	TotalPages int `json:"total_pages"`
}

// PaginatedResponse wraps data with pagination metadata.
type PaginatedResponse struct {
	Data       interface{}    `json:"data"`
	Pagination PaginationMeta `json:"pagination"`
}

// JSON sends a JSON response with the given status code.
func JSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

// Error sends an error response based on the domain error.
func Error(w http.ResponseWriter, err error) {
	domainErr, ok := err.(*domain.DomainError)
	if !ok {
		domainErr = domain.NewInternalError(err)
	}

	status := mapErrorCodeToStatus(domainErr.Code)
	JSON(w, status, ErrorResponse{
		Error: ErrorBody{
			Code:    string(domainErr.Code),
			Message: domainErr.Message,
			Context: domainErr.Context,
		},
	})
}

// Paginated sends a paginated JSON response.
func Paginated(w http.ResponseWriter, data interface{}, page, perPage, total int) {
	totalPages := total / perPage
	if total%perPage > 0 {
		totalPages++
	}

	JSON(w, http.StatusOK, PaginatedResponse{
		Data: data,
		Pagination: PaginationMeta{
			Page:       page,
			PerPage:    perPage,
			Total:      total,
			TotalPages: totalPages,
		},
	})
}

// Created sends a 201 Created response with JSON body.
func Created(w http.ResponseWriter, data interface{}) {
	JSON(w, http.StatusCreated, data)
}

// OK sends a 200 OK response with JSON body.
func OK(w http.ResponseWriter, data interface{}) {
	JSON(w, http.StatusOK, data)
}

// NoContent sends a 204 No Content response.
func NoContent(w http.ResponseWriter) {
	w.WriteHeader(http.StatusNoContent)
}

func mapErrorCodeToStatus(code domain.ErrorCode) int {
	switch code {
	case domain.ErrCodeTaskNotFound, domain.ErrCodeProjectNotFound, domain.ErrCodeDependencyNotFound,
		domain.ErrCodeSpecNotFound, domain.ErrCodeSpecDepNotFound:
		return http.StatusNotFound
	case domain.ErrCodeAlreadyClaimed, domain.ErrCodeSpecAlreadyCancelled:
		return http.StatusConflict
	case domain.ErrCodeNotOwner:
		return http.StatusForbidden
	case domain.ErrCodeInvalidTransition, domain.ErrCodeValidationFailed, domain.ErrCodeCycleDetected,
		domain.ErrCodeSpecNotCancelled:
		return http.StatusBadRequest
	case domain.ErrCodeInternalError:
		return http.StatusInternalServerError
	default:
		return http.StatusInternalServerError
	}
}
