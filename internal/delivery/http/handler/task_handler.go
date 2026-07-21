package handler

import (
	"net/http"
	"strconv"

	"github.com/go-playground/validator/v10"
	"github.com/ihsanuta/task-management-api/internal/delivery/http/dto"
	appmw "github.com/ihsanuta/task-management-api/internal/delivery/http/middleware"
	"github.com/ihsanuta/task-management-api/internal/usecase"
	"github.com/ihsanuta/task-management-api/pkg/apperror"
	"github.com/ihsanuta/task-management-api/pkg/response"
	"github.com/labstack/echo/v4"
)

type TaskHandler struct {
	uc        *usecase.TaskUsecase
	validator *validator.Validate
}

func NewTaskHandler(uc *usecase.TaskUsecase, validator *validator.Validate) *TaskHandler {
	return &TaskHandler{uc: uc, validator: validator}
}

// Create godoc
// @Summary Create task baru
// @Description Mendaftarkan task baru ke dalam sistem
// @Tags Task
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body dto.CreateTaskRequest true "Data Task"
// @Success 201 {object} response.Envelope "Task berhasil dibuat"
// @Failure 400 {object} response.Envelope "Payload tidak valid"
// @Router /task [post]
func (h *TaskHandler) Create(c echo.Context) error {
	var req dto.CreateTaskRequest
	if err := c.Bind(&req); err != nil {
		return response.Error(c, apperror.NewValidation("invalid JSON body"))
	}

	err := h.validator.Struct(req)
	if err != nil {
		return response.Error(c, apperror.NewValidationError(err))
	}

	idemKey := c.Request().Header.Get("Idempotency-Key")
	if idemKey == "" {
		return response.Error(c, apperror.ErrInvalidIdempotencyKey)
	}

	result, err := h.uc.CreateTask(c.Request().Context(), appmw.UserID(c), appmw.TeamID(c), req, idemKey)
	if err != nil {
		return response.Error(c, err)
	}

	return response.Success(c, result.Status, result.Task)
}

// List godoc
// @Summary List task
// @Description List task dalam sistem
// @Tags Task
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param search query string false "Filter task berdasarkan title"
// @Param status query string false "Filter task berdasarkan status" Enums(active, completed, pending)
// @Param limit query int false "Jumlah maksimal task yang dikembalikan" default(10)
// @Param page query int false "Nomor halaman" default(1)
// @Success 200 {object} response.Envelope "List task ditampilkan"
// @Failure 400 {object} response.Envelope "Payload tidak valid"
// @Router /task [get]
func (h *TaskHandler) List(c echo.Context) error {
	status := c.QueryParam("status")
	search := c.QueryParam("search")
	page, _ := strconv.Atoi(c.QueryParam("page"))
	limit, _ := strconv.Atoi(c.QueryParam("limit"))

	tasks, total, err := h.uc.ListTasks(c.Request().Context(), appmw.UserID(c), appmw.TeamID(c), status, search, page, limit)
	if err != nil {
		return response.Error(c, err)
	}
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 10
	}
	totalPages := int((total + int64(limit) - 1) / int64(limit))
	return response.SuccessWithMeta(c, http.StatusOK, tasks, response.Meta{
		Page: page, Limit: limit, TotalItems: total, TotalPages: totalPages,
	})
}

// Get godoc
// @Summary Detail task
// @Description Detail task dalam sistem
// @Tags Task
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "ID Task"
// @Success 200 {object} response.Envelope "Detail task ditampilkan"
// @Failure 400 {object} response.Envelope "Payload tidak valid"
// @Router /task/{id} [get]
func (h *TaskHandler) Get(c echo.Context) error {
	id := c.Param("id")
	task, err := h.uc.GetTask(c.Request().Context(), appmw.UserID(c), appmw.TeamID(c), id)
	if err != nil {
		return response.Error(c, err)
	}
	return response.Success(c, http.StatusOK, task)
}

// Update godoc
// @Summary Update task
// @Description Mengupdate task di dalam sistem
// @Tags Task
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "ID Task"
// @Param request body dto.UpdateTaskRequest true "Data Task"
// @Success 201 {object} response.Envelope "Task berhasil diupdate"
// @Failure 400 {object} response.Envelope "Payload tidak valid"
// @Router /task/{id} [put]
func (h *TaskHandler) Update(c echo.Context) error {
	id := c.Param("id")
	var req dto.UpdateTaskRequest
	if err := c.Bind(&req); err != nil {
		return response.Error(c, apperror.NewValidation("invalid JSON body"))
	}
	task, err := h.uc.UpdateTask(c.Request().Context(), appmw.UserID(c), appmw.TeamID(c), id, req)
	if err != nil {
		return response.Error(c, err)
	}
	return response.Success(c, http.StatusOK, task)
}

// Delete godoc
// @Summary Delete task
// @Description Menghapus task di dalam sistem
// @Tags Task
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "ID Task"
// @Success 201 {object} response.Envelope "Task berhasil dihapus"
// @Failure 400 {object} response.Envelope "Payload tidak valid"
// @Router /task/{id} [delete]
func (h *TaskHandler) Delete(c echo.Context) error {
	id := c.Param("id")
	if err := h.uc.DeleteTask(c.Request().Context(), appmw.UserID(c), appmw.TeamID(c), id); err != nil {
		return response.Error(c, err)
	}
	return response.Success(c, http.StatusOK, map[string]string{"message": "task deleted successfully"})
}

// Create godoc
// @Summary Create task baru
// @Description Mendaftarkan task baru ke dalam sistem
// @Tags Task
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "ID Task"
// @Param request body dto.AssignTaskRequest true "Data Task"
// @Success 201 {object} response.Envelope "Task berhasil diassign"
// @Failure 400 {object} response.Envelope "Payload tidak valid"
// @Router /task/{id}/assign [post]
func (h *TaskHandler) Assign(c echo.Context) error {
	id := c.Param("id")
	var req dto.AssignTaskRequest
	if err := c.Bind(&req); err != nil {
		return response.Error(c, apperror.NewValidation("invalid JSON body"))
	}
	err := h.validator.Struct(req)
	if err != nil {
		return response.Error(c, apperror.NewValidationError(err))
	}
	task, err := h.uc.AssignTask(c.Request().Context(), appmw.UserID(c), appmw.TeamID(c), id, req.AssigneeID)
	if err != nil {
		return response.Error(c, err)
	}
	return response.Success(c, http.StatusOK, task)
}
