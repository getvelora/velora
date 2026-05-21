# Velora

Velora is an open-source, local-first media server and web client focused on simple deployment, efficient playback, and
persistent media caching. It is being built for container-based environments, with Apple-device browser playback as the
first client target.

The project is early. The current slice boots a Dockerized Go server, serves the bundled React web client, uses SQLite
by default, and includes FFmpeg in the runtime image for future media probing and transcoding work. External Postgres is
available as an advanced deployment option.

## Goals

- Run locally with Docker Compose and later in any container environment with the same container paths.
- Keep durable server state under `/config` so containers can be recreated without data loss.
- Prefer direct play whenever the client can handle the original media.
- Support hardware-aware transcoding through FFmpeg.
- Keep derived media artifacts in a persistent cache.
- Use a focused web client before native platform clients.
- Stay focused, lightweight, and operationally simple.

## Stack

- Go server using the standard library HTTP router.
- SQLite by default, with optional external Postgres.
- Goose for database migrations, embedded into the binary per dialect and applied on startup.
- React, Vite, and TypeScript for the web client, with SPA deep-link routing falling back to `index.html`.
- FFmpeg in the server image for media inspection and future streaming work.
- Docker Compose for local development.
- GitHub Actions CI builds and tests both apps on push and pull request.

## Local Development

Use the root helper for common project tasks:

```bash
./velora help              # list available commands
./velora create            # build the image and start the SQLite stack on http://localhost:8080
./velora create --postgres # same but with the bundled Postgres compose profile
./velora status            # docker compose ps
./velora logs              # follow container logs
./velora destroy           # stop containers, KEEP named volumes and host bind mounts
./velora clean              # stop containers, DROP compose-managed volumes (SQLite db is lost)
./velora commit            # commit staged changes with a Conventional Commit message
```

First-time setup:

```bash
cp .env.example .env       # defaults work for the SQLite stack; no edits required
./velora create
```

Open the web client at <http://localhost:8080>.

### Web client dev server

For tight web iteration without rebuilding the container, run Vite directly while the stack is up:

```bash
cd apps/web
npm run dev                # vite on :5173, /api proxies to the running container on :8080
```

## HTTP API

Two endpoints are live today.

### `GET /api/health`

```bash
curl http://localhost:8080/api/health
```

```json
{
  "status": "ok",
  "database": "ok",
  "databaseDriver": "sqlite"
}
```

`status` is `ok` when the database ping succeeds and `degraded` (with HTTP 503) when it fails. `databaseDriver` reflects
`VELORA_DATABASE_DRIVER` — `sqlite` or `postgres`.

### `GET /api/libraries`

Returns the configured media roots, ordered by insertion:

```bash
curl http://localhost:8080/api/libraries
```

```json
[
  {
    "id": 1,
    "name": "Movies",
    "path": "/media/movies",
    "createdAt": "2026-05-20T18:23:01.123456789Z",
    "updatedAt": "2026-05-20T18:23:01.123456789Z"
  }
]
```

### `POST /api/libraries`

Create a library. `name` and `path` are required; the path must be unique and must use the container path (`/media/...`), not the host path.

```bash
curl -X POST http://localhost:8080/api/libraries \
  -H 'Content-Type: application/json' \
  -d '{"name":"Movies","path":"/media/movies"}'
```

Returns `201` with the created library. `409 Conflict` if the path is already configured; `400 Bad Request` for missing
fields or invalid JSON.

## Local Mounts

Docker Compose maps these development folders into the Velora container:

- `dev/config` -> `/config`
- `dev/cache` -> `/cache`
- `dev/media` -> `/media`

**Use the container paths (`/media`, `/cache`, `/config`) in code and in API payloads, not the `dev/*` host paths.**
That contract lets the same binary move from local Docker to Unraid (or any other host) just by remapping bind mounts.

The default SQLite database is stored at:

```text
dev/config/velora.db
```

In a container environment, map `/config` to durable app storage, `/cache` to cache storage, and `/media` to source
media. For example:

```text
/config -> appdata/velora
/cache  -> cache/velora
/media  -> media share
```

`./velora destroy` keeps data under those mounts; `./velora clean` drops compose-managed volumes (the SQLite db included).

## Optional Postgres

SQLite is the default and requires no additional services. To use Postgres locally, enable the bundled profile:

```bash
./velora create --postgres
```

`./velora create --postgres` brings up a `postgres:18-alpine` service and wires Velora's database connection at it.
Credentials come from `.env`:

