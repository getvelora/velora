# Getting Started

A walkthrough from cloning the repo to your first library. ~5 minutes.

## Prerequisites

- Docker Desktop (macOS, Windows with WSL2) or Docker Engine + Compose (Linux).
- Port `8080` free on the host. If it's taken, set `VELORA_HTTP_PORT=8081` (or any free port) in `.env` later.

## 1. Clone and configure

```bash
git clone https://github.com/getvelora/velora.git
cd velora
cp .env.example .env
```

The default `.env` runs the SQLite stack on port 8080 with bind mounts under `./dev/`. No edits required unless you
want a different port, Postgres, or different host paths — see [Configuration](configuration.md).

## 2. Build and start

```bash
./velora create
```

This runs a multi-stage Docker build (web client → Go server → Alpine + FFmpeg runtime) and brings up one container.
First build takes ~30 seconds; rebuilds use the layer cache.

Watch the boot logs if you want to see migrations applying:

```bash
./velora logs
```

When you see `velora server listening on :8080`, it's ready.

## 3. Verify

```bash
curl http://localhost:8080/api/health
# {"status":"ok","database":"ok","databaseDriver":"sqlite"}
```

Then open <http://localhost:8080> in a browser. You should see the Velora web shell with a green status pill.

## 4. Create a library

A library is a configured root path inside `/media` that Velora will scan for media files. Right now libraries are
configuration-only — the scanner ships in the next milestone.

Drop a test file where the container can see it:

```bash
mkdir -p dev/media/movies
# put a file or two in dev/media/movies/
```

Tell Velora about that root:

```bash
curl -X POST http://localhost:8080/api/libraries \
  -H 'Content-Type: application/json' \
  -d '{"name":"Movies","path":"/media/movies"}'
```

You should get back a `201` with the created library, including its `id`, `createdAt`, and `updatedAt`.

> **Important**: the `path` field is the **container** path (`/media/movies`), not the host path (`./dev/media/movies`).
> The same value works the same way whether you run Velora on your laptop or on a NAS — that's the whole point of
> [container paths](configuration.md#container-paths).

List your libraries to confirm:

```bash
curl http://localhost:8080/api/libraries
```

## 5. Stop and clean up

```bash
./velora destroy     # stop containers, keep your data (the SQLite db survives)
./velora clean       # stop containers AND drop compose volumes (data is lost)
```

`./velora destroy` is what you want for everyday "I'm done for the day"; `./velora clean` is for "give me a fresh
state."

## Next steps

- Point Velora at real media paths on your machine or a NAS: [Configuration](configuration.md).
- See what the API can do: [API Reference](api-reference.md).
- Stuck? [Troubleshooting](troubleshooting.md).
