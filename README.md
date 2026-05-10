# System REST API

REST API system built with Go and Gin. Designed for high concurrency — stateless JWT auth, PostgreSQL connection pooling, layered clean architecture.

## Tech Stack

| Component      | Technology       |
| -------------- | ---------------- |
| Language       | Go 1.21+         |
| Framework      | Gin Gonic        |
| Database       | PostgreSQL 16    |
| ORM            | GORM             |
| Authentication | JWT (HS256)      |
| Passwords      | bcrypt           |
| Config         | godotenv         |
| Container      | Docker + Compose |

## Architecture

```
/
├── main.go              # Entry point — wires config, DB, router, graceful shutdown
├── config/              # Loads .env into Config struct
├── database/            # PostgreSQL connection + pool config + AutoMigrate
├── models/              # GORM entity definitions
├── repositories/        # Data layer — raw DB queries via GORM
├── services/            # Business logic — validation, rules, orchestration
├── controllers/         # HTTP layer — parses request, calls service, writes response
├── middlewares/         # Auth (JWT), CORS, Logger
├── routes/              # Registers all endpoints and applies middlewares
└── utils/               # JWT helpers, response envelope, error types
```

### Request flow

```
Client → Router → Middleware(s) → Controller → Service → Repository → PostgreSQL
                                                                ↑
                                                   Business rules here
```

## Prerequisites

- Go 1.21+
- Docker + Docker Compose

## Setup

### Option A — Docker (recommended)

```bash
cp .env.example .env
# Edit .env with your values
docker-compose up --build
```

### Option B — Local

```bash
# Requires a running PostgreSQL instance
cp .env.example .env
# Edit .env with your DB credentials
go mod download
go run main.go
```

The server auto-runs database migrations on startup.

## Environment Variables

| Variable      | Default         | Required | Description                 |
| ------------- | --------------- | -------- | --------------------------- |
| `APP_PORT`    | `8080`          | No       | HTTP server port            |
| `APP_ENV`     | `development`   | No       | Environment label           |
| `DB_HOST`     | `localhost`     | No       | PostgreSQL host             |
| `DB_PORT`     | `5432`          | No       | PostgreSQL port             |
| `DB_USER`     | `postgres`      | No       | Database user               |
| `DB_PASSWORD` | —               | No       | Database password           |
| `DB_NAME`     | `school_system` | No       | Database name               |
| `JWT_SECRET`  | —               | **Yes**  | Secret key for signing JWTs |

## API Endpoints

### Response envelope

All endpoints return a consistent JSON envelope:

```json
// Success
{ "success": true, "data": { ... } }

// Error
{ "success": false, "error": "message", "code": "ERROR_CODE" }
```

### Health

| Method | Path             | Auth | Description  |
| ------ | ---------------- | ---- | ------------ |
| GET    | `/api/v1/health` | No   | Health check |

```bash
curl http://localhost:8080/api/v1/health
# {"success":true,"data":{"status":"ok"}}
```

### Users

| Method | Path            | Auth | Description     |
| ------ | --------------- | ---- | --------------- |
| POST   | `/api/v1/users` | Yes  | Create new user |

**POST /api/v1/users**

```bash
curl -X POST http://localhost:8080/api/v1/users \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{
    "name":     "John Doe",
    "email":    "john@school.com",
    "password": "secret123",
    "role":     "student"
  }'
```

Valid roles: `admin`, `teacher`, `student`, `parent`

```json
// 201 Created
{
    "success": true,
    "data": {
        "id": 1,
        "name": "John Doe",
        "email": "john@school.com",
        "role": "student",
        "created_at": "2026-05-10T14:00:00Z"
    }
}
```

| Status | Code               | Reason                   |
| ------ | ------------------ | ------------------------ |
| 201    | —                  | User created             |
| 401    | `UNAUTHORIZED`     | Missing or invalid JWT   |
| 409    | `EMAIL_TAKEN`      | Email already registered |
| 422    | `INVALID_ROLE`     | Role not in allowed list |
| 422    | `VALIDATION_ERROR` | Missing/malformed fields |

## Authentication

Endpoints marked **Auth: Yes** require a Bearer token in the `Authorization` header:

```
Authorization: Bearer <jwt>
```

JWTs are signed with HS256 using `JWT_SECRET`. Each token carries `user_id` and `role` claims, which are injected into the request context by `middlewares/auth.go` and available to any downstream handler.

> A `/auth/login` endpoint is not yet implemented. To test protected endpoints during development, generate a token manually using `utils.GenerateToken`.

## Adding a New Module

Follow this pattern for each new entity (e.g., `Course`):

1. **Model** — `models/course.go` — GORM struct with indexes
2. **Repository** — `repositories/course_repository.go` — interface + GORM implementation
3. **Service** — `services/course_service.go` — business logic, request/response types
4. **Controller** — `controllers/course_controller.go` — HTTP binding + service call
5. **Routes** — wire in `routes/routes.go`:

    ```go
    courseRepo := repositories.NewCourseRepository(db)
    courseSvc  := services.NewCourseService(courseRepo)
    courseCtrl := controllers.NewCourseController(courseSvc)

    courses := api.Group("/courses", middlewares.Auth(cfg.JWTSecret))
    courses.GET("",     courseCtrl.List)
    courses.POST("",    courseCtrl.Create)
    courses.GET("/:id", courseCtrl.GetByID)
    ```

6. **AutoMigrate** — add `&models.Course{}` to `database/database.go`
