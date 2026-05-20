# Propuestas de refactoring para testabilidad de `BaseRequest`

Ordenadas de **menor a mayor impacto**. Cada una resuelve el problema anterior y agrega su propia mejora.

---

## Estado actual antes del refactoring (`UserExistsByMail` / `UserExistsByEmail`)

En el código actual, el método existente se llama `UserExistsByEmail` en `services/system_service.go` (equivalente funcional al `UserExistsByMail` mencionado).

```go
// UserExistsByEmail checks if a user with the given email exists in the database
func UserExistsByEmail(email string) (bool, error) {
    systemUser, err := getSystemUser()
    if err != nil {
        return false, fmt.Errorf("error getting system user: %v", err)
    }

    user := &foundation.User{
        Username: email,
    }

    // Create a base request
    baseRequest, err := foundation.NewBaseRequestWithModel(user, *systemUser)
    if err != nil {
        return false, fmt.Errorf("error creating base request: %v", err)
    }

    // Set up the find options to search by email
    findOptions := user.GetFindOptions(baseRequest)
    findOptions.AddEquals("username", email)
    baseRequest.SetFindOptions(findOptions)

    // Execute the find query
    response := user.Find(baseRequest)
    if response.Error != nil {
        return false, response.Error
    }

    // If we found at least one user with this email, return true
    return response.TotalRows > 0, nil
}
```

---

## Nivel 1: Usar variable overridable en `UserExistsByEmail`

**Impacto:** Mínimo. Sin cambios de API, sin tocar callers.

**Problema:** `UserExistsByEmail` (system_service.go:657) llama directo a `foundation.NewBaseRequestWithModel`, ignorando la variable `newBaseRequestWithModel` que ya existe para `ResetPassword`.

**Solución:**

```go
// system_service.go:657 — cambiar de
baseRequest, err := foundation.NewBaseRequestWithModel(user, *systemUser)
// a
baseRequest, err := newBaseRequestWithModel(user, *systemUser)
```

**En el test:**

```go
original := newBaseRequestWithModel
defer func() { newBaseRequestWithModel = original }()
newBaseRequestWithModel = func(model *foundation.User, user foundation.User) (*foundation.BaseRequest, error) {
    return &foundation.BaseRequest{
        Repo: mockRepo,
        User: user,
        Model: model,
    }, nil
}
```

**No cubre:** `UserExistsByEmail` usa `user.Find(baseRequest)` internamente — el mock de `BaseRequest` necesita un `Repo` que responda a `Find()`.

---

## Nivel 2: Skip repo creation si `request.Repo` ya está seteado

**Impacto:** Bajo. No rompe callers existentes, solo agrega un guard.

**Problema:** `NewFoundationBaseRequestWithRepository` (services.go:126) siempre llama a `NewRepositoryFromModel`, incluso si `request.Repo` ya es un mock.

**Solución:**

```go
// services.go:128
if request.RepoModel != nil && request.Repo == nil {
    // solo crear repo si no hay uno ya inyectado
    if request.RepoModel.GetRepoID() == "" {
        request.RepoModel.SetRepoID(request.RepoID)
    }
    repo, err := foundation.NewRepositoryFromModel(request.RepoModel, request.Connection)
    if err != nil {
        return &foundation.BaseRequest{}, err
    }
    request.Repo = repo
}
```

Esto permite que **controllers** (controllers.go:66) siga funcionando igual, pero un test que pase `request.Repo = mockRepo` no dispara conexión a MongoDB.

**En el test:**

```go
request := services.NewServiceRequest()
request.Repo = mockRepo       // no toca MongoDB
request.RepoModel = mockModel // necesario para pasar validaciones
baseRequest, err := services.NewFoundationBaseRequestWithRepository(request)
```

---

## Nivel 3: Aceptar `*foundation.BaseRequest` como parámetro

**Impacto:** Medio. Cambia firma de funciones, hay que actualizar callers.

**Problema:** Funciones como `UserExistsByEmail` y `ResetPassword` crean su propio `BaseRequest` internamente, acoplando lógica de negocio con creación de infraestructura.

**Solución (ejemplo para `UserExistsByEmail`):**

```go
// system_service.go
func UserExistsByEmail(request *foundation.BaseRequest, email string) (bool, error) {
    findOptions := request.Model.GetFindOptions(request)
    findOptions.AddEquals("username", email)
    request.SetFindOptions(findOptions)

    response := request.Model.Find(request)
    if response.Error != nil {
        return false, response.Error
    }
    return response.TotalRows > 0, nil
}
```

El caller existente (line 708) cambia de:

```go
exists, err := UserExistsByEmail(email)
```

a:

```go
baseRequest, err := foundation.NewBaseRequestWithModel(user, *systemUser)
// ... error handling ...
exists, err := UserExistsByEmail(baseRequest, email)
```

**En el test:**

