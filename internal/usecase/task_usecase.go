package usecase

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/ihsanuta/task-management-api/internal/delivery/http/dto"
	"github.com/ihsanuta/task-management-api/internal/domain"
	"github.com/ihsanuta/task-management-api/internal/repository"
	"github.com/ihsanuta/task-management-api/pkg/apperror"
)

type TaskUsecase struct {
	tasks    repository.TaskRepository
	users    repository.UserRepository
	idemRepo repository.IdempotencyRepository
	idemTTL  time.Duration
}

func NewTaskUsecase(tasks repository.TaskRepository, users repository.UserRepository, idemRepo repository.IdempotencyRepository, idemTTL time.Duration) *TaskUsecase {
	return &TaskUsecase{tasks: tasks, users: users, idemRepo: idemRepo, idemTTL: idemTTL}
}

type IdempotentResult struct {
	Task     *dto.TaskResponse
	Status   int
	Replayed bool
}

func (uc *TaskUsecase) CreateTask(ctx context.Context, ownerID, teamID string, req dto.CreateTaskRequest, idemKey string) (*IdempotentResult, error) {
	if idemKey != "" {
		if _, err := uuid.Parse(idemKey); err != nil {
			return nil, apperror.ErrInvalidIdempotencyKey
		}
		return uc.createTaskIdempotent(ctx, ownerID, teamID, req, idemKey)
	}

	task, err := uc.buildAndPersistTask(ctx, ownerID, teamID, req)
	if err != nil {
		return nil, err
	}
	resp := dto.ToTaskResponse(*task)
	return &IdempotentResult{Task: &resp, Status: http.StatusCreated, Replayed: false}, nil
}

func (uc *TaskUsecase) createTaskIdempotent(ctx context.Context, ownerID, teamID string, req dto.CreateTaskRequest, idemKey string) (*IdempotentResult, error) {
	requestHash := hashRequest(req)

	existing, err := uc.idemRepo.GetByKey(ctx, idemKey)
	if err != nil {
		return nil, apperror.NewInternal(err)
	}
	if existing != nil {
		return replayOrReject(existing, requestHash)
	}

	claimed, err := uc.idemRepo.TryCreate(ctx, &domain.IdempotencyRecord{
		Key:         idemKey,
		UserID:      ownerID,
		Endpoint:    "POST /tasks",
		RequestHash: requestHash,
		Status:      domain.IdempotencyProcessing,
		CreatedAt:   time.Now().UTC(),
		ExpiresAt:   time.Now().UTC().Add(uc.idemTTL),
	})
	if err != nil {
		return nil, apperror.NewInternal(err)
	}
	if !claimed {
		again, err := uc.idemRepo.GetByKey(ctx, idemKey)
		if err != nil {
			return nil, apperror.NewInternal(err)
		}
		if again != nil {
			return replayOrReject(again, requestHash)
		}
		return nil, apperror.ErrIdempotencyInProgress
	}

	task, err := uc.buildAndPersistTask(ctx, ownerID, teamID, req)
	if err != nil {
		_ = uc.idemRepo.Delete(ctx, idemKey)
		return nil, err
	}

	resp := dto.ToTaskResponse(*task)
	body, _ := json.Marshal(resp)
	if err := uc.idemRepo.Complete(ctx, idemKey, http.StatusCreated, body); err != nil {
		slog.Error("failed to persist idempotency completion", "error", err, "idempotency_key", idemKey)
	}

	return &IdempotentResult{Task: &resp, Status: http.StatusCreated, Replayed: false}, nil
}

func replayOrReject(existing *domain.IdempotencyRecord, requestHash string) (*IdempotentResult, error) {
	if existing.RequestHash != requestHash {
		return nil, apperror.ErrIdempotencyKeyReused
	}
	switch existing.Status {
	case domain.IdempotencyCompleted:
		var resp dto.TaskResponse
		if err := json.Unmarshal(existing.ResponseBody, &resp); err != nil {
			return nil, apperror.NewInternal(err)
		}
		return &IdempotentResult{Task: &resp, Status: existing.ResponseStatus, Replayed: true}, nil
	default: // still processing
		return nil, apperror.ErrIdempotencyInProgress
	}
}

func hashRequest(req dto.CreateTaskRequest) string {
	sum := sha256.Sum256([]byte(req.Title + "\x00" + req.Description + "\x00" + req.Status))
	return hex.EncodeToString(sum[:])
}

func (uc *TaskUsecase) buildAndPersistTask(ctx context.Context, ownerID, teamID string, req dto.CreateTaskRequest) (*domain.Task, error) {
	status := domain.TaskStatus(req.Status)
	if status == "" {
		status = domain.TaskStatusPending
	}
	now := time.Now().UTC()
	task := &domain.Task{
		ID:          uuid.NewString(),
		Title:       req.Title,
		Description: req.Description,
		Status:      status,
		OwnerID:     ownerID,
		TeamID:      teamID,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if err := uc.tasks.Create(ctx, task); err != nil {
		return nil, apperror.NewInternal(err)
	}
	return task, nil
}

func (uc *TaskUsecase) ListTasks(ctx context.Context, userID, teamID, status, search string, page, limit int) ([]dto.TaskResponse, int64, error) {
	if status != "" && !domain.TaskStatus(status).Valid() {
		return nil, 0, apperror.NewValidation("status must be one of: pending, in_progress, done")
	}
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 10
	}

	tasks, total, err := uc.tasks.List(ctx, repository.TaskFilter{
		OwnerOrAssigneeID: userID,
		TeamID:            teamID,
		Status:            domain.TaskStatus(status),
		Search:            search,
		Page:              page,
		Limit:             limit,
	})
	if err != nil {
		return nil, 0, apperror.NewInternal(err)
	}

	out := make([]dto.TaskResponse, 0, len(tasks))
	for _, t := range tasks {
		out = append(out, dto.ToTaskResponse(t))
	}
	return out, total, nil
}

