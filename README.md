# Velora

Velora is a local-first media server with a bundled React web client. The Go server, FFmpeg, and the built web assets
ship as a single container; Apple-device browser playback is the first client target.

The project is pre-release. SQLite is the default, Postgres is opt-in, and the server exposes health, library
registration, and synchronous incremental media scanning APIs. ffprobe inspection is the next milestone.

## Goals

- One-container deploy that works the same locally (Docker Compose) and on real hosts (Unraid, etc.).
- Container paths (`/config`, `/cache`, `/media`) as the only contract; host paths are a compose concern.
- Direct play whenever the client can handle the original media; FFmpeg-driven transcoding otherwise.
- Lightweight, opinionated, easy to read.

## Stack

- Go (stdlib HTTP), `database/sql` over `modernc.org/sqlite` (default) or `pgx/v5` (Postgres).
- Goose migrations, embedded per dialect, applied on startup.
- React + Vite + TypeScript, served by the Go binary with SPA deep-link routing.
- FFmpeg in the runtime image. Docker Compose for local dev. GitHub Actions for CI.

## Quick start

```bash
cp .env.example .env       # defaults work for the SQLite stack
./velora create            # build the image and bring the stack up on :8080
curl http://localhost:8080/api/health
```

The `./velora` wrapper covers the common lifecycle commands:

```bash
./velora create [--postgres]   # build + up; --postgres enables the bundled Postgres profile
./velora destroy               # stop containers, keep volumes and host bind mounts
./velora clean                 # stop containers, drop compose-managed volumes (SQLite db is lost)
./velora status                # docker compose ps
./velora logs                  # follow container logs
./velora lint                  # run the pinned Go linter
```

For step-by-step setup, deeper config patterns, and troubleshooting, see [`docs/`](docs/).

## HTTP API

### `GET /api/health`

```bash
curl http://localhost:8080/api/health
# {"status":"ok","database":"ok","databaseDriver":"sqlite"}
```

`status` is `degraded` (HTTP 503) when the database ping fails. `databaseDriver` echoes `VELORA_DATABASE_DRIVER`.

### `GET /api/libraries`

Lists configured media roots, ordered by insertion:

```bash
curl http://localhost:8080/api/libraries
# [{"id":1,"name":"Movies","path":"/media/movies","createdAt":"...","updatedAt":"..."}]
```

### `POST /api/libraries`

Creates a library. `name` and `path` are required; `path` must be unique and must use the container path.

```bash
curl -X POST http://localhost:8080/api/libraries \
  -H 'Content-Type: application/json' \
  -d '{"name":"Movies","path":"/media/movies"}'
# 201 with the created library
```

Returns `400` for missing fields, `409` if the path is already configured.

### `POST /api/libraries/{id}/scan`

Walks one library, persists supported video files, and returns reconciliation counts:

```bash
curl -X POST http://localhost:8080/api/libraries/1/scan
# {"libraryId":1,"discovered":2,"added":2,...}
```

Scans skip symlinks and unsupported files. A second concurrent scan of the same library returns `409`.

### `GET /api/libraries/{id}/files`

Lists the persisted file inventory, including files currently marked `missing`:

```bash
curl http://localhost:8080/api/libraries/1/files
```

## Local development

Server tests — the repo carries checked-in build/mod caches, so point both vars at them:

```bash
cd apps/server
GOCACHE=$PWD/.cache/go-build GOMODCACHE=$PWD/.cache/go-mod go test -race ./...
# Single package:
GOCACHE=$PWD/.cache/go-build GOMODCACHE=$PWD/.cache/go-mod go test ./internal/libraries -v
```

Web client live-reload (the container must be up so the proxy has a target):

```bash
cd apps/web
npm run dev                # vite on :5173, /api proxies to the running container on :8080
```

For a production-shape build of the web client: `npm run build` (`tsc` + `vite build`).

## Configuration

The app reads and writes three container mount points. **Use the container paths in code and API payloads, not the
`dev/*` host paths** — that contract lets the same binary move from local Docker to any other host by remapping mounts.

| Container path | Purpose                                                          |
|----------------|------------------------------------------------------------------|
| `/config`      | Durable runtime config (SQLite db at `/config/velora.db`)        |
| `/cache`       | Rebuildable derived artifacts (transcodes, thumbnails)           |
| `/media`       | Source media (treat as user-owned input)                         |

Environment variables (container side; defaults shown):

| Variable                 | Default             | Notes                                                |
|--------------------------|---------------------|------------------------------------------------------|
| `PORT`                   | `8080`              | HTTP listen port                                     |
| `WEB_DIST_DIR`           | `/app/web`          | Built web assets the Go server serves                |
| `VELORA_CONFIG_DIR`      | `/config`           | Override the durable-state mount point               |
| `VELORA_CACHE_DIR`       | `/cache`            | Override the cache mount point                       |
| `VELORA_MEDIA_DIR`       | `/media`            | Override the source-media mount point                |
| `VELORA_DATABASE_DRIVER` | `sqlite`            | `sqlite` or `postgres`                               |
| `VELORA_DATABASE_URL`    | `/config/velora.db` | SQLite file path or Postgres DSN                     |

Host-side overrides (where bind mounts come from on your machine): `VELORA_HOST_CONFIG_DIR`, `VELORA_HOST_CACHE_DIR`,
`VELORA_HOST_MEDIA_DIR`, `VELORA_HTTP_PORT`. Defaults map to `./dev/*` and `:8080`.

To use an external Postgres, set `VELORA_DATABASE_DRIVER=postgres` and `VELORA_DATABASE_URL=postgres://…` in `.env`,
then `./velora create` (no `--postgres` needed; that flag is only for the bundled compose service).

## Project layout

```text
apps/
  server/                   Go API + media server (module: github.com/getvelora/velora/apps/server)
    cmd/velora/             Entrypoint with graceful shutdown
    internal/               Packages for database, libraries, media files, migrations, storage, health, and web
  web/                      React/Vite/TypeScript client
dev/                        Local bind-mount sources (config, cache, media)
.github/workflows/          CI pipeline
docker-compose.yml          Local stack
Dockerfile                  Multi-stage build (web + server → alpine + ffmpeg)
velora                      Convenience CLI around docker compose
```

## CI

GitHub Actions runs server formatting/build/vet/race tests, pinned Go linting, the web build, and a Docker smoke test
on every push and PR to `develop`. Contributors can enable the committed pre-commit lint hook with
`git config core.hooksPath .githooks`. See `.github/workflows/ci.yml`.

Dependabot runs weekly for Go, npm, GitHub Actions, and Docker dependencies. Patch updates are grouped and
automatically squash-merged after all required checks pass. GitHub then closes the PR and deletes its branch. Minor
and major updates intentionally remain open for manual review; failed patch updates also remain open. When dependency
PRs overlap in the same manifest or lockfile, merge them sequentially: close superseded PRs, then refresh the next
distinct update against the latest `develop`.

## Status

Implemented:

- One-container Docker deploy with graceful shutdown and FFmpeg installed.
- SQLite default, optional bundled or external Postgres.
- Goose migrations applied at startup, with per-dialect embedded SQL.
- `/api/health` and `/api/libraries` (list + create).
- Synchronous incremental media discovery with persisted missing/restored state.
- SPA deep-link routing for the served web client.
- GitHub Actions CI on push + PR.

## Roadmap

The next milestone is ffprobe-based stream and container inspection. See [`ROADMAP.md`](ROADMAP.md) for the planned
path through metadata, playback, hardware-accelerated transcoding, 4K/HDR support, household profiles, portable
deployment, and instance-to-instance library sharing.
