# StarRocks Admin

Backend and frontend foundation for a StarRocks cluster administration application, using current stable releases.

## Stack

| Part | Technologies |
| --- | --- |
| Backend | Go 1.27.2, gosoline v0.66.2, httpserver v0.6.4, gin-contrib/cors v1.7.9, gosoline-project/sqlc v0.4.0 |
| Frontend | React 19.3.0, TypeScript 7.0.2, Vite 8.3.4, TanStack Router 1.170.42 and Query 5.104.1, Ant Design 6.6.5 |
| Tooling | Bun 1.4.3, mise, Oxlint 1.87.0, gofumpt 0.12.0, golangci-lint 2.14.0 with justtrackio/go-linters v0.3.0, mockery 3.8.0 |
| Distribution | React assets embedded in a static Go binary; distroless Docker image |

Versions were checked against official package registries and release APIs on 2026-10-11. Oxlint replaces ESLint and supports TypeScript 7, including type-aware linting through `oxlint-tsgolint`. Go transitive dependencies follow the upgraded frameworks' dependency graph.

The Vite integration uses the maintained `@tanstack/router-plugin` package with automatic route code splitting. Ant Design components use the v6 APIs. CI actions and the Go/Bun Docker image tags match the upgraded tooling. Add cluster clients as features require them.

## Development

Install the pinned tools and dependencies:

```sh
mise install
cd frontend
bun install --frozen-lockfile
cd ../backend
go mod download
```

Run these in separate terminals from the project root:

```sh
mise run dev:backend
mise run dev:frontend
```

Alternatively, run `go run . -c config.sandbox.yml` from `backend/` and `bun run dev` from `frontend/`.

The backend listens on port **8082**. Vite serves the frontend on port **5173** and proxies `/api` to the backend. The dashboard calls `GET /api/health` through TanStack Query and displays application availability. This endpoint does not check StarRocks connectivity. No cluster operations are implemented yet.

`frontend/.env.example` documents the optional `VITE_API_BASE_URL` override. Backend defaults are embedded from `backend/config.dist.yml`; `-c` loads a YAML override, and gosoline supports environment overrides.

## StarRocks connections

The backend uses `github.com/gosoline-project/sqlc` v0.4.0 with its MySQL driver. A single `sqlc.default` connection is configured in `backend/config.dist.yml`. Supply deployment-specific hosts and credentials through Git-ignored local config files or gosoline environment variables. Example connection settings:

| Environment | Example host | Port | User | Password |
| --- | --- | --- | --- | --- |
| Development | `starrocks.dev.example.com` | 9030 | `star_admin` | configured externally |
| Production | `starrocks.prod.example.com` | 9030 | `star_admin` | configured externally |

The committed default host is `127.0.0.1`. Create a local `config.prod.yml` with the connection settings for your deployment, then load it with:

```sh
cd backend
go run . -c config.prod.yml
```

For local development, create `config.sandbox.yml` with your development connection settings. Production binaries can also receive `-c backend/config.prod.yml` from the project root. Select a database account and authentication settings appropriate for your cluster.

Startup opens and pings the selected cluster, failing with an error if it cannot connect. The kernel's service module closes the shared pool after HTTP/application modules stop. Future handlers and services obtain the same `sqlc.Client` through `internal.ProvideStarRocksClient`. The pool allows 10 open and 2 idle connections, with a 3-second dial timeout and 10-second read/write timeouts. No default database is selected, so future queries should qualify tables or configure `sqlc.default.uri.database`.

Migrations, multi-statements, and resource lifecycle operations are disabled. The connection setup performs no schema changes. `/api/health` continues to report application availability rather than probing the database on each request.

## Validation

```sh
mise run ci
```

This runs backend lint, Go tests, frontend lint, and the frontend production/type build. Backend lint builds a custom golangci-lint; the first run downloads its plugin. Backend tests verify cluster configuration and connection-pool shutdown.

Frontend lint settings are in `frontend/.oxlintrc.json`. They enable Oxlint's correctness and suspicious checks across its core JavaScript, TypeScript, Unicorn, Oxc, React, accessibility, and import rules. React Hooks, Fast Refresh, explicit `any`, unhandled promises, and misused promises are checked explicitly. Type-aware linting is enabled; generated routes and build output are ignored. Fast Refresh recognizes TanStack's route constructors. The configuration permits CSS side-effect imports and uses React's automatic JSX runtime. `bun run lint` fails on warnings as well as errors, and `bun run lint:fix` applies safe fixes. Type checking remains part of `bun run build` via TypeScript 7. See [Oxlint configuration](https://oxc.rs/docs/guide/usage/linter/config) and [type-aware linting](https://oxc.rs/docs/guide/usage/linter/type-aware).

## Production

```sh
mise run build
./backend/star-admin
```

The build copies `frontend/dist` into `backend/public` before compiling, so the binary serves both the frontend and `/api`. A tracked placeholder page allows the backend to compile before the first frontend build. After changing routes or frontend code, rebuild to update embedded assets.

```sh
docker build -f backend/build/Dockerfile -t star-admin .
docker run --rm -p 8082:8082 star-admin
```

GitHub Actions provide CI plus amd64/arm64 image builds and GHCR publishing. Pull requests build images without publishing. `backend/build/Dockerfile.release` packages prebuilt binaries from `dist/<arch>/star-admin`.

## Layout

```text
backend/
  main.go                 # gosoline lifecycle and HTTP registration
  internal/               # HTTP handlers and future cluster services
  config.dist.yml         # embedded defaults
  config.sandbox.yml      # development overrides
  public/                 # embedded frontend assets
  build/                  # Dockerfiles
frontend/
  src/api/                # typed HTTP client and response schemas
  src/hooks/              # TanStack Query hooks
  src/routes/             # TanStack file-based routes
  src/main.tsx            # router and query providers
mise.toml                 # pinned tools and dev/build/CI tasks
.github/workflows/        # CI and multi-architecture image publishing
```
