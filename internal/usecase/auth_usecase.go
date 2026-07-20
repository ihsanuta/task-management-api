package usecase

import (
	"context"
	"regexp"
	"strings"

	"github.com/google/uuid"
	"github.com/ihsanuta/task-management-api/internal/delivery/http/dto"
	"github.com/ihsanuta/task-management-api/internal/domain"
	"github.com/ihsanuta/task-management-api/internal/repository"
	"github.com/ihsanuta/task-management-api/pkg/apperror"
	"github.com/ihsanuta/task-management-api/pkg/jwtutil"
	"github.com/ihsanuta/task-management-api/pkg/pwhash"
)

var emailRe = regexp.MustCompile(`^[^\s@]+@[^\s@]+\.[^\s@]+$`)

type AuthUsecase struct {
	users repository.UserRepository
	teams repository.TeamRepository
	jwt   *jwtutil.Manager
}

func NewAuthUsecase(users repository.UserRepository, teams repository.TeamRepository, jwt *jwtutil.Manager) *AuthUsecase {
	return &AuthUsecase{users: users, teams: teams, jwt: jwt}
}

func (uc *AuthUsecase) Register(ctx context.Context, req dto.RegisterRequest) (*dto.AuthResponse, error) {
	req.Name = strings.TrimSpace(req.Name)
	req.Email = strings.ToLower(strings.TrimSpace(req.Email))
	teamID := req.TeamID
	if teamID == "" {
		team := &domain.Team{ID: uuid.NewString(), Name: req.Name + "'s Team"}
		if err := uc.teams.Create(ctx, team); err != nil {
			return nil, apperror.NewInternal(err)
		}
		teamID = team.ID
	} else if _, err := uc.teams.GetByID(ctx, teamID); err != nil {
		return nil, apperror.NewValidation("team_id does not exist")
	}

	hash, err := pwhash.Hash(req.Password)
	if err != nil {
		return nil, apperror.NewInternal(err)
	}

	user := &domain.User{
		ID:           uuid.NewString(),
		Name:         req.Name,
		Email:        req.Email,
		PasswordHash: hash,
		TeamID:       teamID,
	}
	if err := uc.users.Create(ctx, user); err != nil {
		if appErr, ok := err.(*apperror.AppError); ok {
			return nil, appErr
		}
		return nil, apperror.NewInternal(err)
	}

	token, err := uc.jwt.Generate(user.ID, user.TeamID, user.Email)
	if err != nil {
		return nil, apperror.NewInternal(err)
	}

	return &dto.AuthResponse{Token: token, User: dto.ToUserResponse(*user)}, nil
}

func (uc *AuthUsecase) Login(ctx context.Context, req dto.LoginRequest) (*dto.AuthResponse, error) {
	email := strings.ToLower(strings.TrimSpace(req.Email))
	user, err := uc.users.GetByEmail(ctx, email)
	if err != nil {
		return nil, apperror.ErrInvalidCredentials
	}
	if !pwhash.Verify(req.Password, user.PasswordHash) {
		return nil, apperror.ErrInvalidCredentials
	}
	token, err := uc.jwt.Generate(user.ID, user.TeamID, user.Email)
	if err != nil {
		return nil, apperror.NewInternal(err)
	}
	return &dto.AuthResponse{Token: token, User: dto.ToUserResponse(*user)}, nil
}
