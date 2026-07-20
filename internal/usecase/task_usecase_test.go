package usecase_test

import (
	"context"
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/ihsanuta/task-management-api/internal/delivery/http/dto"
	"github.com/ihsanuta/task-management-api/internal/domain"
	"github.com/ihsanuta/task-management-api/internal/repository/mock"
	"github.com/ihsanuta/task-management-api/internal/usecase"
)

func userDomainFixture(id, teamID string) *domain.User {
	return &domain.User{
		ID:           id,
		Name:         "Fixture User",
		Email:        id + "@example.com",
		PasswordHash: "irrelevant-for-this-test",
		TeamID:       teamID,
	}
}

func newTaskUC() (*usecase.TaskUsecase, *mock.TaskRepository) {
	taskRepo := mock.NewTaskRepository()
	userRepo := mock.NewUserRepository()
	idemRepo := mock.NewIdempotencyRepository()
	uc := usecase.NewTaskUsecase(taskRepo, userRepo, idemRepo, 24*time.Hour)
	return uc, taskRepo
}

const (
	testOwnerID = "owner-1"
	testTeamID  = "team-1"
)

// --- Basic CRUD sanity (not the focus of the assignment, but cheap to add) ---

func TestCreateTask_WithoutIdempotencyKey(t *testing.T) {
	uc, repo := newTaskUC()
	res, err := uc.CreateTask(context.Background(), testOwnerID, testTeamID, dto.CreateTaskRequest{Title: "Write report"}, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Status != http.StatusCreated {
		t.Fatalf("expected 201, got %d", res.Status)
	}
	if repo.CreateCalls != 1 {
		t.Fatalf("expected exactly 1 task created, got %d", repo.CreateCalls)
	}
}

func TestCreateTask_MissingTitle(t *testing.T) {
	uc, _ := newTaskUC()
	_, err := uc.CreateTask(context.Background(), testOwnerID, testTeamID, dto.CreateTaskRequest{}, "")
	if err == nil {
		t.Fatal("expected validation error for missing title")
	}
}

func TestCreateTask_InvalidIdempotencyKeyFormat(t *testing.T) {
	uc, _ := newTaskUC()
	_, err := uc.CreateTask(context.Background(), testOwnerID, testTeamID, dto.CreateTaskRequest{Title: "X"}, "not-a-uuid")
	if err == nil {
		t.Fatal("expected error for malformed Idempotency-Key")
	}
}

// --- 5.1 Race Condition: Idempotency ---

// TestIdempotency_Sequential verifies that a second request sent with the
// same Idempotency-Key *after* the first has completed does not create a
// new task and instead returns an identical response.
func TestIdempotency_Sequential(t *testing.T) {
	uc, repo := newTaskUC()
	ctx := context.Background()
	key := uuid.NewString()
	req := dto.CreateTaskRequest{Title: "Ship the release", Description: "v1.0"}

	first, err := uc.CreateTask(ctx, testOwnerID, testTeamID, req, key)
	if err != nil {
		t.Fatalf("first request should succeed: %v", err)
	}
	if first.Status != http.StatusCreated || first.Replayed {
		t.Fatalf("expected a fresh 201 creation, got status=%d replayed=%v", first.Status, first.Replayed)
	}

	second, err := uc.CreateTask(ctx, testOwnerID, testTeamID, req, key)
	if err != nil {
		t.Fatalf("second (duplicate) request should succeed via replay: %v", err)
	}
	if !second.Replayed {
		t.Fatal("expected second request to be flagged as a replay")
	}
	if second.Task.ID != first.Task.ID {
		t.Fatalf("expected identical task id, got first=%s second=%s", first.Task.ID, second.Task.ID)
	}
	if second.Status != first.Status {
		t.Fatalf("expected identical status code, got first=%d second=%d", first.Status, second.Status)
	}

	if repo.CreateCalls != 1 {
		t.Fatalf("expected exactly 1 task to be created in the database, got %d", repo.CreateCalls)
	}
}

// TestIdempotency_SameKeyDifferentPayload verifies reusing a key with a
// different request body is rejected rather than silently accepted.
func TestIdempotency_SameKeyDifferentPayload(t *testing.T) {
	uc, _ := newTaskUC()
	ctx := context.Background()
	key := uuid.NewString()

	_, err := uc.CreateTask(ctx, testOwnerID, testTeamID, dto.CreateTaskRequest{Title: "Task A"}, key)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	_, err = uc.CreateTask(ctx, testOwnerID, testTeamID, dto.CreateTaskRequest{Title: "Task B (different)"}, key)
	if err == nil {
		t.Fatal("expected an error when the same idempotency key is reused with a different payload")
	}
}

// TestIdempotency_ConcurrentDuplicate is the critical race-condition proof
// required by the assignment: N goroutines fire the *same* Idempotency-Key
// simultaneously. Regardless of scheduling, exactly one task must be
// created — every other goroutine must either receive the replayed
// response or a 409 "in progress" conflict, never a second task.
func TestIdempotency_ConcurrentDuplicate(t *testing.T) {
	uc, repo := newTaskUC()
	ctx := context.Background()
	key := uuid.NewString()
	req := dto.CreateTaskRequest{Title: "Concurrent create", Description: "race test"}

	const n = 100
	var wg sync.WaitGroup
	results := make([]*usecase.IdempotentResult, n)
	errs := make([]error, n)

	// Use a start barrier so all goroutines hit CreateTask as close to
	// simultaneously as possible, maximizing the chance of exposing a
	// race condition if one existed.
	start := make(chan struct{})
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			<-start
			res, err := uc.CreateTask(ctx, testOwnerID, testTeamID, req, key)
			results[i] = res
			errs[i] = err
		}(i)
	}
	close(start)
	wg.Wait()

	if repo.CreateCalls != 1 {
		t.Fatalf("race condition detected: expected exactly 1 task created across %d concurrent requests, got %d", n, repo.CreateCalls)
	}

	successCount := 0
	for i := 0; i < n; i++ {
		if errs[i] == nil && results[i] != nil {
			successCount++
		}
	}
	if successCount == 0 {
		t.Fatal("expected at least one request to succeed")
	}

	// Every successful response (fresh creation or replay) must reference
	// the same underlying task id.
	var referenceID string
	for i := 0; i < n; i++ {
		if errs[i] != nil || results[i] == nil {
			continue
		}
		if referenceID == "" {
			referenceID = results[i].Task.ID
			continue
		}
		if results[i].Task.ID != referenceID {
			t.Fatalf("inconsistent task id across concurrent responses: %s vs %s", referenceID, results[i].Task.ID)
		}
	}
}

