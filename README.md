# Task Management API (Multi-User)

A REST API for multi-user task management built in Go

## Tech stack

- Go 1.25 (stdlib `net/http` + `http.ServeMux` method/path routing — no web framework)
- PostgreSQL (via `database/sql` + `github.com/lib/pq`)
- `github.com/golang-jwt/jwt/v5` for JWT
- `github.com/google/uuid` for UUIDs
- Password hashing via PBKDF2-HMAC-SHA256 (stdlib `crypto/*` only)
- `log/slog` for structured JSON logging
- `github.com/go-playground/validator/v10` for Validation

No web framework, ORM, or validation library is used on purpose, to keep the
dependency surface small and every request path easy to trace end-to-end.

---

## 1. Running the project

### Option A — Docker Compose (recommended, zero local setup)

```bash
cp .env.example .env   # optional, docker-compose.yml already has working defaults
docker compose up -d
```

This starts Postgres, applies nothing automatically — run the migration once the
DB is up (compose does not auto-migrate, see below), then the API is available at
`http://localhost:8080`.

### Option B — Local Go toolchain

Prerequisites: Go 1.22+, a running PostgreSQL instance.

```bash
# 1. Configure environment
cp .env.example .env
# edit .env if your Postgres isn't on localhost:5432

# 2. Install dependencies
go mod tidy

# 3. Create the database and apply the schema
s
make migrate-up            # or: psql "$DATABASE_URL" -f migrations/000001_init.up.sql

# 4. Run the API
make run                   # or: go run ./cmd/api
```

The server listens on `APP_PORT` (default `8080`).

### Running tests

```bash
make test-race    # go test ./... -race -v
```

Unit tests run entirely against in-memory mock repositories — **no database or
network connection is required**. The idempotency race-condition tests in
`internal/usecase/task_usecase_test.go` are run with `-race` to prove there is
no data race in the claim-then-create path.

For testing api with swagger, the API is available at
`http://localhost:8080/swagger/index.html` 

---

## 2. Authentication

| Method | Endpoint         | Description                          |
|--------|------------------|---------------------------------------|
| POST   | `/auth/register` | Create a user (and JWT)               |
| POST   | `/auth/login`    | Authenticate and receive a JWT        |

`POST /auth/register`
```json
{ "name": "Alice", "email": "alice@example.com", "password": "supersecret1" }
```
`team_id` is optional — omit it to have a new team created automatically for
the user (used later by the "assign task" feature, which requires the
assignee to be in the same team).

Response (both endpoints):
```json
{
  "status": "success",
  "data": {
    "token": "<jwt>",
    "user": { "id": "...", "name": "Alice", "email": "alice@example.com", "team_id": "..." }
  },
  "timestamp": "2026-07-20T05:00:00Z"
}
```

All `/tasks*` endpoints require `Authorization: Bearer <token>`.

---

## 3. API Endpoints

| Method | Endpoint                | Auth | Description                                  |
|--------|--------------------------|------|-----------------------------------------------|
| POST   | `/tasks`                 | ✅   | Create task (supports `Idempotency-Key`)      |
| GET    | `/tasks`                 | ✅   | List tasks (filter, search, pagination)       |
| GET    | `/tasks/{id}`             | ✅   | Task detail                                   |
| PUT    | `/tasks/{id}`             | ✅   | Update task                                   |
| DELETE | `/tasks/{id}`             | ✅   | Delete task                                   |
| POST   | `/tasks/{id}/assign`      | ✅   | Assign task to another user in the same team  |
| GET    | `/healthz`               | ❌   | Liveness check                                |

A task is visible to a user if they are its **owner** or its **assignee**, and
it belongs to their team. Only the **owner** can delete a task; owner or
assignee can update it.

### Create task
```
POST /tasks
Authorization: Bearer <token>
Idempotency-Key: 3fa85f64-5717-4562-b3fc-2c963f66afa6
Content-Type: application/json

{ "title": "Write proposal", "description": "Q3 roadmap", "status": "pending" }
```
`status` is optional (defaults to `pending`); one of `pending`, `in_progress`, `done`.

### List tasks
```
GET /tasks?status=pending&search=proposal&page=1&limit=10
```
- `status` — exact match on task status (optional)
- `search` — case-insensitive substring match on title (optional)
- `page`, `limit` — pagination, defaults `page=1`, `limit=10`, `limit` capped at 100

Response includes pagination `meta`:
```json
{
  "status": "success",
  "data": [ { "id": "...", "title": "..." } ],
  "meta": { "page": 1, "limit": 10, "total_items": 23, "total_pages": 3 },
  "timestamp": "..."
}
```

### Assign task
```
POST /tasks/{id}/assign
{ "assignee_id": "<user-id-in-same-team>" }
```
Runs update + audit log + (mocked) notification inside one DB transaction — see
[Database transaction integrity](#database-transaction-integrity) below.

---

## 4. Architecture

```
cmd/api/main.go            → composition root: config, DB, repos, usecases, router, graceful shutdown
internal/
  config/                  → env-based configuration
  domain/                  → plain entities (User, Team, Task, TaskLog, IdempotencyRecord)
  repository/              → interfaces (ports)
    postgres/               → Postgres implementations
    mock/                   → in-memory, thread-safe fakes used by unit tests
  usecase/                 → business logic (AuthUsecase, TaskUsecase) — depends only on
                              repository interfaces, not concrete implementations
  delivery/http/
    handler/                → thin HTTP handlers: decode → call usecase → encode
    middleware/              → RequestID, Logger, Recover, Auth (JWT)
    dto/                     → request/response shapes, decoupled from domain entities
    router.go                → route table + middleware chain wiring
pkg/
  apperror/                → single structured error type + shared error codes
  response/                → consistent JSON success/error envelope
  jwtutil/                 → JWT issue/verify
  pwhash/                  → PBKDF2 password hashing
  logger/                  → slog JSON logger setup
migrations/                → raw SQL schema (up/down)
```
