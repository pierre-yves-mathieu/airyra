package middleware

import (
	"context"
	"database/sql"
	"net/http"
	"regexp"

	"github.com/go-chi/chi/v5"

	"github.com/airyra/airyra/internal/api/response"
	"github.com/airyra/airyra/internal/domain"
	"github.com/airyra/airyra/internal/store"
)

const (
	// ProjectKey is the context key for the project name.
	ProjectKey contextKey = "project"
	// DBKey is the context key for the database connection.
	DBKey contextKey = "db"
)

// Valid project name pattern: alphanumeric, hyphens, underscores, 1-64 chars.
var validProjectName = regexp.MustCompile(`^[a-zA-Z0-9_-]{1,64}$`)

// ProjectContext middleware validates the project name and injects the DB connection.
func ProjectContext(manager *store.Manager) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			project := chi.URLParam(r, "project")

			// Validate project name
			if !validProjectName.MatchString(project) {
				response.Error(w, domain.NewValidationError([]string{
					"Invalid project name. Must be 1-64 alphanumeric characters, hyphens, or underscores.",
				}))
				return
			}

			// Get or create database connection
			db, err := manager.GetDB(project)
			if err != nil {
				response.Error(w, domain.NewInternalError(err))
				return
			}

			// Add project and DB to context
			ctx := context.WithValue(r.Context(), ProjectKey, project)
			ctx = context.WithValue(ctx, DBKey, db)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// GetProject retrieves the project name from context.
func GetProject(ctx context.Context) string {
	if project, ok := ctx.Value(ProjectKey).(string); ok {
		return project
	}
	return ""
}

// GetDB retrieves the database connection from context.
func GetDB(ctx context.Context) *sql.DB {
	if db, ok := ctx.Value(DBKey).(*sql.DB); ok {
		return db
	}
	return nil
}
