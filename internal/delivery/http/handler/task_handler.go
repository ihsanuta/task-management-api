package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-playground/validator/v10"
	"github.com/ihsanuta/task-management-api/internal/delivery/http/dto"
	appmw "github.com/ihsanuta/task-management-api/internal/delivery/http/middleware"
	"github.com/ihsanuta/task-management-api/internal/usecase"
	"github.com/ihsanuta/task-management-api/pkg/apperror"
	"github.com/ihsanuta/task-management-api/pkg/response"
)

type TaskHandler struct {
	uc        *usecase.TaskUsecase
	validator *validator.Validate
}

func NewTaskHandler(uc *usecase.TaskUsecase, validator *validator.Validate) *TaskHandler {
	return &TaskHandler{uc: uc, validator: validator}
}

func (h *TaskHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req dto.CreateTaskRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, apperror.NewValidation("invalid JSON body"))
		return
	}

	err := h.validator.Struct(req)
	if err != nil {
		response.Error(w, apperror.NewValidationError(err))
		return
	}

	idemKey := r.Header.Get("Idempotency-Key")
	if idemKey == "" {
		response.Error(w, apperror.ErrInvalidIdempotencyKey)
		return
	}

	result, err := h.uc.CreateTask(r.Context(), appmw.UserID(r.Context()), appmw.TeamID(r.Context()), req, idemKey)
	if err != nil {
		response.Error(w, err)
		return
	}

	response.Success(w, result.Status, result.Task)
}

func (h *TaskHandler) List(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	status := q.Get("status")
	search := q.Get("search")
	page, _ := strconv.Atoi(q.Get("page"))
	limit, _ := strconv.Atoi(q.Get("limit"))

	tasks, total, err := h.uc.ListTasks(r.Context(), appmw.UserID(r.Context()), appmw.TeamID(r.Context()), status, search, page, limit)
	if err != nil {
		response.Error(w, err)
		return
	}
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 10
	}
	totalPages := int((total + int64(limit) - 1) / int64(limit))
	response.SuccessWithMeta(w, http.StatusOK, tasks, response.Meta{
		Page: page, Limit: limit, TotalItems: total, TotalPages: totalPages,
	})
}

func (h *TaskHandler) Get(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	task, err := h.uc.GetTask(r.Context(), appmw.UserID(r.Context()), appmw.TeamID(r.Context()), id)
	if err != nil {
		response.Error(w, err)
		return
	}
	response.Success(w, http.StatusOK, task)
}

func (h *TaskHandler) Update(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var req dto.UpdateTaskRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, apperror.NewValidation("invalid JSON body"))
		return
	}
	task, err := h.uc.UpdateTask(r.Context(), appmw.UserID(r.Context()), appmw.TeamID(r.Context()), id, req)
	if err != nil {
		response.Error(w, err)
		return
	}
	response.Success(w, http.StatusOK, task)
}

func (h *TaskHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if err := h.uc.DeleteTask(r.Context(), appmw.UserID(r.Context()), appmw.TeamID(r.Context()), id); err != nil {
		response.Error(w, err)
		return
	}
	response.Success(w, http.StatusOK, map[string]string{"message": "task deleted successfully"})
}

func (h *TaskHandler) Assign(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var req dto.AssignTaskRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, apperror.NewValidation("invalid JSON body"))
		return
	}
	if req.AssigneeID == "" {
		response.Error(w, apperror.NewValidation("assignee_id is required"))
		return
	}
	task, err := h.uc.AssignTask(r.Context(), appmw.UserID(r.Context()), appmw.TeamID(r.Context()), id, req.AssigneeID)
	if err != nil {
		response.Error(w, err)
		return
	}
	response.Success(w, http.StatusOK, task)
}