func (uc *TaskUsecase) GetTask(ctx context.Context, userID, teamID, id string) (*dto.TaskResponse, error) {
	task, err := uc.getOwnedOrVisibleTask(ctx, userID, teamID, id)
	if err != nil {
		return nil, err
	}
	resp := dto.ToTaskResponse(*task)
	return &resp, nil
}

func (uc *TaskUsecase) UpdateTask(ctx context.Context, userID, teamID, id string, req dto.UpdateTaskRequest) (*dto.TaskResponse, error) {
	task, err := uc.tasks.GetByID(ctx, id)
	if err != nil {
		return nil, apperror.ErrTaskNotFound
	}
	if task.TeamID != teamID {
		return nil, apperror.ErrTaskForbidden
	}
	if task.OwnerID != userID && (task.AssigneeID == nil || *task.AssigneeID != userID) {
		return nil, apperror.ErrTaskForbidden
	}

	if req.Title != nil {
		title := strings.TrimSpace(*req.Title)
		if title == "" {
			return nil, apperror.NewValidation("title cannot be empty")
		}
		task.Title = title
	}
	if req.Description != nil {
		task.Description = *req.Description
	}
	if req.Status != nil {
		if !domain.TaskStatus(*req.Status).Valid() {
			return nil, apperror.NewValidation("status must be one of: pending, in_progress, done")
		}
		task.Status = domain.TaskStatus(*req.Status)
	}
	task.UpdatedAt = time.Now().UTC()

	if err := uc.tasks.Update(ctx, task); err != nil {
		return nil, apperror.NewInternal(err)
	}
	resp := dto.ToTaskResponse(*task)
	return &resp, nil
}

func (uc *TaskUsecase) DeleteTask(ctx context.Context, userID, teamID, id string) error {
	task, err := uc.tasks.GetByID(ctx, id)
	if err != nil {
		return apperror.ErrTaskNotFound
	}
	if task.TeamID != teamID {
		return apperror.ErrTaskForbidden
	}
	if task.OwnerID != userID {
		return apperror.ErrTaskForbidden
	}
	if err := uc.tasks.Delete(ctx, id); err != nil {
		return apperror.NewInternal(err)
	}
	return nil
}

func (uc *TaskUsecase) getOwnedOrVisibleTask(ctx context.Context, userID, teamID, id string) (*domain.Task, error) {
	task, err := uc.tasks.GetByID(ctx, id)
	if err != nil {
		return nil, apperror.ErrTaskNotFound
	}
	if task.TeamID != teamID {
		return nil, apperror.ErrTaskForbidden
	}
	if task.OwnerID != userID && (task.AssigneeID == nil || *task.AssigneeID != userID) {
		return nil, apperror.ErrTaskForbidden
	}
	return task, nil
}

func (uc *TaskUsecase) AssignTask(ctx context.Context, actorID, teamID, taskID, assigneeID string) (*dto.TaskResponse, error) {
	assignee, err := uc.users.GetByID(ctx, assigneeID)
	if err != nil {
		return nil, apperror.ErrAssigneeNotFound
	}
	if assignee.TeamID != teamID {
		return nil, apperror.ErrAssigneeNotInTeam
	}

	var result domain.Task
	err = uc.tasks.WithTx(ctx, func(tx repository.TxTaskRepository) error {
		task, err := tx.GetByIDForUpdate(ctx, taskID)
		if err != nil {
			return apperror.ErrTaskNotFound
		}
		if task.TeamID != teamID {
			return apperror.ErrTaskForbidden
		}

		oldAssignee := task.AssigneeID
		newAssignee := assigneeID

		if err := tx.UpdateAssignee(ctx, taskID, &newAssignee); err != nil {
			return apperror.NewInternal(err)
		}

		if err := tx.InsertTaskLog(ctx, &domain.TaskLog{
			TaskID:        taskID,
			Action:        "ASSIGNED",
			OldAssigneeID: oldAssignee,
			NewAssigneeID: &newAssignee,
			ChangedBy:     actorID,
			CreatedAt:     time.Now().UTC(),
		}); err != nil {
			return apperror.NewInternal(err)
		}

		if err := notifyAssignee(ctx, assignee.Email, taskID); err != nil {
			return apperror.NewInternal(err)
		}

		task.AssigneeID = &newAssignee
		result = *task
		return nil
	})
	if err != nil {
		if appErr, ok := err.(*apperror.AppError); ok {
			return nil, appErr
		}
		return nil, apperror.NewInternal(err)
	}

	resp := dto.ToTaskResponse(result)
	return &resp, nil
}

func notifyAssignee(_ context.Context, email, taskID string) error {
	slog.Info("notification sent", "channel", "mock", "to", email, "task_id", taskID, "message", "you have been assigned a new task")
	return nil
}
