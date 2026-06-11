# Velora

Velora is a local-first media server with a bundled React web client. The Go server, FFmpeg, and built web assets ship
as a single container; Apple-device browser playback is the first client target.

The project is pre-release. SQLite is the default, Postgres is opt-in, and the server currently supports health
checks, library registration, synchronous incremental media scanning, and ffprobe-based media inspection.

## Goals

- Ship one portable container for Docker, Unraid, and other container hosts.
- Keep `/config`, `/cache`, and `/media` as the stable deployment contract.
- Direct-play compatible media and use FFmpeg when conversion is required.
- Stay lightweight, opinionated, and easy to operate.

## Stack

- Go standard-library HTTP server with SQLite or Postgres through `database/sql`.
- Goose database migrations applied at startup.
- React, Vite, and TypeScript web client served by the Go binary.
- FFmpeg in the runtime image, with GitHub Actions for CI.

## Quick Start

Velora publishes a complete image to `ghcr.io/getvelora/velora`. End users do not need this repository, Node, Go, an
`.env` file, or a local image build.

```bash
mkdir velora
cd velora
curl -LO https://github.com/getvelora/velora/releases/latest/download/compose.yml
# Replace /path/to/your/media in compose.yml.
docker compose up -d
curl http://localhost:8080/api/health
```

Compose creates durable config and cache volumes, mounts your media read-only, and serves Velora at
<http://localhost:8080>.

See [Getting Started](docs/getting-started.md) for Docker CLI setup, library creation, upgrades, and removal.

## Documentation

- [Documentation index](docs/README.md)
- [Getting started](docs/getting-started.md)
- [Configuration](docs/configuration.md)
- [API reference](docs/api-reference.md)
- [Troubleshooting](docs/troubleshooting.md)
- [Development guide](docs/development.md)
- [Contributing](CONTRIBUTING.md)
- [Roadmap](ROADMAP.md)

## Status

Implemented:

- One-container deployment with the Go server, web client, and FFmpeg.
- SQLite by default, with optional external Postgres.
- Embedded per-dialect migrations applied at startup.
- Health and library registration APIs.
- Incremental media discovery with persisted missing and restored states.
- Typed container, video, audio, subtitle, codec, and HDR-related inspection through ffprobe.
- SPA deep-link routing for the bundled web client.

The next milestone is grouping inspected files into movies, series, seasons, and episodes. See
[ROADMAP.md](ROADMAP.md) for the planned path through metadata, playback, transcoding, household profiles, portable
deployment, and library sharing.
