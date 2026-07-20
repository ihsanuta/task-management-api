package response

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/ihsanuta/task-management-api/pkg/apperror"
)

type Meta struct {
	Page       int   `json:"page,omitempty"`
	Limit      int   `json:"limit,omitempty"`
	TotalItems int64 `json:"total_items,omitempty"`
	TotalPages int   `json:"total_pages,omitempty"`
}

type Envelope struct {
	Status    string      `json:"status"`
	Code      string      `json:"code,omitempty"`
	Message   string      `json:"message,omitempty"`
	Data      interface{} `json:"data,omitempty"`
	Meta      *Meta       `json:"meta,omitempty"`
	Timestamp string      `json:"timestamp"`
}

func write(w http.ResponseWriter, httpStatus int, env Envelope) []byte {
	env.Timestamp = time.Now().UTC().Format(time.RFC3339Nano)
	body, err := json.Marshal(env)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"status":"error","code":"INTERNAL_ERROR","message":"failed to encode response"}`))
		return nil
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(httpStatus)
	_, _ = w.Write(body)
	return body
}
func Success(w http.ResponseWriter, httpStatus int, data interface{}) []byte {
	return write(w, httpStatus, Envelope{Status: "success", Data: data})
}

func SuccessWithMeta(w http.ResponseWriter, httpStatus int, data interface{}, meta Meta) []byte {
	return write(w, httpStatus, Envelope{Status: "success", Data: data, Meta: &meta})
}

func Error(w http.ResponseWriter, err error) []byte {
	appErr, ok := err.(*apperror.AppError)
	if !ok {
		appErr = apperror.NewInternal(err)
	}
	return write(w, appErr.HTTPStatus, Envelope{
		Status:  "error",
		Code:    appErr.Code,
		Message: appErr.Message,
	})
}
