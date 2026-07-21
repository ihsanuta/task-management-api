package middleware

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
)

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(status int) {
	r.status = status
	r.ResponseWriter.WriteHeader(status)
}

func Logger() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			start := time.Now()
			err := next(c)

			if err != nil {
				c.Error(err)
			}

			req := c.Request()
			res := c.Response()
			latency := time.Since(start)

			attrs := []any{
				"request_id", GetRequestID(req.Context()),
				"method", req.Method,
				"path", req.URL.Path,
				"status", res.Status,
				"latency_ms", latency.Milliseconds(),
				"remote_addr", req.RemoteAddr,
			}

			switch {
			case res.Status >= 500:
				slog.Error("http_request", attrs...)
			case res.Status >= 400:
				slog.Warn("http_request", attrs...)
			default:
				slog.Info("http_request", attrs...)
			}

			return nil
		}
	}
}
