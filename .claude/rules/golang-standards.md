# Estándares de Go

Reglas de código aplicables a todo el código Go de este repositorio.

## Formato y herramientas

- El código debe pasar `gofmt`/`goimports` y `golangci-lint run ./...` (`.golangci.yml`) sin issues antes de darse por terminado.
- No se edita a mano nada bajo `internal/repository/postgres/generated` (código generado por sqlc); se regenera con `make generate`.
- Nunca usar `//nolint` para silenciar un hallazgo real; corregir la causa. Si el linter se equivoca, justificar el `//nolint:<regla>` con un comentario breve.

## Nombres e identificadores

- Nombres idiomáticos: `camelCase` para privados, `PascalCase` para exportados. Sin prefijos tipo `I` en interfaces ni `_` en identificadores.
- Interfaces con pocos métodos y nombradas por lo que hacen (`Reader`, `AppointmentRepository`), no por su implementación (`AppointmentRepositoryImpl`).
- Comentario de documentación (`// Nombre ...`) en todo identificador exportado (función, tipo, constante). Si el nombre ya lo dice todo, el comentario es una frase corta, no relleno.
- Evitar nombres genéricos (`data`, `info`, `obj`) cuando el dominio ofrece un nombre más preciso.

## Manejo de errores

- Nunca descartar un error (`_ = err` o ignorarlo) salvo que sea explícitamente seguro y se documente por qué.
- Envolver errores con contexto usando `fmt.Errorf("haciendo X: %w", err)`, nunca con `%v` cuando el llamador pueda necesitar `errors.Is`/`errors.As`.
- Los errores de dominio (`domain.ErrNotFound`, `ErrAlreadyExists`, etc.) son el vocabulario compartido entre capas; no inventar tipos de error nuevos para casos ya cubiertos por un sentinel existente.
- `panic` solo para errores de programación irrecuperables (invariantes rotos), nunca para flujo de control ni para errores esperables (entrada inválida, fallo de red, fila no encontrada).
- No usar `recover` fuera del middleware de recovery centralizado (`internal/middlewares`).

## Diseño de paquetes y capas

- Las interfaces se declaran junto a quien las consume (el `usecase`), no junto a la implementación (`repository`). No crear una interfaz hasta que exista un segundo consumidor o se necesite para test doubles.
- Dirección de dependencias: `handler → usecase → repository interface ← repository impl`, y todas hacia `domain`. `domain` no importa nada de las otras capas.
- Un paquete no debe importar `internal/handler` desde `internal/usecase` ni `internal/repository` (evitar dependencias invertidas o ciclos).
- Funciones cortas y con una responsabilidad. Si una función usecase/handler supera ~40-50 líneas o mezcla validación + orquestación + mapeo de errores, dividirla.
- Evitar parámetros booleanos ambiguos (`func Create(..., force bool)`); preferir una opción con nombre o dos funciones.

## Concurrencia y contexto

- Toda función que haga I/O (DB, red, cache) recibe `context.Context` como primer parámetro y lo propaga; nunca se usa `context.Background()` dentro de una capa que ya tiene un `ctx` disponible.
- Ningún goroutine se lanza sin una forma clara de terminarlo (cancelación por `ctx`, `sync.WaitGroup`, canal de cierre). Evitar goroutines "fire and forget" que puedan filtrarse (leak).
- Proteger estado compartido mutable con el mecanismo más simple que resuelva el caso (mutex, canal); no optimizar prematuramente con estructuras lock-free.

## Testing

- Tests en Go estándar (`testing`), estilo table-driven cuando haya varios casos de entrada/salida.
- Los tests de `usecase` dependen de interfaces (mocks/fakes), no de implementaciones concretas de `postgres`.
- Nombrar los tests por comportamiento (`TestCreate_RejectsPastStartTime`), no por implementación interna.
- `go test -race -cover ./...` (`make test`) debe pasar; no mergear código que introduzca una data race conocida.

## Dependencias

- No agregar una librería externa para algo que la stdlib ya resuelve bien.
- Toda nueva dependencia directa se agrega con `go get` y se corre `make tidy`; no editar `go.sum` a mano.
