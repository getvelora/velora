# Configuration

Velora's public deployment contract is the published container image:

```text
ghcr.io/getvelora/velora:latest
```

The image already includes the server, built web client, and FFmpeg. Configure it through your Compose file,
`docker run` command, or container manager. A repository checkout and `.env` file are not required.

## Required Mounts

Velora uses three stable container paths:

| Container path | Access     | Purpose                                                   |
|----------------|------------|-----------------------------------------------------------|
| `/config`      | read/write | Durable state and the default SQLite database             |
| `/cache`       | read/write | Rebuildable thumbnails, previews, and transcode artifacts |
| `/media`       | read-only  | User-owned source media                                   |

The public `compose.yml` uses named volumes for `/config` and `/cache`. Replace `/path/to/your/media` with the actual
absolute host path:

```yaml
services:
  velora:
    image: ghcr.io/getvelora/velora:latest
    restart: unless-stopped
    ports:
      - "8080:8080"
    volumes:
      - velora-config:/config
      - velora-cache:/cache
      - /mnt/media:/media:ro

volumes:
  velora-config:
  velora-cache:
```

Container paths are also the API contract. If the host directory `/mnt/media/movies` is mounted at `/media`, register
the library as `/media/movies`, never `/mnt/media/movies`.

## Port

The container listens on port `8080`. Change only the host side to expose a different port:

```yaml
ports:
  - "8081:8080"
```

## Environment Variables

Defaults require no environment configuration.

| Variable                 | Default             | Purpose                                         |
|--------------------------|---------------------|-------------------------------------------------|
| `PORT`                   | `8080`              | HTTP port inside the container                  |
| `WEB_DIST_DIR`           | `/app/web`          | Built web assets included in the image          |
| `VELORA_CONFIG_DIR`      | `/config`           | Durable configuration mount                     |
| `VELORA_CACHE_DIR`       | `/cache`            | Rebuildable cache mount                         |
| `VELORA_MEDIA_DIR`       | `/media`            | Source-media mount                              |
| `VELORA_DATABASE_DRIVER` | `sqlite`            | `sqlite` or `postgres`                          |
| `VELORA_DATABASE_URL`    | `/config/velora.db` | SQLite path or a complete Postgres connection URL |

Only override the container paths when your deployment platform requires different mount targets.

## Database

### SQLite

SQLite is the default and recommended database for a single-host installation. Its database file lives at
`/config/velora.db`, so preserving the `/config` volume preserves the database.

### External Postgres

To use an existing Postgres server, add:

```yaml
environment:
  VELORA_DATABASE_DRIVER: postgres
  VELORA_DATABASE_URL: postgres://user:password@postgres-host:5432/velora?sslmode=disable
```

Velora does not require Postgres for normal installation. The bundled Postgres profile in `compose.dev.yml` exists
only for contributor testing.

## Upgrades

Compose:

```bash
docker compose pull
docker compose up -d
```

Velora applies embedded Goose migrations at startup. Back up `/config` before upgrading between releases.

## Backup And Recovery

Back up `/config`; it contains durable application state. `/cache` is safe to recreate. `/media` remains user-owned
and is never part of a Velora backup.

The exact backup command depends on whether `/config` is a named volume, bind mount, or storage managed by Unraid,
TrueNAS, Synology, or another platform.

## Contributor Configuration

Repository development intentionally uses a different setup. `./velora` targets `compose.dev.yml`, supports optional
`.env` overrides, mounts source-built web assets, and can start a local Postgres service. See
[`CONTRIBUTING.md`](../CONTRIBUTING.md); do not copy those development conventions into a public deployment.
