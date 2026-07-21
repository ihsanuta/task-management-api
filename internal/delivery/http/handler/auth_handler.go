package handler

import (
	"net/http"

	"github.com/go-playground/validator/v10"
	"github.com/ihsanuta/task-management-api/internal/delivery/http/dto"
	"github.com/ihsanuta/task-management-api/internal/usecase"
	"github.com/ihsanuta/task-management-api/pkg/apperror"
	"github.com/ihsanuta/task-management-api/pkg/response"
	"github.com/labstack/echo/v4"
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

func (h *AuthHandler) Register(c echo.Context) error {
	var req dto.RegisterRequest

	if err := c.Bind(&req); err != nil {
		return response.Error(c, apperror.NewValidation("invalid JSON body"))
	}

	err := h.validator.Struct(req)
	if err != nil {
		return response.Error(c, apperror.NewValidationError(err))
	}

	res, err := h.uc.Register(c.Request().Context(), req)
	if err != nil {
		return response.Error(c, err)
	}
	return response.Success(c, http.StatusCreated, res)
}

func (h *AuthHandler) Login(c echo.Context) error {
	var req dto.LoginRequest
	if err := c.Bind(&req); err != nil {
		return response.Error(c, apperror.NewValidation("invalid JSON body"))
	}

	err := h.validator.Struct(req)
	if err != nil {
		return response.Error(c, apperror.NewValidationError(err))
	}

	res, err := h.uc.Login(c.Request().Context(), req)
	if err != nil {
		return response.Error(c, err)
	}
	return response.Success(c, http.StatusOK, res)
}
