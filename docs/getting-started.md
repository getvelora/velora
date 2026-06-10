# Getting Started

Velora ships as one prebuilt container containing the Go server, web client, and FFmpeg. Running Velora does not
require cloning the repository, installing Node or Go, building an image, or creating an `.env` file.

## Prerequisites

- Docker Engine, Docker Desktop, or another OCI-compatible container manager.
- A host directory containing your movies and TV files.
- Port `8080` available, or another host port of your choice.

## Docker Compose (recommended)

Download the deployment example:

```bash
mkdir velora
cd velora
curl -LO https://github.com/getvelora/velora/releases/latest/download/compose.yml
```

Edit `compose.yml` and replace `/path/to/your/media` with the absolute host path containing your media. Then start:

```bash
docker compose up -d
docker compose logs -f velora
```

Compose pulls `ghcr.io/getvelora/velora:latest`, creates durable named volumes for `/config` and `/cache`, mounts your
media read-only at `/media`, and publishes Velora at <http://localhost:8080>.

To upgrade later:

```bash
docker compose pull
docker compose up -d
```

SQLite is enabled by default. If you already operate Postgres, see
[External Postgres](configuration.md#external-postgres) before starting the container.

## Docker CLI

The equivalent command without Compose is:

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

Replace `/path/to/your/media` before running it. Upgrade by pulling the new image and recreating the container with
the same mounts.

## Verify

```bash
curl http://localhost:8080/api/health
# {"status":"ok","database":"ok","databaseDriver":"sqlite"}
```

Then open <http://localhost:8080>.

When Postgres is configured, the same response reports `"databaseDriver":"postgres"`.

## Create And Scan A Library

The API uses container paths, so a library under your mounted media must begin with `/media`:

```bash
curl -X POST http://localhost:8080/api/libraries \
  -H 'Content-Type: application/json' \
  -d '{"name":"Movies","path":"/media/movies"}'
```

Use the returned ID to scan and list files:

```bash
curl -X POST http://localhost:8080/api/libraries/1/scan
curl http://localhost:8080/api/libraries/1/files
```

## Stop Or Remove

With Compose:

```bash
docker compose stop          # stop while preserving all data
docker compose down          # remove the container, preserve named volumes
docker compose down --volumes # also delete Velora's config/database and cache
```

The public setup is intentionally separate from contributor development. Developers working from source should use
[`CONTRIBUTING.md`](../CONTRIBUTING.md) and `./velora`, which targets `compose.dev.yml`.
