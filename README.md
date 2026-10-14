# Tasks API

[![CI](https://github.com/dangyyyy/ApiProject/actions/workflows/ci.yml/badge.svg)](https://github.com/dangyyyy/ApiProject/actions/workflows/ci.yml)

REST API for managing tasks, written in Go using only the standard library router and PostgreSQL.

## Features

- CRUD for tasks with input validation
- Pagination, filtering, search and sorting: `GET /tasks?completed=true&search=go&sort=title&order=asc&limit=20&offset=0`
- Database migrations with [goose](https://github.com/pressly/goose), applied automatically on startup
- Structured logging with `log/slog` (text or JSON) and a request ID in every log line
- Middleware: request ID, access log, panic recovery, JSON errors for 404/405
- Timeouts at every level: HTTP server, database queries, startup connection
- Graceful shutdown on `SIGINT` / `SIGTERM`
- Configuration via environment variables and a `.env` file
- Multi-stage Docker image (~24 MB) running as a non-root user
- Unit tests for handlers, middleware, models and config; CI with tests, linter and Docker build

## Tech stack

| | |
|---|---|
| Language | Go 1.26 |
| HTTP | `net/http` (`ServeMux` with method and path patterns) |
| Database | PostgreSQL 15, [sqlx](https://github.com/jmoiron/sqlx), [lib/pq](https://github.com/lib/pq) |
| Migrations | [goose](https://github.com/pressly/goose) |
| Logging | `log/slog` |
| Infrastructure | Docker, Docker Compose, GitHub Actions, golangci-lint |

## Quick start

Requirements: Docker with Docker Compose.

```bash
git clone https://github.com/dangyyyy/ApiProject.git
cd ApiProject
cp .env.example .env
docker compose up -d --build
```

Set your own `DB_PASSWORD` in `.env` before the first start.

Check that it works:

```bash
curl localhost:8080/healthz
```

## API

| Method | Path | Description | Success |
|---|---|---|---|
| `GET` | `/healthz` | Liveness check | `200` |
| `GET` | `/tasks` | List tasks with filters and pagination | `200` |
| `POST` | `/tasks` | Create a task | `201` |
| `GET` | `/tasks/{id}` | Get a task by ID | `200` |
| `PUT` | `/tasks/{id}` | Partially update a task | `200` |
| `DELETE` | `/tasks/{id}` | Delete a task | `200` |

### List query parameters

| Parameter | Values | Default |
|---|---|---|
| `completed` | `true`, `false` | no filter |
| `search` | substring of the title, case-insensitive, up to 100 characters | no search |
| `sort` | `created_at`, `title` | `created_at` |
| `order` | `asc`, `desc` | `desc` |
| `limit` | 1–100 | 20 |
| `offset` | 0 or more | 0 |

### Examples

Create a task:

```bash
curl -X POST localhost:8080/tasks \
  -H "Content-Type: application/json" \
  -d '{"title": "Learn Go", "description": "Finish the REST API"}'
```

```json
{
  "id": 3,
  "title": "Learn Go",
  "description": "Finish the REST API",
  "completed": false,
  "created_at": "2026-10-14T15:00:00Z",
  "updated_at": "2026-10-14T15:00:00Z"
}
```

List completed tasks, two per page:

```bash
curl "localhost:8080/tasks?completed=true&limit=2"
```

```json
{
  "items": [ ... ],
  "total": 7,
  "limit": 2,
  "offset": 0
}
```

Mark a task as completed (only the fields you send are changed):

```bash
curl -X PUT localhost:8080/tasks/3 \
  -H "Content-Type: application/json" \
  -d '{"completed": true}'
```

### Errors

All errors are returned as JSON with an appropriate status code:

```json
{"error": "task not found"}
```

| Status | When |
|---|---|
| `400` | Invalid JSON, unknown field, validation error, bad query parameter |
| `404` | Task or route not found |
| `405` | Method not allowed (the `Allow` header lists allowed methods) |
| `500` | Unexpected server error (details are only logged, never sent to the client) |
| `504` | Database query timed out |

Every response has an `X-Request-ID` header. Send your own `X-Request-ID` to trace a request across services.

## Configuration

| Variable | Default | Description |
|---|---|---|
| `SERVER_PORT` | `8080` | HTTP port |
| `LOG_FORMAT` | `text` | `text` or `json` |
| `LOG_LEVEL` | `info` | `debug`, `info`, `warn`, `error` |
| `DB_HOST` | `localhost` | PostgreSQL host |
| `DB_PORT` | `5432` | PostgreSQL port |
| `DB_USER` | — | **required** |
| `DB_PASSWORD` | — | **required** |
| `DB_NAME` | — | **required** |
| `DB_SSLMODE` | `disable` | PostgreSQL SSL mode |

Environment variables take precedence over values in `.env`.

## Project structure

```
.
├── cmd/api/              entry point: config, wiring, graceful shutdown
├── internal/
│   ├── config/           configuration from environment variables
│   ├── database/         PostgreSQL store, connection, migrations runner
│   ├── handlers/         HTTP handlers, routing, request parsing
│   ├── logger/           slog handler that adds request ID from context
│   ├── middleware/       request ID, logging, recover, JSON errors
│   └── models/           domain types and input validation
├── migrations/           SQL migrations embedded into the binary
├── Dockerfile
└── docker-compose.yml
```

Request flow:

```
HTTP request
  → RequestID → Logging → Recover → JSONErrors     (middleware)
  → ServeMux → Handler                              (parse, validate)
  → TaskStore interface → PostgreSQL                (with query timeout)
```

Handlers depend on a `TaskStore` interface, not on the database directly, so they are tested with an in-memory fake.

## Development

Run only the database in Docker and the API locally:

```bash
docker compose up -d postgres
go run ./cmd/api
```

Tests and linter:

```bash
go test ./... -race -cover
golangci-lint run
```

Migrations are applied automatically on startup. To manage them by hand, install the [goose CLI](https://github.com/pressly/goose#install):

```bash
export GOOSE_DRIVER=postgres GOOSE_DBSTRING="postgres://taskuser:<password>@localhost:5432/tasksdb?sslmode=disable"
goose -dir migrations status
goose -dir migrations -s create add_something sql
```