// TestIdempotency_ConcurrentDuplicate_HighContention re-runs the race test
// multiple times to reduce the chance of a flaky pass hiding a real race;
// combined with `go test -race` this gives strong confidence the
// claim-then-create path has no data race.
func TestIdempotency_ConcurrentDuplicate_HighContention(t *testing.T) {
	for run := 0; run < 5; run++ {
		uc, repo := newTaskUC()
		ctx := context.Background()
		key := uuid.NewString()
		req := dto.CreateTaskRequest{Title: "Repeated race test"}

		var wg sync.WaitGroup
		const n = 50
		for i := 0; i < n; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				_, _ = uc.CreateTask(ctx, testOwnerID, testTeamID, req, key)
			}()
		}
		wg.Wait()

		if repo.CreateCalls != 1 {
			t.Fatalf("run %d: expected exactly 1 task created, got %d", run, repo.CreateCalls)
		}
	}
}

// --- Database transaction integrity (AssignTask) ---

func TestAssignTask_Success(t *testing.T) {
	taskRepo := mock.NewTaskRepository()
	userRepo := mock.NewUserRepository()
	idemRepo := mock.NewIdempotencyRepository()
	uc := usecase.NewTaskUsecase(taskRepo, userRepo, idemRepo, time.Hour)
	ctx := context.Background()

	_ = userRepo.Create(ctx, userDomainFixture("assignee-1", testTeamID))

	created, err := uc.CreateTask(ctx, testOwnerID, testTeamID, dto.CreateTaskRequest{Title: "Assign me"}, "")
	if err != nil {
		t.Fatalf("setup failed: %v", err)
	}

	updated, err := uc.AssignTask(ctx, testOwnerID, testTeamID, created.Task.ID, "assignee-1")
	if err != nil {
		t.Fatalf("assign should succeed: %v", err)
	}
	if updated.AssigneeID == nil || *updated.AssigneeID != "assignee-1" {
		t.Fatalf("expected task to be assigned to assignee-1, got %v", updated.AssigneeID)
	}
}

func TestAssignTask_RejectsAssigneeOutsideTeam(t *testing.T) {
	taskRepo := mock.NewTaskRepository()
	userRepo := mock.NewUserRepository()
	idemRepo := mock.NewIdempotencyRepository()
	uc := usecase.NewTaskUsecase(taskRepo, userRepo, idemRepo, time.Hour)
	ctx := context.Background()

	_ = userRepo.Create(ctx, userDomainFixture("outsider-1", "another-team"))

	created, err := uc.CreateTask(ctx, testOwnerID, testTeamID, dto.CreateTaskRequest{Title: "Assign me"}, "")
	if err != nil {
		t.Fatalf("setup failed: %v", err)
	}

	_, err = uc.AssignTask(ctx, testOwnerID, testTeamID, created.Task.ID, "outsider-1")
	if err == nil {
		t.Fatal("expected an error assigning a task to a user outside the team")
	}
}
