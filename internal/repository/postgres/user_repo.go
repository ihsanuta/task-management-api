package postgres

import (
	"context"
	"database/sql"
	"errors"

	"github.com/ihsanuta/task-management-api/internal/domain"
	"github.com/ihsanuta/task-management-api/pkg/apperror"
	"github.com/lib/pq"
)

type UserRepository struct{ db *sql.DB }

func NewUserRepository(db *sql.DB) *UserRepository { return &UserRepository{db: db} }

func (r *UserRepository) Create(ctx context.Context, u *domain.User) error {
	query := `INSERT INTO users (id, name, email, password_hash, team_id, created_at, updated_at)
	          VALUES ($1, $2, $3, $4, $5, now(), now())`
	_, err := r.db.ExecContext(ctx, query, u.ID, u.Name, u.Email, u.PasswordHash, u.TeamID)
	if err != nil {
		var pqErr *pq.Error
		if errors.As(err, &pqErr) && pqErr.Code == "23505" { // unique_violation
			return apperror.ErrEmailAlreadyRegistered
		}
		return err
	}
	return nil
}

func (r *UserRepository) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	query := `SELECT id, name, email, password_hash, team_id, created_at, updated_at FROM users WHERE email = $1`
	u := &domain.User{}
	err := r.db.QueryRowContext(ctx, query, email).Scan(&u.ID, &u.Name, &u.Email, &u.PasswordHash, &u.TeamID, &u.CreatedAt, &u.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, apperror.NewNotFound("user not found")
	}
	if err != nil {
		return nil, err
	}
	return u, nil
}

func (r *UserRepository) GetByID(ctx context.Context, id string) (*domain.User, error) {
	query := `SELECT id, name, email, password_hash, team_id, created_at, updated_at FROM users WHERE id = $1`
	u := &domain.User{}
	err := r.db.QueryRowContext(ctx, query, id).Scan(&u.ID, &u.Name, &u.Email, &u.PasswordHash, &u.TeamID, &u.CreatedAt, &u.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, apperror.NewNotFound("user not found")
	}
	if err != nil {
		return nil, err
	}
	return u, nil
}

type TeamRepository struct{ db *sql.DB }

func NewTeamRepository(db *sql.DB) *TeamRepository { return &TeamRepository{db: db} }

func (r *TeamRepository) Create(ctx context.Context, t *domain.Team) error {
	query := `INSERT INTO teams (id, name, created_at) VALUES ($1, $2, now())`
	_, err := r.db.ExecContext(ctx, query, t.ID, t.Name)
	return err
}

func (r *TeamRepository) GetByID(ctx context.Context, id string) (*domain.Team, error) {
	query := `SELECT id, name, created_at FROM teams WHERE id = $1`
	t := &domain.Team{}
	err := r.db.QueryRowContext(ctx, query, id).Scan(&t.ID, &t.Name, &t.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, apperror.NewNotFound("team not found")
	}
	if err != nil {
		return nil, err
	}
	return t, nil
}
