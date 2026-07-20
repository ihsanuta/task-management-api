package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/ihsanuta/task-management-api/internal/config"
	apphttp "github.com/ihsanuta/task-management-api/internal/delivery/http"
	"github.com/ihsanuta/task-management-api/internal/delivery/http/handler"
	"github.com/ihsanuta/task-management-api/internal/repository/postgres"
	"github.com/ihsanuta/task-management-api/internal/usecase"
	"github.com/ihsanuta/task-management-api/pkg/jwtutil"
	applog "github.com/ihsanuta/task-management-api/pkg/logger"
)

func main() {
	cfg := config.Load()
	log := applog.New(cfg.Env)

	db, err := postgres.Connect(cfg.DatabaseURL)
	if err != nil {
		log.Error("failed to connect to database", "error", err)
		os.Exit(1)
	}
	defer db.Close()

	// Validator
	validator := validator.New()

	// Repositories
	userRepo := postgres.NewUserRepository(db)
	teamRepo := postgres.NewTeamRepository(db)
	taskRepo := postgres.NewTaskRepository(db)
	idemRepo := postgres.NewIdempotencyRepository(db)

	jwtManager := jwtutil.NewManager(cfg.JWTSecret, cfg.JWTExpiry)
	authUC := usecase.NewAuthUsecase(userRepo, teamRepo, jwtManager)
	taskUC := usecase.NewTaskUsecase(taskRepo, userRepo, idemRepo, cfg.IdempotencyTTL)

	handlers := apphttp.Handlers{
		Auth: handler.NewAuthHandler(authUC, validator),
		Task: handler.NewTaskHandler(taskUC, validator),
	}
	router := apphttp.NewRouter(handlers, jwtManager)

	srv := &http.Server{
		Addr:         ":" + cfg.AppPort,
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		log.Info("server starting", "port", cfg.AppPort, "env", cfg.Env)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Error("server failed", "error", err)
			os.Exit(1)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info("shutting down server...")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Error("forced shutdown", "error", err)
	}
	slog.Info("server exited cleanly")
}
