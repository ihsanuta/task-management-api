package mock

import (
	"context"
	"sync"

	"github.com/google/uuid"
	"github.com/ihsanuta/task-management-api/internal/domain"
	"github.com/ihsanuta/task-management-api/pkg/apperror"
)

type UserRepository struct {
	mu    sync.Mutex
	users map[string]domain.User // keyed by ID
}

func NewUserRepository() *UserRepository {
	return &UserRepository{users: make(map[string]domain.User)}
}

func (r *UserRepository) Create(_ context.Context, u *domain.User) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, existing := range r.users {
		if existing.Email == u.Email {
			return apperror.ErrEmailAlreadyRegistered
		}
	}
	if u.ID == "" {
		u.ID = uuid.NewString()
	}
	r.users[u.ID] = *u
	return nil
}

func (r *UserRepository) GetByEmail(_ context.Context, email string) (*domain.User, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, u := range r.users {
		if u.Email == email {
			cp := u
			return &cp, nil
		}
	}
	return nil, apperror.NewNotFound("user not found")
}

func (r *UserRepository) GetByID(_ context.Context, id string) (*domain.User, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	u, ok := r.users[id]
	if !ok {
		return nil, apperror.NewNotFound("user not found")
	}
	cp := u
	return &cp, nil
}
