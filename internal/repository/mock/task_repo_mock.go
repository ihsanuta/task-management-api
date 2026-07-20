// Package mock provides in-memory, thread-safe fakes of the repository
// interfaces so use cases can be unit tested without a real database or
// any external service. They are intentionally simple but correct with
// respect to concurrency, which is what the idempotency race-condition
// tests depend on.
package mock

import (
	"context"
	"sort"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/google/uuid"
	"github.com/ihsanuta/task-management-api/internal/domain"
	"github.com/ihsanuta/task-management-api/internal/repository"
	"github.com/ihsanuta/task-management-api/pkg/apperror"
)

type TaskRepository struct {
	mu       sync.Mutex
	tasks    map[string]domain.Task
	taskLogs []domain.TaskLog

	// CreateCalls counts how many times Create actually inserted a row.
	// Tests use this to assert no duplicate task was created under
	// concurrent idempotent requests.
	CreateCalls int64
}

func NewTaskRepository() *TaskRepository {
	return &TaskRepository{tasks: make(map[string]domain.Task)}
}

func (r *TaskRepository) Create(_ context.Context, t *domain.Task) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if t.ID == "" {
		t.ID = uuid.NewString()
	}
	r.tasks[t.ID] = *t
	atomic.AddInt64(&r.CreateCalls, 1)
	return nil
}

func (r *TaskRepository) GetByID(_ context.Context, id string) (*domain.Task, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	t, ok := r.tasks[id]
	if !ok {
		return nil, apperror.ErrTaskNotFound
	}
	cp := t
	return &cp, nil
}

func (r *TaskRepository) List(_ context.Context, f repository.TaskFilter) ([]domain.Task, int64, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	var filtered []domain.Task
	for _, t := range r.tasks {
		if t.TeamID != f.TeamID {
			continue
		}
		if t.OwnerID != f.OwnerOrAssigneeID && (t.AssigneeID == nil || *t.AssigneeID != f.OwnerOrAssigneeID) {
			continue
		}
		if f.Status != "" && t.Status != f.Status {
			continue
		}
		if f.Search != "" && !strings.Contains(strings.ToLower(t.Title), strings.ToLower(f.Search)) {
			continue
		}
		filtered = append(filtered, t)
	}
	sort.Slice(filtered, func(i, j int) bool { return filtered[i].CreatedAt.After(filtered[j].CreatedAt) })

	total := int64(len(filtered))
	page, limit := f.Page, f.Limit
	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 10
	}
	start := (page - 1) * limit
	if start > len(filtered) {
		start = len(filtered)
	}
	end := start + limit
	if end > len(filtered) {
		end = len(filtered)
	}
	return filtered[start:end], total, nil
}

func (r *TaskRepository) Update(_ context.Context, t *domain.Task) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.tasks[t.ID]; !ok {
		return apperror.ErrTaskNotFound
	}
	r.tasks[t.ID] = *t
	return nil
}

func (r *TaskRepository) Delete(_ context.Context, id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.tasks[id]; !ok {
		return apperror.ErrTaskNotFound
	}
	delete(r.tasks, id)
	return nil
}

// WithTx fakes a transaction: it locks the whole repo for the duration of
// fn so the tx-scoped operations below are atomic relative to other
// callers, then applies (or discards) the mutation depending on fn's error.
func (r *TaskRepository) WithTx(ctx context.Context, fn func(tx repository.TxTaskRepository) error) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	tx := &txTaskRepository{parent: r, logs: []domain.TaskLog{}}
	if err := fn(tx); err != nil {
		return err // nothing was mutated outside the tx snapshot, i.e. rollback
	}
	// commit
	if tx.updatedTask != nil {
		r.tasks[tx.updatedTask.ID] = *tx.updatedTask
	}
	r.taskLogs = append(r.taskLogs, tx.logs...)
	return nil
}

type txTaskRepository struct {
	parent      *TaskRepository
	updatedTask *domain.Task
	logs        []domain.TaskLog
}

func (tx *txTaskRepository) GetByIDForUpdate(_ context.Context, id string) (*domain.Task, error) {
	t, ok := tx.parent.tasks[id]
	if !ok {
		return nil, apperror.ErrTaskNotFound
	}
	cp := t
	return &cp, nil
}

func (tx *txTaskRepository) UpdateAssignee(_ context.Context, taskID string, assigneeID *string) error {
	t, ok := tx.parent.tasks[taskID]
	if !ok {
		return apperror.ErrTaskNotFound
	}
	t.AssigneeID = assigneeID
	tx.updatedTask = &t
	return nil
}

func (tx *txTaskRepository) InsertTaskLog(_ context.Context, log *domain.TaskLog) error {
	if log.ID == "" {
		log.ID = uuid.NewString()
	}
	tx.logs = append(tx.logs, *log)
	return nil
}
