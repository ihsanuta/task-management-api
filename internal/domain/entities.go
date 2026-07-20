package domain

import "time"

// TaskStatus enumerates the allowed lifecycle states of a Task.
type TaskStatus string

const (
	TaskStatusPending    TaskStatus = "pending"
	TaskStatusInProgress TaskStatus = "in_progress"
	TaskStatusDone       TaskStatus = "done"
)

func (s TaskStatus) Valid() bool {
	switch s {
	case TaskStatusPending, TaskStatusInProgress, TaskStatusDone:
		return true
	}
	return false
}

type Team struct {
	ID        string    `json:"id" db:"id"`
	Name      string    `json:"name" db:"name"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
}

type User struct {
	ID           string    `json:"id" db:"id"`
	Name         string    `json:"name" db:"name"`
	Email        string    `json:"email" db:"email"`
	PasswordHash string    `json:"-" db:"password_hash"`
	TeamID       string    `json:"team_id" db:"team_id"`
	CreatedAt    time.Time `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time `json:"updated_at" db:"updated_at"`
}

type Task struct {
	ID          string     `json:"id" db:"id"`
	Title       string     `json:"title" db:"title"`
	Description string     `json:"description" db:"description"`
	Status      TaskStatus `json:"status" db:"status"`
	OwnerID     string     `json:"owner_id" db:"owner_id"`
	AssigneeID  *string    `json:"assignee_id,omitempty" db:"assignee_id"`
	TeamID      string     `json:"team_id" db:"team_id"`
	CreatedAt   time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at" db:"updated_at"`
}

type TaskLog struct {
	ID            string    `json:"id" db:"id"`
	TaskID        string    `json:"task_id" db:"task_id"`
	Action        string    `json:"action" db:"action"`
	OldAssigneeID *string   `json:"old_assignee_id,omitempty" db:"old_assignee_id"`
	NewAssigneeID *string   `json:"new_assignee_id,omitempty" db:"new_assignee_id"`
	ChangedBy     string    `json:"changed_by" db:"changed_by"`
	CreatedAt     time.Time `json:"created_at" db:"created_at"`
}

// IdempotencyStatus tracks the lifecycle of an idempotent request.
type IdempotencyStatus string

const (
	IdempotencyProcessing IdempotencyStatus = "processing"
	IdempotencyCompleted  IdempotencyStatus = "completed"
)

// IdempotencyRecord represents a stored idempotency key and, once the
// original request has finished processing, the response that was returned
// for it so retries can be replayed verbatim.
type IdempotencyRecord struct {
	Key            string            `json:"key" db:"key"`
	UserID         string            `json:"user_id" db:"user_id"`
	Endpoint       string            `json:"endpoint" db:"endpoint"`
	RequestHash    string            `json:"request_hash" db:"request_hash"`
	Status         IdempotencyStatus `json:"status" db:"status"`
	ResponseStatus int               `json:"response_status" db:"response_status"`
	ResponseBody   []byte            `json:"response_body" db:"response_body"`
	CreatedAt      time.Time         `json:"created_at" db:"created_at"`
	ExpiresAt      time.Time         `json:"expires_at" db:"expires_at"`
}
