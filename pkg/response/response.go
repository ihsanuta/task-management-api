package response

import (
	"time"

	"github.com/ihsanuta/task-management-api/pkg/apperror"
	"github.com/labstack/echo/v4"
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

func write(c echo.Context, httpStatus int, env Envelope) error {
	env.Timestamp = time.Now().UTC().Format(time.RFC3339Nano)
	return c.JSON(httpStatus, env)
}
func Success(c echo.Context, httpStatus int, data interface{}) error {
	return write(c, httpStatus, Envelope{Status: "success", Data: data})
}

func SuccessWithMeta(c echo.Context, httpStatus int, data interface{}, meta Meta) error {
	return write(c, httpStatus, Envelope{Status: "success", Data: data, Meta: &meta})
}

func Error(c echo.Context, err error) error {
	appErr, ok := err.(*apperror.AppError)
	if !ok {
		appErr = apperror.NewInternal(err)
	}
	return write(c, appErr.HTTPStatus, Envelope{
		Status:  "error",
		Code:    appErr.Code,
		Message: appErr.Message,
	})
}
