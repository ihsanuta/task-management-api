package repository

import (
	"context"

	"github.com/ihsanuta/task-management-api/internal/domain"
)

type UserRepository interface {
	Create(ctx context.Context, u *domain.User) error
	GetByEmail(ctx context.Context, email string) (*domain.User, error)
	GetByID(ctx context.Context, id string) (*domain.User, error)
}

type TeamRepository interface {
	Create(ctx context.Context, t *domain.Team) error
	GetByID(ctx context.Context, id string) (*domain.Team, error)
}

type TaskFilter struct {
	OwnerOrAssigneeID string // restrict to tasks visible to this user
	TeamID            string
	Status            domain.TaskStatus
	Search            string
	Page              int
	Limit             int
}

type TaskRepository interface {
	Create(ctx context.Context, t *domain.Task) error
	GetByID(ctx context.Context, id string) (*domain.Task, error)
	List(ctx context.Context, f TaskFilter) ([]domain.Task, int64, error)
	Update(ctx context.Context, t *domain.Task) error
	Delete(ctx context.Context, id string) error
	WithTx(ctx context.Context, fn func(txRepo TxTaskRepository) error) error
}

type TxTaskRepository interface {
	GetByIDForUpdate(ctx context.Context, id string) (*domain.Task, error)
	UpdateAssignee(ctx context.Context, taskID string, assigneeID *string) error
	InsertTaskLog(ctx context.Context, log *domain.TaskLog) error
}

type IdempotencyRepository interface {
	GetByKey(ctx context.Context, key string) (*domain.IdempotencyRecord, error)
	TryCreate(ctx context.Context, rec *domain.IdempotencyRecord) (claimed bool, err error)
	Complete(ctx context.Context, key string, responseStatus int, responseBody []byte) error
	Delete(ctx context.Context, key string) error
}
