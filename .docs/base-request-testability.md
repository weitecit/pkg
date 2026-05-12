# Análisis de testabilidad de `BaseRequest`

## Dos caminos, dos realidades distintas

### ✅ Camino fácil: `NewBaseRequest(model, repo, user)` — **totalmente mockeable**

Las 3 dependencias son triviales de inyectar:

| Dependencia | Tipo | ¿Fácil de mockear? |
|---|---|---|
| `Repo` | `Repository` (interface) | ✅ Sí — 13 métodos, pero podés usar un mock parcial/stub |
| `Model` | `RepositoryModel` (interface) | ✅ Sí — 11 métodos, mayormente getters/setters |
| `User` | `struct` (concreto) | ✅ Sí — lo construís con `foundation.User{ID: &oid, Username: "test"}` |

Este es el path que usa **`services.NewFoundationBaseRequest(request)`** cuando ya le pasaste `request.Repo` como mock. Cero magia, cero infraestructura.

---

### ❌ Camino difícil: `NewBaseRequestWithModel(model, user)` — **NO es mockeable**

Internamente hace:

```go
repo, err := NewRepositoryFromModel(model, user.Connection)
// → NewRepository(connection, model.GetRepoType(), ...)
// → NewMongoRepository(connection, database, collection, isGlobal)
```

**Hardcodea MongoDB.** No hay interface, no hay inyección, no hay punto de fuga. Si tu servicio llama a `NewBaseRequestWithModel`, necesitás una MongoDB real o refactorizar.

---

## ¿Qué pasa en `services/`?

| Función | Path | ¿Mockeable? |
|---|---|---|
| `NewFoundationBaseRequest(request)` | `NewBaseRequest(model, request.Repo, user)` — repo viene del `ServiceRequest` | ✅ Sí, seteás `request.Repo = mockRepo` |
| `NewFoundationBaseRequestWithRepository(request)` | Ignora `request.Repo`, llama internamente `NewRepositoryFromModel` | ❌ No, crea Mongo real |
| `UserExistsByEmail` | Llama directo a `NewBaseRequestWithModel` | ❌ No |
| `ResetPassword` | Usa `newBaseRequestWithModel` (variable override) | ✅ Sí, reasignás la variable en test |

---

## La variable overridable — patrón actual en `system_service.go`

```go
var newBaseRequestWithModel = func(model *foundation.User, user foundation.User) (*foundation.BaseRequest, error) {
    return foundation.NewBaseRequestWithModel(model, user)
}
```

Se puede sobreescribir en tests (como ya hacen con `cloneHttpClient`), pero:
- Solo cubre `ResetPassword`
- **`UserExistsByEmail` no lo usa** — llama directo a `foundation.NewBaseRequestWithModel`, bypassing la variable
- Es un fix parcial, no resuelve el problema de raíz

---

## Conclusión

| Escenario | Qué se necesita |
|---|---|
| Servicio usa `NewFoundationBaseRequest` con `Repo` ya seteado | Nada — pasás `request.Repo = mockRepo` y funcionás |
| Servicio usa `NewBaseRequest` directo | Nada — pasás el mock de `Repository` directo |
| Servicio llama a `NewFoundationBaseRequestWithRepository` | Refactor: necesita aceptar un `Repository` externo |
| Servicio llama a `NewBaseRequestWithModel` | Refactor: necesita separar la creación del repo de la del request |
| Servicio usa `UserExistsByEmail` | Refactor mínimo: cambiar `UserExistsByEmail` para usar la variable overridable o aceptar un `Repository` |

**El problema no es `BaseRequest` en sí — es `NewRepositoryFromModel` que te fuerza a MongoDB en ciertos caminos.** Cualquier servicio que pase por `NewFoundationBaseRequest` con `request.Repo` ya seteado se testea sin problemas.
