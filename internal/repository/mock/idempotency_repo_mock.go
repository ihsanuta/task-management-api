package mock

import (
	"context"
	"sync"

	"github.com/ihsanuta/task-management-api/internal/domain"
)

// IdempotencyRepository is an in-memory fake. TryCreate uses a single mutex
// to make the "check-then-insert" step atomic, which is exactly what
// prevents duplicate task creation when many goroutines race on the same
// Idempotency-Key (mirrors an `INSERT ... ON CONFLICT DO NOTHING` in
// Postgres, see internal/repository/postgres/idempotency_repo.go).
type IdempotencyRepository struct {
	mu      sync.Mutex
	records map[string]domain.IdempotencyRecord
}

func NewIdempotencyRepository() *IdempotencyRepository {
	return &IdempotencyRepository{records: make(map[string]domain.IdempotencyRecord)}
}

func (r *IdempotencyRepository) GetByKey(_ context.Context, key string) (*domain.IdempotencyRecord, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	rec, ok := r.records[key]
	if !ok {
		return nil, nil
	}
	cp := rec
	return &cp, nil
}

func (r *IdempotencyRepository) TryCreate(_ context.Context, rec *domain.IdempotencyRecord) (bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.records[rec.Key]; exists {
		return false, nil
	}
	r.records[rec.Key] = *rec
	return true, nil
}

func (r *IdempotencyRepository) Complete(_ context.Context, key string, status int, body []byte) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	rec, ok := r.records[key]
	if !ok {
		return nil
	}
	rec.Status = domain.IdempotencyCompleted
	rec.ResponseStatus = status
	rec.ResponseBody = body
	r.records[key] = rec
	return nil
}

func (r *IdempotencyRepository) Delete(_ context.Context, key string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.records, key)
	return nil
}
