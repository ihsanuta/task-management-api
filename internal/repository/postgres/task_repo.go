package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/ihsanuta/task-management-api/internal/domain"
	"github.com/ihsanuta/task-management-api/internal/repository"
	"github.com/ihsanuta/task-management-api/pkg/apperror"
)

type TaskRepository struct{ db *sql.DB }

func NewTaskRepository(db *sql.DB) *TaskRepository { return &TaskRepository{db: db} }

func (r *TaskRepository) Create(ctx context.Context, t *domain.Task) error {
	query := `INSERT INTO tasks (id, title, description, status, owner_id, assignee_id, team_id, created_at, updated_at)
	          VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`
	_, err := r.db.ExecContext(ctx, query, t.ID, t.Title, t.Description, t.Status, t.OwnerID, t.AssigneeID, t.TeamID, t.CreatedAt, t.UpdatedAt)
	return err
}

func (r *TaskRepository) GetByID(ctx context.Context, id string) (*domain.Task, error) {
	query := `SELECT id, title, description, status, owner_id, assignee_id, team_id, created_at, updated_at FROM tasks WHERE id = $1`
	return scanTask(r.db.QueryRowContext(ctx, query, id))
}

func scanTask(row *sql.Row) (*domain.Task, error) {
	t := &domain.Task{}
	err := row.Scan(&t.ID, &t.Title, &t.Description, &t.Status, &t.OwnerID, &t.AssigneeID, &t.TeamID, &t.CreatedAt, &t.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, apperror.ErrTaskNotFound
	}
	if err != nil {
		return nil, err
	}
	return t, nil
}

func (r *TaskRepository) List(ctx context.Context, f repository.TaskFilter) ([]domain.Task, int64, error) {
	where := []string{"team_id = $1", "(owner_id = $2 OR assignee_id = $2)"}
	args := []interface{}{f.TeamID, f.OwnerOrAssigneeID}
	idx := 3

	if f.Status != "" {
		where = append(where, fmt.Sprintf("status = $%d", idx))
		args = append(args, f.Status)
		idx++
	}
	if f.Search != "" {
		where = append(where, fmt.Sprintf("title ILIKE $%d", idx))
		args = append(args, "%"+f.Search+"%")
		idx++
	}
	whereClause := strings.Join(where, " AND ")

	var total int64
	countQuery := `SELECT COUNT(*) FROM tasks WHERE ` + whereClause
	if err := r.db.QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	limit := f.Limit
	offset := (f.Page - 1) * f.Limit
	listQuery := fmt.Sprintf(`SELECT id, title, description, status, owner_id, assignee_id, team_id, created_at, updated_at
		FROM tasks WHERE %s ORDER BY created_at DESC LIMIT $%d OFFSET $%d`, whereClause, idx, idx+1)
	args = append(args, limit, offset)

	rows, err := r.db.QueryContext(ctx, listQuery, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var tasks []domain.Task
	for rows.Next() {
		var t domain.Task
		if err := rows.Scan(&t.ID, &t.Title, &t.Description, &t.Status, &t.OwnerID, &t.AssigneeID, &t.TeamID, &t.CreatedAt, &t.UpdatedAt); err != nil {
			return nil, 0, err
		}
		tasks = append(tasks, t)
	}
	return tasks, total, rows.Err()
}

func (r *TaskRepository) Update(ctx context.Context, t *domain.Task) error {
	query := `UPDATE tasks SET title=$1, description=$2, status=$3, updated_at=$4 WHERE id=$5`
	res, err := r.db.ExecContext(ctx, query, t.Title, t.Description, t.Status, t.UpdatedAt, t.ID)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return apperror.ErrTaskNotFound
	}
	return nil
}

func (r *TaskRepository) Delete(ctx context.Context, id string) error {
	res, err := r.db.ExecContext(ctx, `DELETE FROM tasks WHERE id = $1`, id)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return apperror.ErrTaskNotFound
	}
	return nil
}

func (r *TaskRepository) WithTx(ctx context.Context, fn func(tx repository.TxTaskRepository) error) error {
	sqlTx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}

	txRepo := &pgTxTaskRepository{tx: sqlTx}
	if err := fn(txRepo); err != nil {
		if rbErr := sqlTx.Rollback(); rbErr != nil {
			return fmt.Errorf("rollback failed: %v (original error: %w)", rbErr, err)
		}
		return err
	}
	return sqlTx.Commit()
}

type pgTxTaskRepository struct{ tx *sql.Tx }

func (t *pgTxTaskRepository) GetByIDForUpdate(ctx context.Context, id string) (*domain.Task, error) {
	query := `SELECT id, title, description, status, owner_id, assignee_id, team_id, created_at, updated_at
	          FROM tasks WHERE id = $1 FOR UPDATE`
	row := t.tx.QueryRowContext(ctx, query, id)
	task := &domain.Task{}
	err := row.Scan(&task.ID, &task.Title, &task.Description, &task.Status, &task.OwnerID, &task.AssigneeID, &task.TeamID, &task.CreatedAt, &task.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, apperror.ErrTaskNotFound
	}
	if err != nil {
		return nil, err
	}
	return task, nil
}

func (t *pgTxTaskRepository) UpdateAssignee(ctx context.Context, taskID string, assigneeID *string) error {
	_, err := t.tx.ExecContext(ctx, `UPDATE tasks SET assignee_id = $1, updated_at = now() WHERE id = $2`, assigneeID, taskID)
	return err
}

func (t *pgTxTaskRepository) InsertTaskLog(ctx context.Context, log *domain.TaskLog) error {
	if log.ID == "" {
		log.ID = uuid.NewString()
	}
	_, err := t.tx.ExecContext(ctx,
		`INSERT INTO task_logs (id, task_id, action, old_assignee_id, new_assignee_id, changed_by, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		log.ID, log.TaskID, log.Action, log.OldAssigneeID, log.NewAssigneeID, log.ChangedBy, log.CreatedAt)
	return err
}
