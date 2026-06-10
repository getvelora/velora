# Development

This guide covers repository architecture, local development, testing, and continuous integration. Public
installations should use the published container and follow [Getting Started](getting-started.md).

## Project Layout

```text
apps/
  server/                   Go API and media server
    cmd/velora/             Application entrypoint
    internal/               Database, libraries, media, migrations, storage, health, and web packages
  web/                      React, Vite, and TypeScript client
dev/                        Local config, cache, and sample media
.github/workflows/          Continuous integration and release workflows
compose.yml                 Public deployment example using the published image
compose.dev.yml             Contributor development stack
Dockerfile                  Release image and server-only development target
velora                      Contributor helper around compose.dev.yml
```

The release image contains the Go server, built web client, and FFmpeg. The Go binary serves both `/api/*` routes and
the single-page web application.

Local development separates the builds: `compose.dev.yml` runs the Dockerfile's `server-runtime` target with Go and
FFmpeg, while web assets are built separately and mounted from `apps/web/dist`. The `./velora` helper always targets
this contributor stack; it does not operate the public `compose.yml`.

## Start The Development Stack

```bash
cp .env.development.example .env # optional local overrides
./velora create                  # SQLite
./velora create --postgres       # local Postgres profile
./velora status
./velora logs
```

The stack publishes Velora at <http://localhost:8080>.

Use `./velora destroy` to stop containers while preserving volumes, or `./velora clean` to remove the development
volumes as well.

## Server

The repository carries Go build and module caches under `apps/server/.cache`. Point both cache variables at them:

```bash
cd apps/server
GOCACHE=$PWD/.cache/go-build GOMODCACHE=$PWD/.cache/go-mod go test -race ./...
```

Run one package or test with the same cache configuration:

```bash
GOCACHE=$PWD/.cache/go-build GOMODCACHE=$PWD/.cache/go-mod \
  go test ./internal/health -run TestHandlerReturnsOKWhenDatabaseCheckPasses
```

Run the pinned Go linter from the repository root:

```bash
./velora lint
```

## Web Client

For live reload, keep the development stack running and start Vite separately:

```bash
cd apps/web
npm run dev
```

Vite runs on <http://localhost:5173> and proxies `/api` to the container on port `8080`.

To build the client directly:

```bash
cd apps/web
npm run build
```

To build production-shaped assets without installing Node locally:

```bash
./velora web-build
```

The helper builds assets in a disposable Node container and writes `apps/web/dist`. That directory is mounted into the
running Velora container, so a browser refresh serves the new build without rebuilding or restarting the server.
`./velora create` builds the assets only when `dist` is missing.

## Git Hooks And CI

Enable the repository's committed pre-commit hook once after cloning:

```bash
git config core.hooksPath .githooks
```

The hook runs `./velora lint`.

GitHub Actions checks server formatting, builds, vetting, race-enabled tests, pinned Go linting, the web build, and a
Docker smoke test on pushes and pull requests to `develop`.
