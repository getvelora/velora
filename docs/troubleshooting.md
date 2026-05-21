# Troubleshooting

Common issues and how to diagnose them. If you hit something not covered, capture the logs (`./velora logs`) and open
an issue.

## The stack won't start

### Port conflict

`./velora create` errors with something like *"bind: address already in use"*. Another process is on 8080.

```bash
# See what's bound:
lsof -i :8080

# Either stop that process, or pick a different port in .env:
echo 'VELORA_HTTP_PORT=8081' >> .env
./velora destroy && ./velora create
```

### Docker not running / no socket access

`docker compose` says it can't connect to the daemon. Start Docker Desktop, or on Linux ensure your user is in the
`docker` group:

```bash
sudo usermod -aG docker $USER
# log out and back in
```

### Build fails on Go or Node version

The Dockerfile pins specific versions. If those tags no longer exist on Docker Hub (rare but possible during transition
periods), update the `FROM` lines in `Dockerfile` to a known-good tag. The Go version in `apps/server/go.mod` is the
authoritative target.

## `/api/health` returns 503

Means the database ping is failing.

### With SQLite

Almost always a file-permission or mount issue. Check:

```bash
ls -ld dev/config
docker compose exec velora ls -la /config
```

The container needs to be able to read and write `/config`. If you've bind-mounted a read-only volume or a directory
with restrictive permissions, that'll fail. Velora runs as root inside the container by default, so this mostly bites
on SELinux-enabled Linux hosts or when mounting NFS with `noexec`/`nosuid`.

### With bundled Postgres

The Postgres container hasn't finished booting, or the wait loop timed out:

```bash
./velora logs                                          # check both services
docker compose --profile postgres exec postgres pg_isready -U velora -d velora
```

If `pg_isready` reports `accepting connections` but Velora still 503s, check the DSN that the wrapper built:

```bash
docker compose exec velora env | grep VELORA_DATABASE
```

### With external Postgres

Most common: bad DSN. The format is:

```
postgres://user:password@host:port/dbname?sslmode=disable
```

Note `sslmode=disable` is the easiest starting point but you almost certainly want `sslmode=require` against a remote
database. Network reachability also matters — `host` must be resolvable from inside the Velora container, not just from
the host. Use `host.docker.internal` (Docker Desktop) or a compose network alias if Postgres runs on the same machine.

## My media isn't visible

Almost always container-vs-host path confusion.

Symptoms:
- You created a library with path `./dev/media/movies` (the host path). API accepted it because the path string is just
  text — but nothing under `/media` matches, so scans will find nothing.
- You set `VELORA_HOST_MEDIA_DIR` to a new location but library paths still reference the old one.

Diagnose by checking what `/media` looks like from inside the container:

```bash
docker compose exec velora ls /media
docker compose exec velora ls /media/movies
```

If the directory tree isn't what you expect, fix the bind mount in `.env` and recreate the container. If it's right but
your library path is wrong, delete and recreate the library with the correct container path. (Deletion isn't an API yet
— for now, edit the SQLite db directly or `./velora clean` if you don't mind starting fresh.)

## A migration failed at startup

Velora applies Goose migrations on every boot. If one fails, the container exits with a fatal log and the database is
left in whatever state the failed migration reached.

```bash
./velora logs | grep -iE 'migration|goose'
```

For SQLite during pre-release, the simplest recovery is to nuke and restart:

```bash
./velora destroy
rm dev/config/velora.db
./velora create
```

For Postgres, connect with `psql` and inspect `goose_db_version`. You may need to manually delete the failed row before
the next boot will retry.

## Changed `.env` and nothing happened

Compose re-reads `.env` only when bringing a service up. After editing:

```bash
./velora destroy
./velora create
```

Or force a recreate without rebuilding the image:

```bash
docker compose up -d --force-recreate
```

## Resetting everything

When you want a totally clean state (testing migrations, starting over):

```bash
./velora clean              # stops containers + drops compose-managed volumes
rm -rf dev/cache/* dev/config/*
./velora create
```

`dev/media/` is your source files — left alone.

## Inspecting the running container

```bash
# Shell into it:
docker compose exec velora sh

# Watch live logs:
./velora logs

# Hit endpoints from inside (verifies port binding vs in-container listening):
docker compose exec velora wget -qO- http://localhost:8080/api/health
```

## Where to find the version

The build doesn't currently stamp a version. To identify what's deployed, use the git SHA of the commit you built from:

```bash
git rev-parse --short HEAD
```

A versioned `/api/version` endpoint is on the roadmap.
