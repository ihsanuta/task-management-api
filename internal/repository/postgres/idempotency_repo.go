package postgres

import (
	"context"
	"database/sql"
	"errors"

	"github.com/ihsanuta/task-management-api/internal/domain"
)

type IdempotencyRepository struct{ db *sql.DB }

func NewIdempotencyRepository(db *sql.DB) *IdempotencyRepository {
	return &IdempotencyRepository{db: db}
}

func (r *IdempotencyRepository) GetByKey(ctx context.Context, key string) (*domain.IdempotencyRecord, error) {
	query := `SELECT key, user_id, endpoint, request_hash, status, response_status, response_body, created_at, expires_at
	          FROM idempotency_keys WHERE key = $1 AND expires_at > now()`
	rec := &domain.IdempotencyRecord{}
	err := r.db.QueryRowContext(ctx, query, key).Scan(
		&rec.Key, &rec.UserID, &rec.Endpoint, &rec.RequestHash, &rec.Status,
		&rec.ResponseStatus, &rec.ResponseBody, &rec.CreatedAt, &rec.ExpiresAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return rec, nil
}

// TryCreate relies on the primary key constraint on `key` to make the
// claim atomic at the database level: under concurrent requests with the
// same Idempotency-Key, only one INSERT wins; every other connection gets
// ON CONFLICT DO NOTHING and 0 rows affected, so claimed=false. This is
// what guarantees "only exactly one task is created" even under real
// concurrent load across multiple app instances (not just one process).
func (r *IdempotencyRepository) TryCreate(ctx context.Context, rec *domain.IdempotencyRecord) (bool, error) {
	query := `INSERT INTO idempotency_keys (key, user_id, endpoint, request_hash, status, response_status, response_body, created_at, expires_at)
	          VALUES ($1, $2, $3, $4, $5, 0, '{}', $6, $7)
	          ON CONFLICT (key) DO NOTHING`
	res, err := r.db.ExecContext(ctx, query, rec.Key, rec.UserID, rec.Endpoint, rec.RequestHash, rec.Status, rec.CreatedAt, rec.ExpiresAt)
	if err != nil {
		return false, err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return false, err
	}
	return n == 1, nil
}

func (r *IdempotencyRepository) Complete(ctx context.Context, key string, status int, body []byte) error {
	query := `UPDATE idempotency_keys SET status = $1, response_status = $2, response_body = $3 WHERE key = $4`
	_, err := r.db.ExecContext(ctx, query, domain.IdempotencyCompleted, status, body, key)
	return err
}

func (r *IdempotencyRepository) Delete(ctx context.Context, key string) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM idempotency_keys WHERE key = $1`, key)
	return err
}
