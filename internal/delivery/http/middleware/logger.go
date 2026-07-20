package middleware

import (
	"log/slog"
	"net/http"
	"time"
)

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(status int) {
	r.status = status
	r.ResponseWriter.WriteHeader(status)
}

// Logger emits one structured (JSON) log line per request with the
// request id, method, path, status code, and latency. Log level is
// derived from the response status: INFO for 2xx/3xx, WARN for 4xx,
// ERROR for 5xx.
func Logger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}

		next.ServeHTTP(rec, r)

		latency := time.Since(start)
		attrs := []any{
			"request_id", GetRequestID(r.Context()),
			"method", r.Method,
			"path", r.URL.Path,
			"status", rec.status,
			"latency_ms", latency.Milliseconds(),
			"remote_addr", r.RemoteAddr,
		}

		switch {
		case rec.status >= 500:
			slog.Error("http_request", attrs...)
		case rec.status >= 400:
			slog.Warn("http_request", attrs...)
		default:
			slog.Info("http_request", attrs...)
		}
	})
}
