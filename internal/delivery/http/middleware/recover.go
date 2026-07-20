package middleware

import (
	"log/slog"
	"net/http"
	"runtime/debug"

	"github.com/ihsanuta/task-management-api/pkg/apperror"
	"github.com/ihsanuta/task-management-api/pkg/response"
)

// Recover is the global panic handler. It ensures a panic anywhere in the
// handler chain becomes a clean structured 500 response instead of
// crashing the process or leaking a stack trace to the client. The stack
// trace itself is only ever written to the server-side log.
func Recover(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				slog.Error("panic_recovered",
					"request_id", GetRequestID(r.Context()),
					"path", r.URL.Path,
					"panic", rec,
					"stack", string(debug.Stack()),
				)
				response.Error(w, apperror.New(http.StatusInternalServerError, "INTERNAL_ERROR", "an unexpected error occurred"))
			}
		}()
		next.ServeHTTP(w, r)
	})
}
