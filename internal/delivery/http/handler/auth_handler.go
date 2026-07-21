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

// Register godoc
// @Summary Daftar user baru
// @Description Mendaftarkan user baru ke dalam sistem
// @Tags Auth
// @Accept json
// @Produce json
// @Param request body dto.RegisterRequest true "Data Pendaftaran"
// @Success 201 {object} response.Envelope "Berhasil didaftarkan"
// @Failure 400 {object} response.Envelope "Payload tidak valid"
// @Router /auth/register [post]
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

// Login godoc
// @Summary Login user
// @Description Login user ke dalam sistem
// @Tags Auth
// @Accept json
// @Produce json
// @Param request body dto.LoginRequest true "Data login"
// @Success 200 {object} response.Envelope "Berhasil login"
// @Failure 400 {object} response.Envelope "Payload tidak valid"
// @Router /auth/login [post]
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
