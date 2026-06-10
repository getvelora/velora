# Contributing to Velora

Thanks for your interest in Velora. This document covers the practical bits — how to get the project running locally, what we expect from a patch, and where to ask questions.

## Where to ask questions

- **General questions, ideas, "would you accept X":** [GitHub Discussions](https://github.com/getvelora/velora/discussions)
- **Bug reports and concrete feature requests:** [GitHub Issues](https://github.com/getvelora/velora/issues)
- **Security vulnerabilities:** see [SECURITY.md](SECURITY.md) — please do not file public issues for these.

## Project layout

Velora is a single container that bundles a Go HTTP server and a React/Vite web client. The same Go binary serves both `/api/*` and the built web assets.

- `apps/server` — Go server (`cmd/velora` is the entrypoint, internal packages under `internal/`)
- `apps/web` — React/Vite client
- `dev/` — local runtime folders mounted into the container (`dev/config`, `dev/cache`, `dev/media`)
- `docs/` — user-facing docs (setup, configuration, API, troubleshooting)
- `AGENTS.md` — the canonical engineering guide. Read it before writing code.

## Getting the project running

The repo includes a contributor-only `./velora` helper around `compose.dev.yml`. This stack is deliberately different
from the public `compose.yml`, which pulls the released GHCR image. Development builds the `server-runtime` Docker
target containing only Go and FFmpeg; web assets are built separately and bind-mounted into the running server.

```bash
cp .env.development.example .env # optional local overrides
./velora create            # build the server image and start SQLite
./velora create --postgres # build and start with local Postgres
./velora logs              # follow container logs
./velora status            # show container status
./velora destroy           # stop containers, keep volumes
./velora clean             # stop containers and remove compose volumes
./velora web-build         # rebuild mounted web assets without rebuilding the app image
./velora lint              # run the pinned Go linter
```

The development stack publishes on `http://localhost:8080`. The web dev server (`apps/web`) runs separately on `:5173`
and proxies `/api` to `:8080`.

Enable the repository's Git hooks once after cloning:

```bash
git config core.hooksPath .githooks
```

## Running the tests

### Server (Go)

The repo carries checked-in build/mod caches under `apps/server/.cache` so CI and local runs don't rebuild from scratch. Point both vars at them:

```bash
cd apps/server
GOCACHE=$PWD/.cache/go-build GOMODCACHE=$PWD/.cache/go-mod go test ./...
```

To run a single test:

```bash
GOCACHE=$PWD/.cache/go-build GOMODCACHE=$PWD/.cache/go-mod \
  go test ./internal/health -run TestHandlerReturnsOKWhenDatabaseCheckPasses
```

### Web (Vite + TypeScript)

```bash
cd apps/web
npm run build   # tsc typecheck + vite build
npm run dev     # vite dev server, proxies /api to :8080
```

When developing through the bundled server instead of Vite, run `./velora web-build`. It reuses containerized npm
dependencies, rebuilds the assets mounted at `/app/web`, and requires only a browser refresh. It does not rebuild or
restart the Go server. `./velora create` builds web assets only when `apps/web/dist` does not exist.

## Branching

The active branch is `develop`. `main` only exists at release time. Open PRs against `develop`. See [AGENTS.md](AGENTS.md) for the full rationale.

## Coding style

The full style guide lives in [AGENTS.md](AGENTS.md). The short version:

- **Go:** `gofmt`, small focused packages, handlers take dependencies as injected values (see `health.NewHandler` for the pattern). Tests live beside the package under test and are named by behavior (`TestConfigDefaultsToSQLiteInConfigDirectory`).
- **TypeScript/React:** functional components, explicit types for API responses, clear names over abbreviations.
- **Migrations:** Goose, must work on both SQLite and Postgres.

## Commits and pull requests

Use one-line Conventional Commits:

```text
feat: add library registration endpoint
fix: preserve local config on shutdown
docs: update local setup instructions
```

A good PR:

- Targets `develop`.
- Has a summary that explains the user-visible change.
- Lists the verification commands you ran (server tests, `npm run build`, manual checks).
- Notes any config or data-persistence impact.
- Includes screenshots or short clips for UI changes.

The committed pre-commit hook runs `./velora lint`. CI also runs the Go build/vet/test matrix, lint, web build, and
Docker smoke test on every PR. Hooks can be bypassed locally, so PRs still need a green CI before merge.

### Dependabot updates

Dependabot checks Go modules, npm packages, GitHub Actions, and Docker images weekly.

- Patch updates are grouped by ecosystem. After all required CI and CodeQL checks pass, GitHub squash-merges the PR,
  closes it, and deletes its branch automatically.
- Minor and major updates are not auto-merged. Their PRs remain open until a maintainer reviews and merges them,
  closes them, or tells Dependabot to ignore that update.
- A failed patch update remains open so its failure can be investigated; automation does not merge or close it.
- Closing a Dependabot PR does not necessarily suppress future versions. Use the relevant `@dependabot ignore`
  command in the PR when an update or version line should remain suppressed.

When multiple dependency PRs modify the same manifest or lockfile, handle them sequentially instead of resolving every
branch against the same stale base:

1. Merge the update that subsumes or is required by another update first.
2. Close any PR made redundant by that merge.
3. Refresh the next distinct PR against the updated `develop` branch and regenerate its lockfile.
4. Confirm the refreshed diff contains only the intended dependency changes and that all required checks pass.

This avoids repeatedly resolving conflicts that will be recreated as soon as another overlapping dependency PR merges.

## License

By contributing, you agree that your contributions will be licensed under the [GNU AGPL v3.0](LICENSE).
