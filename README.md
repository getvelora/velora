# Velora

Velora is a local-first media server with a bundled React web client. The Go server, FFmpeg, and the built web assets
ship as a single container; Apple-device browser playback is the first client target.

The project is pre-release. SQLite is the default, Postgres is opt-in, and the server exposes health, library
registration, and synchronous incremental media scanning APIs. ffprobe inspection is the next milestone.

## Goals

- One published container that runs consistently across Docker, Unraid, and other container hosts.
- Container paths (`/config`, `/cache`, `/media`) as the only contract; host paths are a compose concern.
- Direct play whenever the client can handle the original media; FFmpeg-driven transcoding otherwise.
- Lightweight, opinionated, easy to read.

## Stack

- Go (stdlib HTTP), `database/sql` over `modernc.org/sqlite` (default) or `pgx/v5` (Postgres).
- Goose migrations, embedded per dialect, applied on startup.
- React + Vite + TypeScript, served by the Go binary with SPA deep-link routing.
- FFmpeg in the runtime image. Docker Compose for local dev. GitHub Actions for CI.

## Install

Velora publishes a complete image to `ghcr.io/getvelora/velora`. It already contains the server, web client, and
FFmpeg. End users do not need this repository, Node, Go, `.env`, or an image build.

### Docker Compose (recommended)

```bash
curl -LO https://github.com/getvelora/velora/releases/latest/download/compose.yml
# Edit /path/to/your/media in compose.yml.
docker compose up -d
curl http://localhost:8080/api/health
```

Compose pulls `ghcr.io/getvelora/velora:latest`, creates durable config/cache volumes, and mounts your media read-only.

### Docker CLI

```bash
docker run -d \
  --name velora \
  --restart unless-stopped \
  -p 8080:8080 \
  -v velora-config:/config \
  -v velora-cache:/cache \
  -v /path/to/your/media:/media:ro \
  ghcr.io/getvelora/velora:latest
```

Replace `/path/to/your/media` with the real host path. See [`docs/getting-started.md`](docs/getting-started.md) for the
full setup.

SQLite is the zero-configuration default. To use an existing Postgres server, configure the same image with
`VELORA_DATABASE_DRIVER=postgres` and a complete `VELORA_DATABASE_URL`; no separate Velora image or bundled database
is required. See [External Postgres](docs/configuration.md#external-postgres).

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

This section is for contributors working from a repository clone. It is not the public installation path.

```bash
cp .env.development.example .env   # optional local overrides
./velora create                    # build and start the server-only development image
./velora web-build                 # rebuild mounted web assets without rebuilding the app image
```

`./velora` always uses `compose.dev.yml`. The public `compose.yml` never builds source or mounts development assets.

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

For a production-shape build without installing Node locally:

```bash
./velora web-build
```

This runs the dependency check and `npm run build` in a disposable Node container. The generated `apps/web/dist`
directory is mounted into the running Velora container, so refreshing the browser serves the new assets without
rebuilding or restarting the server. `./velora create` builds these assets only when `dist` is missing, then builds
the `server-runtime` target without running the release Dockerfile's web stage.

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

Contributor-only host overrides are documented in [CONTRIBUTING.md](CONTRIBUTING.md). Public deployments configure
mounts, ports, and optional environment values directly in Compose, `docker run`, or their container manager.

## Project layout

```text
apps/
  server/                   Go API + media server (module: github.com/getvelora/velora/apps/server)
    cmd/velora/             Entrypoint with graceful shutdown
    internal/               Packages for database, libraries, media files, migrations, storage, health, and web
  web/                      React/Vite/TypeScript client
dev/                        Local bind-mount sources (config, cache, media)
.github/workflows/          CI pipeline
compose.yml                 Public GHCR deployment example
compose.dev.yml             Contributor development stack
Dockerfile                  Release image + server-only development target
velora                      Contributor CLI around compose.dev.yml
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
