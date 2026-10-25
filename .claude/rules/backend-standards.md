# Estándares de backend

Reglas de diseño de API, datos y operación para este servicio HTTP.

## Diseño de API

- Rutas REST en minúsculas, en plural para colecciones (`/api/v1/stores`, `/api/v1/stores/{id}`), versionadas bajo `/api/v1`.
- El verbo HTTP expresa la acción: `GET` no muta estado, `POST` crea, `PATCH` actualiza parcialmente, `DELETE` elimina. No usar `GET` con efectos secundarios.
- Toda respuesta de error sigue el formato `application/problem+json` ya establecido en `handler.RespondError`; no inventar un segundo formato de error en un handler nuevo.
- Los códigos de estado se derivan de errores de dominio (`ErrNotFound` → 404, `ErrAlreadyExists` → 409, `ErrUnauthorized` → 401, `ErrForbidden` → 403, `ErrInvalidInput` → 400), nunca se hardcodea un status ad-hoc en el handler cuando ya existe un sentinel adecuado.
- Los recursos protegidos se montan con el middleware de auth en el router (`withAuth(...)`), no con checks manuales dentro del handler.

## Validación de entrada

- Todo payload de entrada (body, query params relevantes) se valida con `internal/validator` antes de llegar al usecase. El usecase no debe asumir que la entrada ya es válida solo porque vino del handler sin pasar por un validador explícito.
- Nunca confiar en datos de entrada para controlar rutas de archivos, queries SQL crudas o comandos del sistema. Toda consulta SQL pasa por sqlc/parámetros, nunca por concatenación de strings.
- Rechazar al borde (handler/usecase) las entradas inválidas con `domain.ErrInvalidInput`; no dejar que un dato malformado se propague hasta la capa de repositorio.

## Seguridad

- Nunca loggear contraseñas, tokens (JWT, refresh), ni secretos de configuración, completos o parciales.
- Los secretos (JWT secret, credenciales de DB/Redis) se leen de variables de entorno (`.env`, nunca committeado); no hardcodear secretos en código, tests, ni en `Makefile`/`docker-compose.yml` de producción.
- Toda comparación de contraseñas/tokens sensibles usa las utilidades ya existentes en `internal/auth` (hashing, verificación constante en tiempo), no comparación directa de strings (`==`).
- Aplicar el principio de menor privilegio: un endpoint que solo necesita leer no debe exponer una ruta de escritura "por si acaso".
- CORS y cabeceras de seguridad se configuran centralizadamente en `middlewares.SecurityHeaders`, no por handler individual.

## Capa de datos

- Todo cambio de esquema pasa por una migración versionada en `migrations/` (`make migrate-create`), nunca se modifica el esquema manualmente contra una base de datos compartida.
- Las migraciones deben ser reversibles: todo `.up.sql` tiene su `.down.sql` correspondiente que revierte el cambio.
- Las queries viven en `internal/repository/postgres/queries/*.sql` y se generan con sqlc (`make generate`); no escribir SQL dinámico armado por concatenación en el código Go.
- Operaciones que deben ser atómicas (p. ej. verificar solapamiento y crear un registro) se ejecutan dentro de una transacción explícita, no como dos llamadas separadas que puedan dejar el estado inconsistente ante una falla a mitad de camino.
- El repositorio es responsable de traducir errores de infraestructura (`pgx.ErrNoRows`, violación de constraint) a errores de dominio; el usecase nunca debe inspeccionar tipos de error específicos de pgx/sqlc.

## Observabilidad

- Usar el logger estructurado (`log/slog`) inyectado, nunca `fmt.Println`/`log.Println` para logs de aplicación.
- Loggear errores no clasificados (los que caen al 500 genérico) con suficiente contexto (operación, id de recurso cuando aplique) para poder diagnosticarlos sin reproducir el request.
- No loggear en nivel `Info`/`Debug` dentro de loops calientes o por cada fila procesada; reservarlo para eventos de negocio relevantes (arranque, conexión, request fallido).

## Configuración y despliegue

- La app lee configuración exclusivamente de variables de entorno (ver `.env.example` como fuente de verdad de qué variables existen); si se agrega una variable nueva, se documenta ahí.
- No asumir que el entorno de producción tiene los mismos defaults que `Makefile`/`docker-compose.yml` (esos son solo para desarrollo local).
- Cualquier recurso externo (DB, Redis) se abre con timeouts explícitos y se cierra (`defer pool.Close()` / `defer client.Close()`) para evitar fugas de conexiones.
