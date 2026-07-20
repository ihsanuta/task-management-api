package handler

import (
	"encoding/json"
	"net/http"

	"github.com/go-playground/validator/v10"
	"github.com/ihsanuta/task-management-api/internal/delivery/http/dto"
	"github.com/ihsanuta/task-management-api/internal/usecase"
	"github.com/ihsanuta/task-management-api/pkg/apperror"
	"github.com/ihsanuta/task-management-api/pkg/response"
)

type AuthHandler struct {
	uc        *usecase.AuthUsecase
	validator *validator.Validate
}

func NewAuthHandler(uc *usecase.AuthUsecase, validator *validator.Validate) *AuthHandler {
	return &AuthHandler{
		uc:        uc,
		validator: validator,
	}
}

func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req dto.RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, apperror.NewValidation("invalid JSON body"))
		return
	}

	err := h.validator.Struct(req)
	if err != nil {
		response.Error(w, apperror.NewValidationError(err))
		return
	}

	res, err := h.uc.Register(r.Context(), req)
	if err != nil {
		response.Error(w, err)
		return
	}
	response.Success(w, http.StatusCreated, res)
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req dto.LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, apperror.NewValidation("invalid JSON body"))
		return
	}
	res, err := h.uc.Login(r.Context(), req)
	if err != nil {
		response.Error(w, err)
		return
	}
	response.Success(w, http.StatusOK, res)
}
