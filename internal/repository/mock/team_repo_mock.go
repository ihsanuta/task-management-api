package mock

import (
	"context"
	"sync"

	"github.com/google/uuid"
	"github.com/ihsanuta/task-management-api/internal/domain"
	"github.com/ihsanuta/task-management-api/pkg/apperror"
)

type TeamRepository struct {
	mu    sync.Mutex
	teams map[string]domain.Team
}

func NewTeamRepository() *TeamRepository {
	return &TeamRepository{teams: make(map[string]domain.Team)}
}

func (r *TeamRepository) Create(_ context.Context, t *domain.Team) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if t.ID == "" {
		t.ID = uuid.NewString()
	}
	r.teams[t.ID] = *t
	return nil
}

func (r *TeamRepository) GetByID(_ context.Context, id string) (*domain.Team, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	t, ok := r.teams[id]
	if !ok {
		return nil, apperror.NewNotFound("team not found")
	}
	cp := t
	return &cp, nil
}
