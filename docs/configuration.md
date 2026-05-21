# Configuration

Everything you'd want to tweak about a running Velora instance: where it reads media from, what database it uses, what
port it listens on.

## Container paths

Velora reads and writes three mount points inside the container:

| Container path | Default host source | Contents                                                  |
|----------------|---------------------|-----------------------------------------------------------|
| `/config`      | `./dev/config`      | Durable state. SQLite db lives here at `/config/velora.db` |
| `/cache`       | `./dev/cache`       | Derived artifacts. Rebuildable; safe to delete            |
| `/media`       | `./dev/media`       | Source media. Read-only contract; Velora never modifies it |

**In code and in API payloads, always use the container paths.** When you POST a library with `"path":"/media/movies"`,
Velora resolves that against the container's `/media`, which compose has bind-mounted from `VELORA_HOST_MEDIA_DIR`.

This indirection is the whole reason the same image runs on your laptop and on Unraid: only the host-side bind sources
change. Server code, schemas, and API payloads stay identical.

### Common mount layouts

**Local dev (default)** — everything in `./dev/`:

```env
VELORA_HOST_CONFIG_DIR=./dev/config
VELORA_HOST_CACHE_DIR=./dev/cache
VELORA_HOST_MEDIA_DIR=./dev/media
```

**Real machine with media on a NAS** — put `/config` and `/cache` on fast local storage, point `/media` at the share:

```env
VELORA_HOST_CONFIG_DIR=/var/lib/velora/config
VELORA_HOST_CACHE_DIR=/var/lib/velora/cache
VELORA_HOST_MEDIA_DIR=/mnt/nas/media
```

**Unraid-style** — typical paths:

```env
VELORA_HOST_CONFIG_DIR=/mnt/user/appdata/velora
VELORA_HOST_CACHE_DIR=/mnt/cache/velora
VELORA_HOST_MEDIA_DIR=/mnt/user/media
```

## Environment variables

### Container side

These configure what the running server sees. Defaults work for the bundled `./dev/` layout.

| Variable                 | Default             | Purpose                                              |
|--------------------------|---------------------|------------------------------------------------------|
| `PORT`                   | `8080`              | HTTP listen port inside the container                |
| `WEB_DIST_DIR`           | `/app/web`          | Where the Go server reads built web assets from      |
| `VELORA_CONFIG_DIR`      | `/config`           | Override the durable-state mount point               |
| `VELORA_CACHE_DIR`       | `/cache`            | Override the cache mount point                       |
| `VELORA_MEDIA_DIR`       | `/media`            | Override the source-media mount point                |
| `VELORA_DATABASE_DRIVER` | `sqlite`            | `sqlite` or `postgres`                               |
| `VELORA_DATABASE_URL`    | `/config/velora.db` | SQLite file path or full Postgres DSN                |

### Host side

These control where compose mounts things from on your machine and which port is exposed:

| Variable                  | Default        | Purpose                                            |
|---------------------------|----------------|----------------------------------------------------|
| `VELORA_HOST_CONFIG_DIR`  | `./dev/config` | Where `/config` is bind-mounted from               |
| `VELORA_HOST_CACHE_DIR`   | `./dev/cache`  | Where `/cache` is bind-mounted from                |
| `VELORA_HOST_MEDIA_DIR`   | `./dev/media`  | Where `/media` is bind-mounted from                |
| `VELORA_HTTP_PORT`        | `8080`         | Port published on the host                         |

### Postgres credentials (only when using the bundled profile)

| Variable             | Default  | Purpose                                  |
|----------------------|----------|------------------------------------------|
| `POSTGRES_DB`        | `velora` | Database name                            |
| `POSTGRES_USER`      | `velora` | Username                                 |
| `POSTGRES_PASSWORD`  | `velora` | Password                                 |
| `POSTGRES_PORT`      | `5432`   | Port published from the postgres service |

## Database

### SQLite (default)

Nothing to configure. The database file is at `/config/velora.db` inside the container, which by default is
`./dev/config/velora.db` on the host. Backup is `cp`:

```bash
cp dev/config/velora.db dev/config/velora.db.bak
```

SQLite is the right choice for a single-host install. Faster than Postgres for media-server workloads, no separate
process to babysit, and the file format is portable.

### Bundled Postgres

For a local Postgres test, the compose file ships a `postgres:18-alpine` service behind a profile:

```bash
./velora create --postgres
```

The wrapper sets `VELORA_DATABASE_DRIVER=postgres` and builds the DSN from the `.env` credentials. Data lives in a named
docker volume (`postgres-18-data`), separate from `dev/`.

### External Postgres

To point Velora at an existing Postgres (separate container, managed service, etc.), skip the `--postgres` flag and set
in `.env`:

```env
VELORA_DATABASE_DRIVER=postgres
VELORA_DATABASE_URL=postgres://user:password@host:5432/velora?sslmode=disable
```

Then plain `./velora create` uses your external database. The bundled `postgres` service stays disabled.

## Migrations

Goose migrations are embedded in the binary (per dialect — see `apps/server/internal/migrations/sql/`) and applied at
container startup. There's no separate "migrate" step.

If you need to inspect state, the `goose_db_version` table tracks applied versions.

## Backup and recovery

Two paths matter:

- `/config` — back this up. Contains the database and any future runtime state.
- `/cache` — safe to lose. Velora regenerates derived artifacts on demand.

`/media` is your source data; Velora only reads it.

To restore from a `/config` backup, drop the files in place and `./velora create`. Migrations re-apply if needed and
pick up where they left off — the `goose_db_version` table is part of the backup.

## Changing config after the stack is running

Compose re-reads `.env` only when (re)creating a service. After editing:

```bash
./velora destroy
./velora create
```

Or force a recreate without changing the image:

```bash
docker compose up -d --force-recreate
```
