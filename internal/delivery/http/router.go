package http

import (
	"net/http"

	"github.com/ihsanuta/task-management-api/internal/delivery/http/handler"
	authmiddleware "github.com/ihsanuta/task-management-api/internal/delivery/http/middleware"
	"github.com/ihsanuta/task-management-api/pkg/jwtutil"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

type Handlers struct {
	Auth *handler.AuthHandler
	Task *handler.TaskHandler
}

func NewRouter(h Handlers, jwtManager *jwtutil.Manager) *echo.Echo {
	e := echo.New()

	e.Use(authmiddleware.Logger())
	e.Use(middleware.RequestID())
	e.Use(middleware.Recover())

	e.GET("/healthz", func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]string{
			"status": "ok",
		})
	})

	auth := e.Group("/auth")
	auth.POST("/register", h.Auth.Register)
	auth.POST("/login", h.Auth.Login)

	// Protected Routes (Tasks)
	tasks := e.Group("/tasks")
	tasks.Use(authmiddleware.Auth(jwtManager))

	tasks.POST("", h.Task.Create)
	tasks.GET("", h.Task.List)
	tasks.GET("/:id", h.Task.Get)
	tasks.PUT("/:id", h.Task.Update)
	tasks.DELETE("/:id", h.Task.Delete)
	tasks.POST("/:id/assign", h.Task.Assign)

	return e
}
