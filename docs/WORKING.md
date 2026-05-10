# Guía de Trabajo: Cómo Desarrollar en Este Proyecto

> **Para quién es esto:** Para el día a día — cómo levantar, cómo agregar endpoints, cómo no romperse la cabeza. Si buscas entender la arquitectura desde cero, lee primero [GUIDE.md](./GUIDE.md).

---

## Tabla de Contenidos

1. [Levantar el proyecto](#1-levantar-el-proyecto)
2. [Obtener un token JWT para probar](#2-obtener-un-token-jwt-para-probar)
3. [Flujo de trabajo diario](#3-flujo-de-trabajo-diario)
4. [Receta A — Nuevo endpoint en módulo existente](#4-receta-a--nuevo-endpoint-en-módulo-existente)
5. [Receta B — Nuevo módulo desde cero](#5-receta-b--nuevo-módulo-desde-cero)
6. [Receta C — Endpoint público vs protegido](#6-receta-c--endpoint-público-vs-protegido)
7. [Receta D — Path params, query params y body](#7-receta-d--path-params-query-params-y-body)
8. [Receta E — Agregar un campo a un modelo existente](#8-receta-e--agregar-un-campo-a-un-modelo-existente)
9. [Errores comunes y cómo resolverlos](#9-errores-comunes-y-cómo-resolverlos)
10. [Referencia rápida](#10-referencia-rápida)

---

## 1. Levantar el proyecto

### Con Docker (recomendado)

```bash
# La primera vez
cp .env.example .env
# Edita .env y pon valores reales en: DB_USER, DB_PASSWORD, JWT_SECRET

# Levantar todo (PostgreSQL + API)
docker-compose up --build

# En arranques posteriores (sin reconstruir imagen)
docker-compose up
```

**Verificar que está corriendo:**

```bash
curl http://localhost:8080/api/v1/health
# Esperado: {"success":true,"data":{"status":"ok"}}
```

### Sin Docker (local)

Requiere PostgreSQL instalado y corriendo en `localhost:5432`.

```bash
# Crear la base de datos manualmente antes
createdb school_system

cp .env.example .env   # edita con tus credenciales locales
go run main.go
```

### Variables de entorno mínimas (`.env`)

```env
APP_PORT=8080
DB_HOST=localhost
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=tu_password
DB_NAME=school_system
JWT_SECRET=cualquier_string_largo_y_secreto
```

> `JWT_SECRET` es la única **obligatoria** — el servidor no arranca sin ella.

---

## 2. Obtener un token JWT para probar

No existe un endpoint de login todavía. Para probar endpoints protegidos, genera el token manualmente:

**Paso 1** — Agrega esto temporalmente al inicio de `main.go`, justo después de `cfg := config.Load()`:

```go
token, _ := utils.GenerateToken(1, "admin", cfg.JWTSecret, 24*time.Hour)
log.Println("DEBUG TOKEN:", token)
```

**Paso 2** — Corre el servidor. Copia el token de la consola.

**Paso 3** — Quita esas dos líneas antes de hacer commit.

**Usar el token en curl:**

```bash
export TOKEN="eyJ..."   # pega tu token aquí

curl -X POST http://localhost:8080/api/v1/users \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"name":"Ana","email":"ana@school.com","password":"secret123","role":"student"}'
```

---

## 3. Flujo de trabajo diario

```
1. docker-compose up          ← levanta el entorno
2. Edita el código
3. Ctrl+C → docker-compose up --build   ← reinicia con cambios
4. Prueba con curl o Postman
5. go vet ./...               ← verifica errores antes de commitear
```

> **Tip:** Mientras desarrollas en local (sin Docker), `go run main.go` recarga más rápido que reconstruir la imagen. Usa Docker principalmente para CI o para probar la imagen final.

---

## 4. Receta A — Nuevo endpoint en módulo existente

**Caso:** el módulo `users` ya existe y quieres agregar `GET /api/v1/users/:id`.

Tocarás **4 archivos en este orden** (de abajo hacia arriba en la arquitectura):

---

### 4.1 Repository — agregar el método de datos

El método `FindByID` ya existe en `repositories/user_repository.go`. Si no existiera, lo agregarías así:

**1. Declararlo en la interface:**

```go
type UserRepository interface {
    Create(ctx context.Context, user *models.User) error
    FindByEmail(ctx context.Context, email string) (*models.User, error)
    FindByID(ctx context.Context, id uint) (*models.User, error)  // ← nuevo
}
```

**2. Implementarlo en la struct:**

```go
func (r *userRepository) FindByID(ctx context.Context, id uint) (*models.User, error) {
    var user models.User
    err := r.db.WithContext(ctx).First(&user, id).Error
    if err != nil {
        return nil, err
    }
    return &user, nil
}
```

> **Regla:** si el método ya está en la interface pero no en la struct, el compilador te avisará con un error claro.

---

### 4.2 Service — agregar la función de negocio

En `services/user_service.go`:

**1. Agregar a la interface:**

```go
type UserService interface {
    CreateUser(ctx context.Context, req CreateUserRequest) (*UserResponse, error)
    GetUserByID(ctx context.Context, id uint) (*UserResponse, error)  // ← nuevo
}
```

**2. Implementar:**

```go
func (s *userService) GetUserByID(ctx context.Context, id uint) (*UserResponse, error) {
    user, err := s.repo.FindByID(ctx, id)
    if err != nil {
        if errors.Is(err, gorm.ErrRecordNotFound) {
            return nil, utils.ErrNotFound
        }
        return nil, utils.ErrInternal
    }

    return &UserResponse{
        ID:        user.ID,
        Name:      user.Name,
        Email:     user.Email,
        Role:      user.Role,
        CreatedAt: user.CreatedAt,
    }, nil
}
```

---

### 4.3 Controller — agregar el handler HTTP

En `controllers/user_controller.go`:

```go
func (uc *UserController) GetByID(c *gin.Context) {
    id, err := strconv.ParseUint(c.Param("id"), 10, 32)
    if err != nil {
        utils.Error(c, http.StatusBadRequest, "invalid user ID", "BAD_REQUEST")
        return
    }

    user, err := uc.service.GetUserByID(c.Request.Context(), uint(id))
    if err != nil {
        var appErr *utils.AppError
        if errors.As(err, &appErr) {
            utils.Error(c, appErr.HTTPCode, appErr.Message, appErr.Code)
            return
        }
        utils.Error(c, http.StatusInternalServerError, "Internal server error", "INTERNAL_ERROR")
        return
    }

    utils.Success(c, user)
}
```

Asegúrate de tener el import de `strconv` en la lista de imports del archivo.

---

### 4.4 Routes — registrar la ruta

En `routes/routes.go`, dentro del grupo `users`:

```go
users := api.Group("/users", middlewares.Auth(cfg.JWTSecret))
users.POST("", userCtrl.Create)
users.GET("/:id", userCtrl.GetByID)   // ← nueva línea
```

---

### Checklist Receta A

- [ ] Método nuevo en la **interface** del repository (si se necesita una query nueva)
- [ ] Método nuevo en la **implementación** del repository
- [ ] Función nueva en la **interface** del service
- [ ] Función nueva en la **implementación** del service
- [ ] Handler nuevo en el **controller**
- [ ] Ruta registrada en **routes.go**
- [ ] `go vet ./...` sin errores

---

## 5. Receta B — Nuevo módulo desde cero

Usa este checklist cada vez que crees un módulo nuevo. El orden importa.

### Archivos a crear (en este orden)

```
models/foo.go                  ← 1. primero
repositories/foo_repository.go ← 2.
services/foo_service.go        ← 3.
controllers/foo_controller.go  ← 4.
```

Y modificar:

```
database/database.go           ← registrar el modelo en AutoMigrate
routes/routes.go               ← cablear y registrar rutas
```

---

### Template 1 — `models/foo.go`

```go
package models

import "gorm.io/gorm"

type Foo struct {
    gorm.Model
    Name string `gorm:"not null"`
    // ... tus campos
}
```

Luego en `database/database.go`:

```go
// Agrega &models.Foo{} a la lista
db.AutoMigrate(&models.User{}, &models.Foo{})
```

---

### Template 2 — `repositories/foo_repository.go`

```go
package repositories

import (
    "context"

    "gin-hola-mundo/models"

    "gorm.io/gorm"
)

type FooRepository interface {
    Create(ctx context.Context, foo *models.Foo) error
    FindByID(ctx context.Context, id uint) (*models.Foo, error)
}

type fooRepository struct {
    db *gorm.DB
}

func NewFooRepository(db *gorm.DB) FooRepository {
    return &fooRepository{db: db}
}

func (r *fooRepository) Create(ctx context.Context, foo *models.Foo) error {
    return r.db.WithContext(ctx).Create(foo).Error
}

func (r *fooRepository) FindByID(ctx context.Context, id uint) (*models.Foo, error) {
    var foo models.Foo
    err := r.db.WithContext(ctx).First(&foo, id).Error
    if err != nil {
        return nil, err
    }
    return &foo, nil
}
```

---

### Template 3 — `services/foo_service.go`

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

// ── Tipos de request/response (viven AQUÍ, no en el controller) ──

type CreateFooRequest struct {
    Name string `json:"name" binding:"required"`
}

type FooResponse struct {
    ID        uint      `json:"id"`
    Name      string    `json:"name"`
    CreatedAt time.Time `json:"created_at"`
}

// ── Interface ──

type FooService interface {
    CreateFoo(ctx context.Context, req CreateFooRequest) (*FooResponse, error)
}

// ── Implementación ──

type fooService struct {
    repo repositories.FooRepository
}

func NewFooService(repo repositories.FooRepository) FooService {
    return &fooService{repo: repo}
}

func (s *fooService) CreateFoo(ctx context.Context, req CreateFooRequest) (*FooResponse, error) {
    foo := &models.Foo{Name: req.Name}

    if err := s.repo.Create(ctx, foo); err != nil {
        return nil, utils.ErrInternal
    }

    return &FooResponse{
        ID:        foo.ID,
        Name:      foo.Name,
        CreatedAt: foo.CreatedAt,
    }, nil
}
```

---

### Template 4 — `controllers/foo_controller.go`

```go
package controllers

import (
    "errors"
    "net/http"

    "gin-hola-mundo/services"
    "gin-hola-mundo/utils"

    "github.com/gin-gonic/gin"
)

type FooController struct {
    service services.FooService
}

func NewFooController(s services.FooService) *FooController {
    return &FooController{service: s}
}

func (fc *FooController) Create(c *gin.Context) {
    var req services.CreateFooRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        utils.Error(c, http.StatusUnprocessableEntity, err.Error(), "VALIDATION_ERROR")
        return
    }

    foo, err := fc.service.CreateFoo(c.Request.Context(), req)
    if err != nil {
        var appErr *utils.AppError
        if errors.As(err, &appErr) {
            utils.Error(c, appErr.HTTPCode, appErr.Message, appErr.Code)
            return
        }
        utils.Error(c, http.StatusInternalServerError, "Internal server error", "INTERNAL_ERROR")
        return
    }

    c.JSON(http.StatusCreated, gin.H{"success": true, "data": foo})
}
```

---

### Template 5 — Cablear en `routes/routes.go`

```go
// Dentro de la función Setup, siguiendo el patrón existente:
fooRepo := repositories.NewFooRepository(db)
fooSvc  := services.NewFooService(fooRepo)
fooCtrl := controllers.NewFooController(fooSvc)

foos := api.Group("/foos", middlewares.Auth(cfg.JWTSecret))
foos.POST("", fooCtrl.Create)
```

---

### Checklist Receta B

- [ ] `models/foo.go` creado
- [ ] `models.Foo{}` agregado a `AutoMigrate` en `database/database.go`
- [ ] `repositories/foo_repository.go` creado (interface + impl)
- [ ] `services/foo_service.go` creado (request/response types + interface + impl)
- [ ] `controllers/foo_controller.go` creado
- [ ] Cableado en `routes/routes.go`
- [ ] `go vet ./...` sin errores
- [ ] Servidor reiniciado (GORM crea la tabla automáticamente)

---

## 6. Receta C — Endpoint público vs protegido

### Endpoint público (sin token)

```go
// En routes/routes.go — fuera de cualquier grupo con Auth
api := r.Group("/api/v1")
api.GET("/health", controllers.Health)   // ← sin middleware
api.POST("/auth/login", authCtrl.Login)  // ← otro ejemplo público
```

### Endpoint protegido (requiere JWT válido)

```go
// Agrupado con middlewares.Auth como segundo argumento del Group
protected := api.Group("/users", middlewares.Auth(cfg.JWTSecret))
protected.POST("", userCtrl.Create)
protected.GET("/:id", userCtrl.GetByID)
```

### Endpoint protegido con verificación de rol dentro del handler

El middleware `Auth` valida el token, pero **no restringe por rol**. La lógica de roles va en el controller:

```go
func (fc *FooController) AdminOnlyAction(c *gin.Context) {
    role := c.GetString("role")
    if role != "admin" {
        utils.Error(c, http.StatusForbidden, "admins only", "FORBIDDEN")
        return
    }
    // ...
}
```

### Leer datos del usuario autenticado en cualquier handler

```go
userID   := c.GetUint("userID")    // uint  — ID del usuario del token
userRole := c.GetString("role")    // string — "admin", "teacher", "student", "parent"
```

---

## 7. Receta D — Path params, query params y body

### Path params — `/users/:id`

```go
// Ruta registrada como: users.GET("/:id", ...)

idStr := c.Param("id")                           // string "42"
id, err := strconv.ParseUint(idStr, 10, 32)      // convierte a uint64
if err != nil {
    utils.Error(c, http.StatusBadRequest, "invalid ID", "BAD_REQUEST")
    return
}
// usar uint(id)
```

### Query params — `/points/student/:id?page=2&limit=10`

```go
page,  _ := strconv.Atoi(c.DefaultQuery("page",  "1"))   // default: 1
limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))  // default: 20

// Si el query param es obligatorio y no tiene default:
filter := c.Query("role")   // "" si no viene
```

### Request body — JSON

Define el tipo en el service, usa `ShouldBindJSON` en el controller:

```go
// En services/foo_service.go:
type CreateFooRequest struct {
    Name  string `json:"name"  binding:"required"`
    Email string `json:"email" binding:"required,email"`
    Age   int    `json:"age"   binding:"omitempty,min=0,max=150"`
}

// En controllers/foo_controller.go:
var req services.CreateFooRequest
if err := c.ShouldBindJSON(&req); err != nil {
    utils.Error(c, http.StatusUnprocessableEntity, err.Error(), "VALIDATION_ERROR")
    return
}
```

**Tags de binding más usados:**

| Tag                   | Qué hace                                   |
| --------------------- | ------------------------------------------ |
| `binding:"required"`  | El campo es obligatorio                    |
| `binding:"email"`     | Debe ser un email válido                   |
| `binding:"min=8"`     | Mínimo 8 caracteres (string) o valor (int) |
| `binding:"max=100"`   | Máximo                                     |
| `binding:"omitempty"` | Solo valida si el campo viene en el JSON   |
| `binding:"oneof=a b"` | Debe ser uno de los valores listados       |

---

## 8. Receta E — Agregar un campo a un modelo existente

**Caso:** quieres agregar un campo `PhoneNumber` al modelo `User`.

### Paso 1 — Editar el struct en `models/user.go`

```go
type User struct {
    gorm.Model
    Name        string `gorm:"not null"`
    Email       string `gorm:"uniqueIndex;not null"`
    Password    string `gorm:"not null"`
    Role        string `gorm:"index;not null;default:'student'"`
    PhoneNumber string `gorm:"default:null"`   // ← nuevo campo, nullable
}
```

### Paso 2 — Reiniciar el servidor

GORM detecta el campo nuevo y ejecuta `ALTER TABLE users ADD COLUMN phone_number TEXT` automáticamente gracias a `AutoMigrate`. No necesitas escribir SQL.

### Caveats importantes

| Situación                             | Qué pasa                                                                                                |
| ------------------------------------- | ------------------------------------------------------------------------------------------------------- |
| Campo nuevo nullable (`default:null`) | AutoMigrate lo agrega sin problema                                                                      |
| Campo nuevo `NOT NULL` sin default    | AutoMigrate falla si ya hay filas en la tabla — agrega un default o migra manualmente                   |
| Renombrar un campo                    | AutoMigrate **no** renombra — crea una columna nueva y deja la vieja. Requiere migración manual con SQL |
| Eliminar un campo                     | AutoMigrate **no** elimina columnas — solo agrega. Para borrar una columna, usa SQL directo             |

### Paso 3 — Actualizar el `UserResponse` en el service (si quieres exponerlo)

```go
type UserResponse struct {
    ID          uint      `json:"id"`
    Name        string    `json:"name"`
    Email       string    `json:"email"`
    Role        string    `json:"role"`
    PhoneNumber string    `json:"phone_number,omitempty"`  // ← nuevo
    CreatedAt   time.Time `json:"created_at"`
}
```

> `omitempty` hace que el campo no aparezca en el JSON si está vacío — útil para campos opcionales.

---

## 9. Errores comunes y cómo resolverlos

### Al arrancar el servidor

| Error                           | Causa                                                            | Solución                                                           |
| ------------------------------- | ---------------------------------------------------------------- | ------------------------------------------------------------------ |
| `JWT_SECRET is required`        | La variable no está en `.env` o el archivo no existe             | Crea `.env` desde `.env.example` y pon un valor en `JWT_SECRET`    |
| `Failed to connect to database` | PostgreSQL no está corriendo o las credenciales son incorrectas  | Verifica `DB_HOST`, `DB_USER`, `DB_PASSWORD` en `.env`             |
| `AutoMigrate failed`            | Nuevo campo `NOT NULL` sin default en tabla con datos existentes | Haz el campo nullable o agrega `default:'valor'` en el tag de GORM |

### Durante el desarrollo

| Error                                                  | Causa                                                                        | Solución                                                                                     |
| ------------------------------------------------------ | ---------------------------------------------------------------------------- | -------------------------------------------------------------------------------------------- |
| `cannot use X as type Y`                               | Pasaste la implementación concreta donde se espera la interface, o viceversa | Revisa que estás pasando el tipo correcto — normalmente el constructor devuelve la interface |
| `*fooRepository does not implement FooRepository`      | Falta implementar algún método de la interface                               | El compilador te dice qué método falta — agrégalo a la struct                                |
| `record not found`                                     | `gorm.ErrRecordNotFound` llegó como `nil` — lo usaste sin verificar          | Siempre verifica `errors.Is(err, gorm.ErrRecordNotFound)` antes de devolver `ErrInternal`    |
| Binding retorna error vacío                            | El tag `binding:"required"` falló pero el mensaje no es claro                | Imprime `err.Error()` — Gin/validator muestra qué campo falló                                |
| La tabla nueva no se creó                              | Olvidaste agregar el modelo a `AutoMigrate`                                  | En `database/database.go` agrega `&models.TuModelo{}` a la lista                             |
| `Authorization header required` en un endpoint público | La ruta está registrada dentro de un grupo con `middlewares.Auth`            | Mueve la ruta fuera del grupo protegido en `routes.go`                                       |

### En las respuestas HTTP

| Síntoma                     | Causa probable                                                                                         |
| --------------------------- | ------------------------------------------------------------------------------------------------------ |
| Siempre retorna `500`       | El service devuelve `utils.ErrInternal` — agrega un `log.Println(err)` temporal para ver el error real |
| `401` en todas las requests | El token expiró, está mal formado, o el `JWT_SECRET` no coincide con el que se usó para generarlo      |
| `422` en el body            | Algún campo de `ShouldBindJSON` falló la validación — el mensaje de error indica cuál                  |

---

## 10. Referencia rápida

### Helpers de respuesta (`utils/`)

```go
utils.Success(c, data)                            // 200 {"success":true,"data":...}
utils.Error(c, http.StatusNotFound, "msg", "CODE") // {"success":false,"error":"msg","code":"CODE"}
```

### Sentinels de error predefinidos

```go
utils.ErrNotFound     // 404 NOT_FOUND
utils.ErrUnauthorized // 401 UNAUTHORIZED
utils.ErrForbidden    // 403 FORBIDDEN
utils.ErrBadRequest   // 400 BAD_REQUEST
utils.ErrInternal     // 500 INTERNAL_ERROR
```

### Leer contexto Gin en un handler

```go
c.GetUint("userID")        // ID del usuario autenticado (del JWT)
c.GetString("role")        // rol del usuario autenticado
c.Param("id")              // path param :id como string
c.Query("page")            // query param ?page=
c.DefaultQuery("limit","20") // query param con valor por defecto
```

### Patrón de manejo de error en controllers

```go
foo, err := fc.service.DoSomething(c.Request.Context(), req)
if err != nil {
    var appErr *utils.AppError
    if errors.As(err, &appErr) {
        utils.Error(c, appErr.HTTPCode, appErr.Message, appErr.Code)
        return
    }
    utils.Error(c, http.StatusInternalServerError, "Internal server error", "INTERNAL_ERROR")
    return
}
utils.Success(c, foo)
```

### Convenciones del proyecto

| Qué                            | Convención                                                  |
| ------------------------------ | ----------------------------------------------------------- |
| Nombres de archivos            | `snake_case.go` (ej. `user_repository.go`)                  |
| Nombres de structs y tipos     | `PascalCase` (ej. `UserRepository`, `CreateUserRequest`)    |
| Nombres de funciones/métodos   | `PascalCase` si son exportados, `camelCase` si son privados |
| Códigos de error en respuestas | `SCREAMING_SNAKE_CASE` (ej. `"EMAIL_TAKEN"`, `"NOT_FOUND"`) |
| Tipos `Request` y `Response`   | Viven en `services/`, no en controllers ni models           |
| Lógica de negocio              | Solo en `services/`, nunca en controllers ni repositories   |
| Queries SQL                    | Solo en `repositories/`, nunca en services ni controllers   |

### Endpoints actuales

| Método | Path             | Auth               | Descripción                  |
| ------ | ---------------- | ------------------ | ---------------------------- |
| `GET`  | `/api/v1/health` | No                 | Verifica que la API responde |
| `POST` | `/api/v1/users`  | Sí (cualquier rol) | Crea un nuevo usuario        |
