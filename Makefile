.PHONY: run build test test-race migrate-up migrate-down docker-up docker-down tidy fmt vet swag

APP_NAME=task-management-api
DB_URL?=postgres://postgres:postgres@localhost:5432/taskdb?sslmode=disable

run:
	go run ./cmd/api

build:
	CGO_ENABLED=0 go build -o bin/api ./cmd/api

test:
	go test ./... -v

test-race:
	go test ./... -race -v

fmt:
	gofmt -w .

vet:
	go vet ./...

tidy:
	go mod tidy

migrate-up:
	psql "$(DB_URL)" -f migrations/000001_init.up.sql

migrate-down:
	psql "$(DB_URL)" -f migrations/000001_init.down.sql

docker-up:
	docker compose up --build -d

docker-down:
	docker compose down -v

swag:
	swag init -g cmd/api/main.go