```go
baseRequest := &foundation.BaseRequest{
    Repo:  mockRepo,
    Model: mockUserModel,
    User:  testUser,
}
exists, err := UserExistsByEmail(baseRequest, "test@example.com")
```

**Tradeoff:** Saca la lógica de creación de `BaseRequest` del servicio, pero traslada la responsabilidad al caller. Para un servicio llamado desde controllers, el controller ya tiene acceso a `ServiceRequest` y puede crear el `BaseRequest` con `NewFoundationBaseRequest`.

---

## Nivel 4: Extraer `RepoFactory` detrás de una interfaz

**Impacto:** Medio-alto. Nueva abstracción, refactor de `NewRepositoryFromModel` y sus consumidores.

**Problema:** `NewRepositoryFromModel` es una función concreta que siempre crea un `MongoRepository`. No se puede mockear sin cambiar TODOS los paths que la usan.

**Solución:**

```go
// foundation/repository.go
type RepoFactory interface {
    FromModel(model RepositoryModel, connection string) (Repository, error)
    Clone(repo Repository, model RepositoryModel) (Repository, error)
}

// Implementación concreta (producción)
type MongoRepoFactory struct{}
func (f *MongoRepoFactory) FromModel(model RepositoryModel, connection string) (Repository, error) {
    return NewRepositoryFromModel(model, connection)
}
func (f *MongoRepoFactory) Clone(repo Repository, model RepositoryModel) (Repository, error) {
    return CloneRepository(repo, model)
}
```

Y cambiar `BaseRequest` para aceptar opcionalmente un `RepoFactory`:

```go
type BaseRequest struct {
    // ... campos existentes ...
    RepoFactory RepoFactory // opcional, default MongoRepoFactory
}
```

O bien, inyectarlo en `ServiceRequest`:

```go
type ServiceRequest struct {
    // ... campos existentes ...
    RepoFactory foundation.RepoFactory
}
```

**En el test:**

```go
request.RepoFactory = &MockRepoFactory{
    Repo: mockRepo,
}
```

**Tradeoff:** Más arquitectura. Vale la pena si hay planes de agregar más backends (no solo MongoDB) o si la creación de repos es un punto de dolor recurrente en tests.

---

## Nivel 5: Constructor injection en services

**Impacto:** Alto. Refactor mayor, toda la capa de servicios pasa a struct con dependencias.

**Problema:** Todas las funciones de `system_service.go` son funciones sueltas (package-level functions) que crean sus dependencias internamente. No hay un punto único de inyección.

**Solución:**

```go
// services/system_service.go
type SystemService struct {
    repoFactory foundation.RepoFactory
    httpClient  *http.Client
}

func NewSystemService(repoFactory foundation.RepoFactory) *SystemService {
    return &SystemService{
        repoFactory: repoFactory,
        httpClient:  &http.Client{Timeout: 30 * time.Second},
    }
}

func (s *SystemService) UserExistsByEmail(email string) (bool, error) {
    // usa s.repoFactory en vez de foundation.NewBaseRequestWithModel
    systemUser, err := getSystemUser()
    // ...
    model := &foundation.User{Username: email}
    repo, err := s.repoFactory.FromModel(model, systemUser.Connection)
    baseRequest, err := foundation.NewBaseRequest(model, repo, *systemUser)
    // ...
}
```

**En el test:**

```go
svc := NewSystemService(&MockRepoFactory{Repo: mockRepo})
exists, err := svc.UserExistsByEmail("test@example.com")
```

**Tradeoff:** Cambio grande. Implica refactor de controllers y de todos los callers. Pero es la solución más limpia y alinea el código con principios SOLID (DIP).

---

## Tabla comparativa

| Nivel | Solución | Impacto | Dependencias mockeables | Esfuerzo |
|---|---|---|---|---|
| 1 | Usar variable overridable en `UserExistsByEmail` | Mínimo | `NewBaseRequestWithModel` | 1 línea prod + test |
| 2 | Skip repo si `request.Repo` ya seteado | Bajo | `NewFoundationBaseRequestWithRepository` | 1 guard + test |
| 3 | Aceptar `*BaseRequest` como parámetro | Medio | `UserExistsByEmail`, `ResetPassword` | Cambiar firmas + callers |
| 4 | Interfaz `RepoFactory` | Medio-alto | Creación de repos en general | Nueva abstracción + refactor |
| 5 | Constructor injection en services | Alto | Todas las dependencias | Refactor mayor |

---

## Recomendación

Hacer **Nivel 1 + 2** YA como quick wins. Después evaluar si el dolor está en:
- **Controllers que llaman a `NewFoundationBaseRequestWithRepository`** → Nivel 2 alcanza
- **Tests de `UserExistsByEmail`/`ResetPassword`** → Nivel 3
- **Todo el codebase tiene este problema** → Nivel 4 o 5
