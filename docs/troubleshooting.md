# Troubleshooting

These instructions assume the public `compose.yml` and the published GHCR image. Contributors using `./velora`
should see the final section.

## The Container Will Not Start

Check status and logs:

```bash
docker compose ps
docker compose logs velora
```

If port `8080` is already in use, change the host side of the port mapping:

```yaml
ports:
  - "8081:8080"
```

Then apply the change:

```bash
docker compose up -d
```

If Docker cannot connect to its daemon, start Docker Desktop or ensure your Linux user has Docker access.

## Pulling The Image Fails

Confirm the image name is:

```text
ghcr.io/getvelora/velora:latest
```

Released images are public and do not require registry authentication. Check the release/tag exists, then retry:

```bash
docker compose pull
docker compose up -d
```

## `/api/health` Returns 503

Inspect logs first:

```bash
docker compose logs velora
```

With default SQLite, verify that `/config` is writable:

```bash
docker compose exec velora sh -c 'touch /config/.write-test && rm /config/.write-test'
```

With external Postgres, confirm `VELORA_DATABASE_URL` is reachable from inside the container. The hostname must resolve
on the container network, not only on the host.

## Media Is Not Visible

Check the mounted tree:

```bash
docker compose exec velora ls -la /media
```

The host mount and API library path are different namespaces. For this mount:

```yaml
- /mnt/storage/videos:/media:ro
```

register a movie directory as `/media/movies`, not `/mnt/storage/videos/movies`.

Also confirm the source exists on the host and that Docker has permission to read it. On Docker Desktop, the host path
may need to be allowed in file-sharing settings.

## A Migration Failed

Velora applies embedded migrations at startup. Preserve `/config`, capture the logs, and do not repeatedly delete or
recreate data while investigating:

```bash
docker compose logs velora
```

For a disposable pre-release installation, a complete reset is:

```bash
docker compose down --volumes
docker compose up -d
```

This permanently deletes the SQLite database and all Velora configuration.

## Upgrade Did Not Change The Version

Pull before recreating:

```bash
docker compose pull
docker compose up -d
```

If the service still uses an old image, inspect it:

```bash
docker compose images
```

## Inspecting The Container

```bash
docker compose exec velora sh
docker compose exec velora wget -qO- http://localhost:8080/api/health
```

## Contributor Development

`./velora` is only for a repository checkout and always targets `compose.dev.yml`:

```bash
./velora status
./velora logs
./velora destroy
./velora create
```

Development `.env`, source mounts, `web-build`, and the local Postgres profile do not apply to public installations.
