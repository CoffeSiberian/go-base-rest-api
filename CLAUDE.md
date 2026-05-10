# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Commands

```bash
# Run locally (requires PostgreSQL)
cp .env.example .env   # set DB creds and JWT_SECRET
go run main.go

# Docker (recommended)
docker-compose up --build

# Build binary
go build -o server .

# Tidy dependencies
go mod tidy
```

No test suite exists yet. No linter config — use `go vet ./...` for basic checks.

## Architecture

Clean layered architecture. Dependency direction is strictly one-way:

```
Controller → Service → Repository → GORM → PostgreSQL
```

**Wiring** happens in `routes/routes.go` — repo, service, and controller are constructed there and injected. No DI framework.

**Request/response types** live in the service layer (`services/`), not in controllers or models.

**Error handling** uses `utils.AppError` (carries HTTP code + error code string). Services return `*AppError` for domain errors or sentinel vars (`utils.ErrInternal`, etc.). Controllers use a helper:

```go
var appErr *utils.AppError
if errors.As(err, &appErr) {
    utils.Error(c, appErr.HTTPCode, appErr.Message, appErr.Code)
} else {
    utils.Error(c, 500, "Internal server error", "INTERNAL_ERROR")
}
```

**Response envelope** — always use `utils.Success(c, data)` / `utils.Error(c, code, msg, errCode)`. Never call `c.JSON` directly in controllers.

**JWT** — `utils.GenerateToken` / `utils.ValidateClaims`. Auth middleware injects `userID` (uint) and `role` (string) into gin context via `c.Set`. Downstream handlers read with `c.GetUint("userID")` / `c.GetString("role")`.

**AutoMigrate** — add new models to the `db.AutoMigrate(...)` call in `database/database.go`.

## Adding a new module

1. `models/foo.go` — GORM struct, add to AutoMigrate
2. `repositories/foo_repository.go` — interface + GORM impl
3. `services/foo_service.go` — request/response types, business logic
4. `controllers/foo_controller.go` — bind request, call service, write response
5. `routes/routes.go` — construct and wire, register route group

## Key env vars

`JWT_SECRET` is required — server fatals on startup if missing. All others have defaults (port 8080, DB `school_system` on localhost:5432).

To test protected endpoints without a login endpoint, generate a token manually:

```go
token, _ := utils.GenerateToken(1, "admin", cfg.JWTSecret, 24*time.Hour)
```
