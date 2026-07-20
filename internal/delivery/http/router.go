package http

import (
	"net/http"

	"github.com/ihsanuta/task-management-api/internal/delivery/http/handler"
	"github.com/ihsanuta/task-management-api/internal/delivery/http/middleware"
	"github.com/ihsanuta/task-management-api/pkg/jwtutil"
)

type Handlers struct {
	Auth *handler.AuthHandler
	Task *handler.TaskHandler
}

func NewRouter(h Handlers, jwtManager *jwtutil.Manager) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	})

	mux.HandleFunc("POST /auth/register", h.Auth.Register)
	mux.HandleFunc("POST /auth/login", h.Auth.Login)

	authMw := middleware.Auth(jwtManager)
	mux.Handle("POST /tasks", authMw(http.HandlerFunc(h.Task.Create)))
	mux.Handle("GET /tasks", authMw(http.HandlerFunc(h.Task.List)))
	mux.Handle("GET /tasks/{id}", authMw(http.HandlerFunc(h.Task.Get)))
	mux.Handle("PUT /tasks/{id}", authMw(http.HandlerFunc(h.Task.Update)))
	mux.Handle("DELETE /tasks/{id}", authMw(http.HandlerFunc(h.Task.Delete)))
	mux.Handle("POST /tasks/{id}/assign", authMw(http.HandlerFunc(h.Task.Assign)))

	var root http.Handler = mux
	root = middleware.Logger(root)
	root = middleware.RequestID(root)
	root = middleware.Recover(root)
	return root
}
