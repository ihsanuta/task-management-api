package dto

import "github.com/ihsanuta/task-management-api/internal/domain"

type RegisterRequest struct {
	Name     string `json:"name" validate:"required"`
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required,min=8"`
	TeamID   string `json:"team_id,omitempty"`
}

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type AuthResponse struct {
	Token string       `json:"token"`
	User  UserResponse `json:"user"`
}

type UserResponse struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Email  string `json:"email"`
	TeamID string `json:"team_id"`
}

func ToUserResponse(u domain.User) UserResponse {
	return UserResponse{ID: u.ID, Name: u.Name, Email: u.Email, TeamID: u.TeamID}
}

type CreateTaskRequest struct {
	Title       string `json:"title" validate:"required"`
	Description string `json:"description" validate:"required"`
	Status      string `json:"status,omitempty" validate:"omitempty,oneof=pending in_progress done"`
}

type UpdateTaskRequest struct {
	Title       *string `json:"title,omitempty"`
	Description *string `json:"description,omitempty"`
	Status      *string `json:"status,omitempty"`
}

type AssignTaskRequest struct {
	AssigneeID string `json:"assignee_id"`
}

type TaskResponse struct {
	ID          string  `json:"id"`
	Title       string  `json:"title"`
	Description string  `json:"description"`
	Status      string  `json:"status"`
	OwnerID     string  `json:"owner_id"`
	AssigneeID  *string `json:"assignee_id,omitempty"`
	TeamID      string  `json:"team_id"`
	CreatedAt   string  `json:"created_at"`
	UpdatedAt   string  `json:"updated_at"`
}

func ToTaskResponse(t domain.Task) TaskResponse {
	return TaskResponse{
		ID:          t.ID,
		Title:       t.Title,
		Description: t.Description,
		Status:      string(t.Status),
		OwnerID:     t.OwnerID,
		AssigneeID:  t.AssigneeID,
		TeamID:      t.TeamID,
		CreatedAt:   t.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt:   t.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
}
