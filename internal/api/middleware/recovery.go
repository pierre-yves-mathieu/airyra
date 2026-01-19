package middleware

import (
	"log"
	"net/http"
	"runtime/debug"

	"github.com/airyra/airyra/internal/api/response"
	"github.com/airyra/airyra/internal/domain"
)

// Recovery middleware catches panics and returns a 500 error.
func Recovery(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				log.Printf("panic recovered: %v\n%s", err, debug.Stack())
				response.Error(w, domain.NewInternalError(nil))
			}
		}()
		next.ServeHTTP(w, r)
	})
}
