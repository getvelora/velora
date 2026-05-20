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
- Goose for database migrations.
- React, Vite, and TypeScript for the web client.
- FFmpeg in the server image for media inspection and future streaming work.
- Docker Compose for local development.

## Local Development

Use the root helper for common project tasks:

```bash
./velora help
```

Create a local environment file:

```bash
cp .env.example .env
```

Start the stack:

```bash
./velora create
```

Start the stack with local Postgres instead of SQLite:

```bash
./velora create --postgres
```

Open the web client:

```text
http://localhost:8080
```

Check server health:

```bash
curl http://localhost:8080/api/health
```

Expected response:

```json
{
  "status": "ok",
  "database": "ok",
  "databaseDriver": "sqlite"
}
```

`status` is `ok` when the database ping succeeds and `degraded` (with HTTP 503) when it fails. `databaseDriver` reflects `VELORA_DATABASE_DRIVER` — `sqlite` or `postgres`.

Stop the stack:

```bash
./velora destroy
```

Remove compose-managed volumes:

```bash
./velora clean
```

## Local Mounts

Docker Compose maps these development folders into the Velora container:

- `dev/config` -> `/config`
- `dev/cache` -> `/cache`
- `dev/media` -> `/media`

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

Destroying and recreating the container does not remove data stored in those mounted paths.

## Optional Postgres

SQLite is the default and requires no additional services. To use Postgres, provide database settings and enable the
Postgres profile:

```bash
./velora create --postgres
```

`./velora create --postgres` uses the bundled Postgres profile and sets Velora's database connection for the local
Postgres service. The local Postgres settings come from `.env`:

```env
POSTGRES_DB=velora
POSTGRES_USER=velora
POSTGRES_PASSWORD=velora
POSTGRES_PORT=5432
```

Velora can also connect to any external Postgres service by setting `VELORA_DATABASE_DRIVER=postgres` and
`VELORA_DATABASE_URL` to that service's connection string before running Docker Compose directly.

## Project Layout

```text
apps/
  server/   Go API and media server
  web/      React web client
dev/
  cache/    Local derived cache artifacts
  config/   Local durable app state
  media/    Local test media
```

## Verification

Run server tests:

```bash
cd apps/server
GOCACHE=$PWD/.cache/go-build GOMODCACHE=$PWD/.cache/go-mod go test ./...
```

Build the web client:

```bash
cd apps/web
npm run build
```

Build and start the Docker image:

```bash
./velora create
```

Commit staged changes:

```bash
./velora commit
```

## Current Status

Implemented:

- Dockerized Go server.
- SQLite default database under `/config`.
- Optional Postgres service.
- `/api/health` endpoint reports overall status and the active DB driver.
- Goose database migrations applied at startup, with per-dialect embedded SQL.
- React web shell served by the Velora image.
- FFmpeg installed in the runtime image.

Next planned milestone:

- Add a `libraries` table.
- Add `/api/libraries`.
- Begin scanning files from `/media`.
