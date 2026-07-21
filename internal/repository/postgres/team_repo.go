package postgres

import (
	"context"
	"database/sql"
	"errors"

	"github.com/ihsanuta/task-management-api/internal/domain"
	"github.com/ihsanuta/task-management-api/pkg/apperror"
)

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
