package middleware

import (
	"log/slog"
	"net/http"
	"runtime/debug"

	"github.com/ihsanuta/task-management-api/pkg/apperror"
	"github.com/labstack/echo/v4"
)

func Recover() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			defer func() {
				if rec := recover(); rec != nil {
					path := c.Request().URL.Path

					reqCtx := c.Request().Context()

					slog.Error("panic_recovered",
						"request_id", GetRequestID(reqCtx),
						"path", path,
						"panic", rec,
						"stack", string(debug.Stack()),
					)

					errResp := apperror.New(http.StatusInternalServerError, "INTERNAL_ERROR", "an unexpected error occurred")

					c.JSON(http.StatusInternalServerError, errResp)
				}
			}()

			// Lanjut ke handler berikutnya
			return next(c)
		}
	}
}
