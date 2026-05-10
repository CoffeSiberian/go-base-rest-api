# Guía Completa del Proyecto: Management API

> **Para quién es esto:** Developers que son nuevos en Go y quieren entender este proyecto, agregar módulos nuevos y escalarlo de forma profesional.

---

## Tabla de Contenidos

1. [¿Qué es este proyecto?](#1-qué-es-este-proyecto)
2. [Diagrama de arquitectura](#2-diagrama-de-arquitectura)
3. [Árbol de carpetas explicado](#3-árbol-de-carpetas-explicado)
4. [Flujo de una request de principio a fin](#4-flujo-de-una-request-de-principio-a-fin)
5. [Go básico para trabajar en este proyecto](#5-go-básico-para-trabajar-en-este-proyecto)
6. [Patrones clave del proyecto](#6-patrones-clave-del-proyecto)
7. [Cómo agregar un módulo nuevo: "100 Points"](#7-cómo-agregar-un-módulo-nuevo-100-points)
8. [Cómo escalar el proyecto profesionalmente](#8-cómo-escalar-el-proyecto-profesionalmente)
9. [Comandos rápidos de referencia](#9-comandos-rápidos-de-referencia)

---

## 1. ¿Qué es este proyecto?

Es un **backend REST API** para un sistema de gestión escolar. Está construido en Go usando el framework **Gin**, con **PostgreSQL** como base de datos y **GORM** como ORM (Object-Relational Mapper).

### Stack tecnológico

| Componente     | Tecnología              | Para qué sirve                              |
| -------------- | ----------------------- | ------------------------------------------- |
| Lenguaje       | Go 1.22+                | Compilado, tipado estático, muy performante |
| HTTP Framework | Gin                     | Routing, middlewares, binding de JSON       |
| ORM            | GORM                    | Mapeo de structs Go a tablas de PostgreSQL  |
| Base de datos  | PostgreSQL 16           | Almacenamiento principal                    |
| Autenticación  | JWT (HS256)             | Tokens de acceso stateless                  |
| Contenedores   | Docker + docker-compose | Desarrollo local sin instalar Postgres      |
| Config         | godotenv                | Lee `.env` y variables de entorno           |

### Endpoints actuales

```
GET  /api/v1/health       → Verifica que la API está viva (público)
POST /api/v1/users        → Crea un usuario (requiere JWT válido)
```

### Roles de usuario en el sistema

```
admin     → Administrador total
teacher   → Docente
student   → Estudiante
parent    → Padre/tutor
```

---

## 2. Diagrama de arquitectura

### Flujo general de una request

```
                          INTERNET
                              │
                              ▼
                    ┌─────────────────┐
                    │   HTTP Request   │
                    │  POST /api/v1/  │
                    │     /users       │
                    └────────┬────────┘
                             │
                    ┌────────▼────────┐
                    │   Middlewares   │
                    │                 │
                    │  ┌───────────┐  │
                    │  │   CORS    │  │  Permite requests de otros orígenes
                    │  └─────┬─────┘  │
                    │        │        │
                    │  ┌─────▼─────┐  │
                    │  │   Auth    │  │  Valida el JWT Bearer token
                    │  │(JWT check)│  │  Inyecta userID y role en el contexto
                    │  └─────┬─────┘  │
                    └────────┼────────┘
                             │
                    ┌────────▼────────┐
                    │   Controller    │
                    │ user_controller │  Bind JSON → llama al service
                    │      .go        │  Maneja errores → escribe respuesta
                    └────────┬────────┘
                             │ llama a
                    ┌────────▼────────┐
                    │    Service      │
                    │ user_service.go │  Lógica de negocio
                    │                 │  Validaciones (role válido, etc.)
                    │                 │  Hasheo de password (bcrypt)
                    └────────┬────────┘
                             │ llama a
                    ┌────────▼────────┐
                    │   Repository    │
                    │user_repository  │  Solo operaciones de datos
                    │      .go        │  Create, FindByEmail, FindByID
                    └────────┬────────┘
                             │ usa
                    ┌────────▼────────┐
                    │     GORM        │
                    │   (ORM layer)   │  Convierte structs Go a SQL
                    └────────┬────────┘
                             │
                    ┌────────▼────────┐
                    │   PostgreSQL    │
                    │   (Database)    │  Almacenamiento persistente
                    └─────────────────┘
```

### Dirección de dependencias (IMPORTANTE)

```
Controller  →  Service  →  Repository  →  GORM  →  PostgreSQL

La dependencia SIEMPRE va hacia la derecha.
El Controller no conoce a Repository.
El Service no conoce a Gin ni a la DB directamente.
```

Esto es lo que hace el código mantenible: cada capa solo conoce a la siguiente.

---

## 3. Árbol de carpetas explicado

```
gin-hola-mundo/
│
├── main.go                    ← Punto de entrada. Arranca config, DB y servidor HTTP.
│
├── config/
│   └── config.go              ← Lee variables de entorno (.env). JWT_SECRET es obligatorio.
│
├── database/
│   └── database.go            ← Conecta a PostgreSQL via GORM. Corre AutoMigrate.
│                                 ⚠️ Registra aquí cada nuevo modelo que crees.
│
├── models/                    ← Structs de GORM = tablas de la base de datos.
│   └── user.go                   Solo definición de columnas, sin lógica.
│
├── repositories/              ← Capa de datos pura. Interfaz + implementación con GORM.
│   └── user_repository.go        Sin reglas de negocio, solo queries.
│
├── services/                  ← Lógica de negocio. También define los tipos
│   └── user_service.go           Request y Response de cada operación.
│
├── controllers/               ← Recibe HTTP, llama al service, responde JSON.
│   ├── health_controller.go      No tiene lógica de negocio.
│   └── user_controller.go
│
├── middlewares/               ← Funciones que se ejecutan antes del controller.
│   ├── auth.go                   Valida JWT.
│   ├── cors.go                   Headers de CORS.
│   └── logger.go                 Log de requests (disponible pero no activado aún).
│
├── routes/
│   └── routes.go              ← Punto de cableado (wiring). Construye repo → service →
│                                 controller y registra todas las rutas.
│
├── utils/
│   ├── errors.go              ← AppError (tipo con HTTPCode + Code + Message) y sentinels.
│   ├── jwt.go                 ← GenerateToken y ValidateClaims.
│   └── response.go            ← utils.Success(c, data) y utils.Error(c, code, msg, errCode).
│
├── docs/
│   └── GUIDE.md               ← Este archivo.
│
├── .env.example               ← Plantilla de variables de entorno.
├── Dockerfile                 ← Build de la imagen Go.
└── docker-compose.yml         ← Levanta API + PostgreSQL juntos.
```

---

## 4. Flujo de una request de principio a fin

Ejemplo concreto: `POST /api/v1/users` con body `{"name":"Ana","email":"ana@school.com","password":"secret123","role":"student"}`

### Paso 1 — `routes/routes.go` registra la ruta

```go
users := api.Group("/users", middlewares.Auth(cfg.JWTSecret))
users.POST("", userCtrl.Create)
// Gin sabe: POST /api/v1/users → pasar por Auth → llamar userCtrl.Create
```

### Paso 2 — `middlewares/auth.go` valida el JWT

```go
header := c.GetHeader("Authorization")  // "Bearer eyJ..."
claims, err := utils.ValidateClaims(token, secret)
c.Set("userID", claims.UserID)          // inyecta en contexto
c.Set("role", claims.Role)
c.Next()                                 // pasa al controller
```

### Paso 3 — `controllers/user_controller.go` recibe la request

```go
var req services.CreateUserRequest
c.ShouldBindJSON(&req)                  // deserializa JSON y valida con binding tags
user, err := uc.service.CreateUser(...)  // delega al service
// maneja el error, responde con c.JSON o utils.Error
```

### Paso 4 — `services/user_service.go` ejecuta la lógica

```go
// Verifica que el role sea válido
// Busca si el email ya existe → error CONFLICT si existe
// Hashea el password con bcrypt
// Llama al repository para guardar
// Retorna UserResponse (sin password)
```

### Paso 5 — `repositories/user_repository.go` guarda en la DB

```go
r.db.WithContext(ctx).Create(user).Error
// GORM genera: INSERT INTO users (name, email, password, role) VALUES (...)
```

### Paso 6 — La respuesta sube de vuelta

```
Repository → Service → Controller → HTTP Response 201 Created

{
  "success": true,
  "data": {
    "id": 1,
    "name": "Ana",
    "email": "ana@school.com",
    "role": "student",
    "created_at": "2026-05-10T12:00:00Z"
  }
}
```

---

## 5. Go básico para trabajar en este proyecto

### 5.1 Paquetes e imports

En Go cada carpeta es un **package**. El nombre del package se declara al inicio del archivo:

```go
package services   // este archivo pertenece al package "services"
```

Para usar código de otro package se importa su ruta de módulo:

```go
import (
    "gin-hola-mundo/models"       // tu propio código
    "gorm.io/gorm"                // dependencia externa
    "github.com/gin-gonic/gin"    // otro package externo
)
```

### 5.2 Structs — las "clases" de Go

Go no tiene clases. Usa **structs** para agrupar datos y métodos:

```go
// Definición del struct (en models/user.go)
type User struct {
    gorm.Model              // embed: agrega ID, CreatedAt, UpdatedAt, DeletedAt
    Name     string
    Email    string
    Password string
    Role     string
}

// Crear una instancia
user := &models.User{
    Name:  "Ana",
    Email: "ana@school.com",
}

// Acceder a campos
fmt.Println(user.Name)  // "Ana"
```

### 5.3 Interfaces — contratos sin herencia

Una **interface** define qué métodos debe tener un tipo, sin importar cómo los implementa:

```go
// Contrato: cualquier cosa que implemente estas funciones es un UserRepository
type UserRepository interface {
    Create(ctx context.Context, user *models.User) error
    FindByEmail(ctx context.Context, email string) (*models.User, error)
    FindByID(ctx context.Context, id uint) (*models.User, error)
}
```

¿Por qué es útil? Porque el Service solo conoce la **interface**, no la implementación real. Esto permite cambiar GORM por otro ORM o usar un mock en los tests sin tocar el service.

```go
type userService struct {
    repo repositories.UserRepository  // ← interface, no la impl concreta
}
```

### 5.4 Punteros (`*` y `&`)

```go
// * en una declaración de tipo → "puntero a T"
var u *models.User       // puntero a un User (puede ser nil)
func (s *userService)    // el receiver es un puntero al struct

// & → "dame la dirección de memoria de esta variable"
user := models.User{Name: "Ana"}
ptr := &user             // ptr ahora apunta al mismo User en memoria

// * en una variable → "desreferencia, dame el valor que apunta"
fmt.Println((*ptr).Name) // "Ana" (equivalente a ptr.Name, Go lo hace automático)
```

**Regla práctica:** cuando una función retorna `*User` y `error`, siempre verifica el error primero antes de usar el valor.

### 5.5 Manejo de errores

Go no tiene excepciones. Los errores son valores:

```go
user, err := repo.FindByEmail(ctx, email)
if err != nil {
    // algo salió mal, no uses user
    return nil, err
}
// aquí err es nil, user es válido
```

**El patrón `errors.As`** (para errores tipados como `AppError`):

```go
var appErr *utils.AppError
if errors.As(err, &appErr) {
    // err es (o envuelve) un *AppError
    utils.Error(c, appErr.HTTPCode, appErr.Message, appErr.Code)
} else {
    utils.Error(c, 500, "Internal server error", "INTERNAL_ERROR")
}
```

### 5.6 `context.Context`

Todos los métodos del repository reciben un `ctx context.Context`. Sirve para:

- **Cancelación:** si el cliente cierra la conexión, la query se cancela automáticamente
- **Timeouts:** puedes poner un límite de tiempo a una operación de DB
- **Trazabilidad:** en el futuro puedes propagar trace IDs para distributed tracing

Siempre pásalo desde el controller así:

```go
// En el controller, el contexto viene de la request HTTP
user, err := uc.service.CreateUser(c.Request.Context(), req)
```

---

## 6. Patrones clave del proyecto

### 6.1 Response envelope — todas las respuestas tienen la misma forma

**Nunca llames `c.JSON` directamente en un controller.** Usa siempre los helpers:

```go
// Respuesta exitosa → HTTP 200
utils.Success(c, data)

// Respuesta de error → HTTP que tú indiques
utils.Error(c, http.StatusConflict, "email already in use", "EMAIL_TAKEN")
```

Ambas producen un JSON con este formato consistente:

```json
// Éxito
{ "success": true, "data": { ... } }

// Error
{ "success": false, "error": "email already in use", "code": "EMAIL_TAKEN" }
```

### 6.2 AppError — errores con HTTP code integrado

Cuando el service quiere comunicar un error de dominio al controller:

```go
// En el service — retorna error con código HTTP embebido
return nil, &utils.AppError{
    HTTPCode: http.StatusConflict,
    Code:     "EMAIL_TAKEN",
    Message:  "email already in use",
}

// Errores genéricos predefinidos en utils/errors.go
return nil, utils.ErrNotFound      // 404
return nil, utils.ErrUnauthorized  // 401
return nil, utils.ErrForbidden     // 403
return nil, utils.ErrInternal      // 500
```

### 6.3 Leer datos del JWT en el controller

El middleware `Auth` inyecta el `userID` y `role` en el contexto de Gin. Cualquier handler protegido puede leerlos:

```go
func (pc *PointController) Assign(c *gin.Context) {
    callerID   := c.GetUint("userID")    // uint — ID del usuario autenticado
    callerRole := c.GetString("role")    // string — "admin", "teacher", etc.

    if callerRole != "admin" && callerRole != "teacher" {
        utils.Error(c, http.StatusForbidden, "only admins and teachers can assign points", "FORBIDDEN")
        return
    }
    // ...
}
```

### 6.4 Inyección de dependencias en `routes/routes.go`

Este archivo es el único lugar donde se "cablea" todo. El orden siempre es:

```go
// 1. Repository recibe la DB
repo := repositories.NewUserRepository(db)

// 2. Service recibe el repository
svc := services.NewUserService(repo)

// 3. Controller recibe el service
ctrl := controllers.NewUserController(svc)

// 4. Registrar las rutas
api.POST("/users", ctrl.Create)
```

---

## 7. Cómo agregar un módulo nuevo: "100 Points"

Este módulo permite que un `admin` o `teacher` asigne puntos a un `student`. Sigue estos 5 pasos **en orden**. Este es el template exacto para cualquier módulo futuro.

---

### Paso 1 de 5 — Model: `models/point.go`

Crea el archivo:

```go
package models

import "gorm.io/gorm"

type Point struct {
    gorm.Model
    StudentID    uint   `gorm:"not null;index"`
    AssignedByID uint   `gorm:"not null"`
    Amount       int    `gorm:"not null"`
    Reason       string `gorm:"not null"`

    // Relaciones (opcionales pero útiles para preloading)
    Student    User `gorm:"foreignKey:StudentID"`
    AssignedBy User `gorm:"foreignKey:AssignedByID"`
}
```

Luego **registra el modelo en `database/database.go`**, en el `AutoMigrate`:

```go
// Antes:
db.AutoMigrate(&models.User{})

// Después:
db.AutoMigrate(&models.User{}, &models.Point{})
```

GORM creará la tabla `points` en PostgreSQL automáticamente al reiniciar el servidor.

---

### Paso 2 de 5 — Repository: `repositories/point_repository.go`

Define la interfaz primero, luego la implementación:

```go
package repositories

import (
    "context"

    "gin-hola-mundo/models"

    "gorm.io/gorm"
)

// Contrato — qué operaciones de datos necesitamos
type PointRepository interface {
    Create(ctx context.Context, point *models.Point) error
    FindByStudent(ctx context.Context, studentID uint) ([]models.Point, error)
    FindByID(ctx context.Context, id uint) (*models.Point, error)
}

// Implementación con GORM
type pointRepository struct {
    db *gorm.DB
}

func NewPointRepository(db *gorm.DB) PointRepository {
    return &pointRepository{db: db}
}

func (r *pointRepository) Create(ctx context.Context, point *models.Point) error {
    return r.db.WithContext(ctx).Create(point).Error
}

func (r *pointRepository) FindByStudent(ctx context.Context, studentID uint) ([]models.Point, error) {
    var points []models.Point
    err := r.db.WithContext(ctx).Where("student_id = ?", studentID).Find(&points).Error
    return points, err
}

func (r *pointRepository) FindByID(ctx context.Context, id uint) (*models.Point, error) {
    var point models.Point
    err := r.db.WithContext(ctx).First(&point, id).Error
    if err != nil {
        return nil, err
    }
    return &point, nil
}
```

---

### Paso 3 de 5 — Service: `services/point_service.go`

Aquí viven los **tipos de request/response** y la **lógica de negocio**:

```go
package services

import (
    "context"
    "errors"
    "net/http"
    "time"

    "gin-hola-mundo/models"
    "gin-hola-mundo/repositories"
    "gin-hola-mundo/utils"

    "gorm.io/gorm"
)

// ---- Tipos de request/response ----

type AssignPointsRequest struct {
    StudentID uint   `json:"student_id" binding:"required"`
    Amount    int    `json:"amount"     binding:"required,min=1"`
    Reason    string `json:"reason"     binding:"required"`
}

type PointResponse struct {
    ID           uint      `json:"id"`
    StudentID    uint      `json:"student_id"`
    AssignedByID uint      `json:"assigned_by_id"`
    Amount       int       `json:"amount"`
    Reason       string    `json:"reason"`
    CreatedAt    time.Time `json:"created_at"`
}

// ---- Interface del service ----

type PointService interface {
    AssignPoints(ctx context.Context, assignedByID uint, req AssignPointsRequest) (*PointResponse, error)
    GetStudentPoints(ctx context.Context, studentID uint) ([]PointResponse, error)
}

// ---- Implementación ----

type pointService struct {
    repo     repositories.PointRepository
    userRepo repositories.UserRepository  // para validar que el student existe
}

func NewPointService(repo repositories.PointRepository, userRepo repositories.UserRepository) PointService {
    return &pointService{repo: repo, userRepo: userRepo}
}

func (s *pointService) AssignPoints(ctx context.Context, assignedByID uint, req AssignPointsRequest) (*PointResponse, error) {
    // Verificar que el estudiante existe
    _, err := s.userRepo.FindByID(ctx, req.StudentID)
    if err != nil {
        if errors.Is(err, gorm.ErrRecordNotFound) {
            return nil, &utils.AppError{HTTPCode: http.StatusNotFound, Code: "STUDENT_NOT_FOUND", Message: "student not found"}
        }
        return nil, utils.ErrInternal
    }

    point := &models.Point{
        StudentID:    req.StudentID,
        AssignedByID: assignedByID,
        Amount:       req.Amount,
        Reason:       req.Reason,
    }

    if err := s.repo.Create(ctx, point); err != nil {
        return nil, utils.ErrInternal
    }

    return &PointResponse{
        ID:           point.ID,
        StudentID:    point.StudentID,
        AssignedByID: point.AssignedByID,
        Amount:       point.Amount,
        Reason:       point.Reason,
        CreatedAt:    point.CreatedAt,
    }, nil
}

func (s *pointService) GetStudentPoints(ctx context.Context, studentID uint) ([]PointResponse, error) {
    points, err := s.repo.FindByStudent(ctx, studentID)
    if err != nil {
        return nil, utils.ErrInternal
    }

    result := make([]PointResponse, len(points))
    for i, p := range points {
        result[i] = PointResponse{
            ID:           p.ID,
            StudentID:    p.StudentID,
            AssignedByID: p.AssignedByID,
            Amount:       p.Amount,
            Reason:       p.Reason,
            CreatedAt:    p.CreatedAt,
        }
    }
    return result, nil
}
```

---

### Paso 4 de 5 — Controller: `controllers/point_controller.go`

```go
package controllers

import (
    "errors"
    "net/http"
    "strconv"

    "gin-hola-mundo/services"
    "gin-hola-mundo/utils"

    "github.com/gin-gonic/gin"
)

type PointController struct {
    service services.PointService
}

func NewPointController(s services.PointService) *PointController {
    return &PointController{service: s}
}

// POST /api/v1/points  → solo admin o teacher
func (pc *PointController) Assign(c *gin.Context) {
    // Verificar autorización usando el rol del JWT
    callerRole := c.GetString("role")
    if callerRole != "admin" && callerRole != "teacher" {
        utils.Error(c, http.StatusForbidden, "only admins and teachers can assign points", "FORBIDDEN")
        return
    }

    var req services.AssignPointsRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        utils.Error(c, http.StatusUnprocessableEntity, err.Error(), "VALIDATION_ERROR")
        return
    }

    callerID := c.GetUint("userID")
    point, err := pc.service.AssignPoints(c.Request.Context(), callerID, req)
    if err != nil {
        var appErr *utils.AppError
        if errors.As(err, &appErr) {
            utils.Error(c, appErr.HTTPCode, appErr.Message, appErr.Code)
            return
        }
        utils.Error(c, http.StatusInternalServerError, "Internal server error", "INTERNAL_ERROR")
        return
    }

    c.JSON(http.StatusCreated, gin.H{"success": true, "data": point})
}

// GET /api/v1/points/student/:studentID  → cualquier usuario autenticado
func (pc *PointController) GetByStudent(c *gin.Context) {
    studentID, err := strconv.ParseUint(c.Param("studentID"), 10, 32)
    if err != nil {
        utils.Error(c, http.StatusBadRequest, "invalid student ID", "BAD_REQUEST")
        return
    }

    points, err := pc.service.GetStudentPoints(c.Request.Context(), uint(studentID))
    if err != nil {
        var appErr *utils.AppError
        if errors.As(err, &appErr) {
            utils.Error(c, appErr.HTTPCode, appErr.Message, appErr.Code)
            return
        }
        utils.Error(c, http.StatusInternalServerError, "Internal server error", "INTERNAL_ERROR")
        return
    }

    utils.Success(c, points)
}
```

---

### Paso 5 de 5 — Routes: cablear en `routes/routes.go`

Agrega estas líneas siguiendo el mismo patrón que los users:

```go
// En routes/routes.go, dentro de la función Setup:

// ── Points ──────────────────────────────────────────────
pointRepo := repositories.NewPointRepository(db)
pointSvc  := services.NewPointService(pointRepo, userRepo) // reutiliza userRepo ya creado
pointCtrl := controllers.NewPointController(pointSvc)

points := api.Group("/points", middlewares.Auth(cfg.JWTSecret))
points.POST("",                        pointCtrl.Assign)
points.GET("/student/:studentID",      pointCtrl.GetByStudent)
```

⚠️ `userRepo` ya fue declarado antes para los users, puedes reutilizarlo directamente.

---

### Resultado final — Nuevos endpoints disponibles

```
POST /api/v1/points                     → Asignar puntos (admin/teacher)
GET  /api/v1/points/student/:studentID  → Ver puntos de un estudiante
```

**Body para POST:**

```json
{
    "student_id": 5,
    "amount": 100,
    "reason": "Excelente participación en clase"
}
```

---

## 8. Cómo escalar el proyecto profesionalmente

### 8.1 Lo más urgente: endpoint de Login (Auth)

Actualmente no hay un endpoint de login — los tokens se deben generar manualmente. Este sería el primer módulo real a agregar:

```
POST /api/v1/auth/login   → recibe email + password, retorna JWT
```

Sigue los mismos 5 pasos del módulo de points, pero el service:

1. Busca el usuario por email
2. Compara el password con `bcrypt.CompareHashAndPassword`
3. Si es válido, llama a `utils.GenerateToken(user.ID, user.Role, secret, 24*time.Hour)`
4. Retorna el token

### 8.2 Paginación en listados

Cuando tengas endpoints de tipo `GET /api/v1/points/student/:id`, agrega paginación desde el inicio:

```go
// En el controller, leer query params
page, _  := strconv.Atoi(c.DefaultQuery("page", "1"))
limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
offset   := (page - 1) * limit

// En el repository, aplicar paginación
db.WithContext(ctx).Where("student_id = ?", studentID).
    Limit(limit).Offset(offset).Find(&points)
```

### 8.3 Tests unitarios — mockear el repository

La razón por la que el service recibe una **interface** y no la implementación concreta es justamente para poder hacer tests sin tocar la base de datos:

```go
// En services/user_service_test.go

type mockUserRepo struct{}

func (m *mockUserRepo) Create(ctx context.Context, user *models.User) error { return nil }
func (m *mockUserRepo) FindByEmail(ctx context.Context, email string) (*models.User, error) {
    return nil, gorm.ErrRecordNotFound  // simula que el email no existe
}
func (m *mockUserRepo) FindByID(ctx context.Context, id uint) (*models.User, error) { ... }

func TestCreateUser_Success(t *testing.T) {
    svc := NewUserService(&mockUserRepo{})
    resp, err := svc.CreateUser(context.Background(), CreateUserRequest{
        Name: "Ana", Email: "ana@test.com", Password: "secret123", Role: "student",
    })
    if err != nil {
        t.Fatalf("expected no error, got %v", err)
    }
    if resp.Email != "ana@test.com" {
        t.Errorf("expected ana@test.com, got %s", resp.Email)
    }
}
```

### 8.4 Logging estructurado

El middleware `logger.go` existe pero no está activado. Para producción, considera usar `slog` (estándar de Go desde 1.21) o `zap`:

```go
// En routes/routes.go, agregar junto a los otros middlewares:
r.Use(middlewares.Logger())

// Para producción, reemplazar por slog estructurado:
slog.Info("request", "method", c.Request.Method, "path", c.Request.URL.Path, "status", c.Writer.Status())
```

### 8.5 CORS restringido para producción

El `cors.go` actual permite `Access-Control-Allow-Origin: *` — correcto para desarrollo, peligroso en producción:

```go
// En middlewares/cors.go, leer el origen desde config:
func CORS(allowedOrigin string) gin.HandlerFunc {
    return func(c *gin.Context) {
        c.Header("Access-Control-Allow-Origin", allowedOrigin)
        // ...
    }
}

// En .env:
// ALLOWED_ORIGIN=https://mi-frontend.com
```

### 8.6 Separación de entornos

```
.env                 ← desarrollo local (git-ignored, nunca commitear)
.env.example         ← plantilla sin valores reales (commitear esto)
.env.production      ← configuración de producción (en el servidor, no en git)
```

### 8.7 Roadmap sugerido de módulos

```
Fase 1 (fundamentos)
  ✅ Users (crear usuario)
  ⬜ Auth (login → JWT)
  ⬜ Auth (refresh token)

Fase 2 (funcionalidad escolar)
  ⬜ Courses (materias)
  ⬜ Enrollments (inscripción de estudiantes a materias)
  ⬜ Points / Grades (100 Points ← estás aquí)

Fase 3 (escala)
  ⬜ Paginación en todos los listados
  ⬜ Tests unitarios en todos los services
  ⬜ Soft-delete correcto con GORM (DeletedAt ya está en gorm.Model)
  ⬜ Rate limiting middleware
  ⬜ CI/CD (GitHub Actions: go test + go build en cada PR)
```

---

## 9. Comandos rápidos de referencia

### Correr el proyecto

```bash
# Con Docker (recomendado, no necesitas PostgreSQL instalado)
cp .env.example .env          # edita DB_USER, DB_PASSWORD, JWT_SECRET
docker-compose up --build     # levanta PostgreSQL + API

# Local (necesitas PostgreSQL corriendo)
go run main.go
```

### Comandos Go útiles

```bash
go mod tidy                   # limpia dependencias, actualiza go.sum
go build -o server .          # compila un binario llamado "server"
go vet ./...                  # análisis estático básico (como un linter ligero)
go test ./...                 # corre todos los tests del proyecto
```

### Generar un JWT manualmente (para probar endpoints protegidos)

Agrega esto temporalmente en `main.go` después de cargar `cfg`, cópialo de la consola y quítalo:

```go
token, _ := utils.GenerateToken(1, "admin", cfg.JWTSecret, 24*time.Hour)
log.Println("DEBUG TOKEN:", token)
```

### Probar endpoints con curl

```bash
# Health check (sin token)
curl http://localhost:8080/api/v1/health

# Crear usuario (con token)
curl -X POST http://localhost:8080/api/v1/users \
  -H "Authorization: Bearer <tu_token>" \
  -H "Content-Type: application/json" \
  -d '{"name":"Ana","email":"ana@school.com","password":"secret123","role":"student"}'

# Asignar puntos (después de agregar el módulo Points)
curl -X POST http://localhost:8080/api/v1/points \
  -H "Authorization: Bearer <token_admin_o_teacher>" \
  -H "Content-Type: application/json" \
  -d '{"student_id":1,"amount":100,"reason":"Participación en clase"}'
```

---

> **Regla de oro para mantener el proyecto limpio:**
> Cada vez que necesites agregar funcionalidad nueva, pregúntate:
> ¿Es lógica de negocio? → va al **service**.
> ¿Es una operación de base de datos? → va al **repository**.
> ¿Es parsing HTTP o respuesta? → va al **controller**.
> ¿Es una tabla nueva? → crea el **model** y agrégalo a **AutoMigrate**.