```env
POSTGRES_DB=velora
POSTGRES_USER=velora
POSTGRES_PASSWORD=velora
POSTGRES_PORT=5432
```

Velora can also connect to any external Postgres service by setting `VELORA_DATABASE_DRIVER=postgres` and
`VELORA_DATABASE_URL` to that service's connection string before running Docker Compose directly.

## Environment Variables

Container variables (defaults shown):

| Variable                 | Default                | Purpose                                                  |
|--------------------------|------------------------|----------------------------------------------------------|
| `PORT`                   | `8080`                 | HTTP listen port                                         |
| `WEB_DIST_DIR`           | `/app/web`             | Directory of built web assets the Go server serves       |
| `VELORA_CONFIG_DIR`      | `/config`              | Durable runtime config (SQLite db lives here by default) |
| `VELORA_CACHE_DIR`       | `/cache`               | Rebuildable derived artifacts (transcodes, thumbnails)   |
| `VELORA_MEDIA_DIR`       | `/media`               | Source media (treat as user-owned input)                 |
| `VELORA_DATABASE_DRIVER` | `sqlite`               | `sqlite` (modernc.org/sqlite) or `postgres` (pgx/v5)     |
| `VELORA_DATABASE_URL`    | `/config/velora.db`    | SQLite file path or Postgres DSN                         |

Host-side compose overrides (control where the container's mount points come from on your machine):

| Variable                  | Default          |
|---------------------------|------------------|
| `VELORA_HOST_CONFIG_DIR`  | `./dev/config`   |
| `VELORA_HOST_CACHE_DIR`   | `./dev/cache`    |
| `VELORA_HOST_MEDIA_DIR`   | `./dev/media`    |
| `VELORA_HTTP_PORT`        | `8080`           |

## Project Layout

```text
apps/
  server/                          Go module: github.com/mdelle/velora/apps/server
    cmd/velora/                    Entrypoint (signal-driven graceful shutdown)
    internal/
      database/                    sql.DB factory for sqlite/postgres
      env/                         Shared env.OrDefault helper
      health/                      GET /api/health
      libraries/                   GET/POST /api/libraries + repository
      migrations/                  Goose runner + embedded per-dialect SQL
        sql/sqlite/
        sql/postgres/
      storage/                     Container mount-point bootstrapping
      web/                         SPA-aware static file handler
  web/                             React/Vite/TypeScript client
    src/
dev/
  cache/                           Local derived cache artifacts
  config/                          Local durable app state (SQLite db)
  media/                           Local test media
.github/workflows/                 CI pipeline (Go + web build/test)
docker-compose.yml                 Local stack definition
Dockerfile                         Multi-stage build (web + server → alpine + ffmpeg)
velora                             Convenience CLI wrapping docker compose
```

## Verification

Server tests — the repo carries checked-in build/mod caches, so point both vars at them or Go will rebuild from scratch:

```bash
cd apps/server
GOCACHE=$PWD/.cache/go-build GOMODCACHE=$PWD/.cache/go-mod go test -race ./...

# Single package:
GOCACHE=$PWD/.cache/go-build GOMODCACHE=$PWD/.cache/go-mod go test ./internal/libraries -run TestRepositoryCreateAndList -v
```

Web client typecheck + production build:

```bash
cd apps/web
npm run build              # tsc + vite build
```

Full stack smoke:

```bash
./velora create
curl http://localhost:8080/api/health
```

## Continuous Integration

GitHub Actions runs the server and web jobs in `.github/workflows/ci.yml` on every push to `main` or `develop`, and on
pull requests targeting either branch. The server job uses the Go version pinned in `apps/server/go.mod`; the web job
uses Node 24. Both jobs cache their dependency stores. Tests run with `-race`.

## Current Status

Implemented:

- Dockerized Go server with graceful shutdown.
- SQLite default database under `/config`.
- Optional Postgres service via the `postgres` compose profile.
- `/api/health` endpoint reports overall status and the active DB driver.
- `/api/libraries` endpoint lists and creates configured media roots.
- `libraries` table with per-dialect Goose migrations applied at startup.
- React web shell served by the Velora image, with SPA deep-link routing.
- FFmpeg installed in the runtime image.
- GitHub Actions CI for server and web builds.

Next planned milestone:

- Begin scanning files from `/media`.
- Expand `/api/libraries` to GET-by-id, PUT, and DELETE.
- Introduce `media_files` and the scanner pipeline.
