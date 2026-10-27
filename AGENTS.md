# StarRocks Admin developer guide

Go backend with gosoline and httpserver, React/TypeScript frontend with TanStack Router, TanStack Query, and Ant Design. Read README.md for setup and architecture.

## Commands

- `mise run dev:backend` / `mise run dev:frontend`: development servers.
- `mise run build`: frontend build, embed assets, backend binary.
- `mise run ci`: custom backend lint, Go tests, frontend lint and build.
- Backend: `go test ./...`, `go vet ./...`, `go build -o star-admin .`.
- Frontend: `bun install --frozen-lockfile`, `bun run lint`, `bun run lint:fix`, `bun run build`.
- Frontend lint uses Oxlint with type-aware checks; configure `frontend/.oxlintrc.json`. Warnings fail CI. TypeScript 7 checks types during the build.

## Development verification

- At the start of each session, check for this project's backend on port 8082 and frontend dev server on port 5173. Reuse running processes across sessions.
- Start only missing servers. Do not create duplicate processes or use alternate ports.
- Start the backend with the production environment configuration: run `go run . -c config.prod.yml` from `backend/`.
- Start the frontend dev server with `mise run dev:frontend` from the project root.
- In each new session, open the dashboard at `http://localhost:5173/` after both servers are ready, including when reusing existing processes.
- Use both running servers to verify changes.
- **After backend code or configuration changes, restart the backend before reporting completion.** The running process does not reload these changes automatically. This restart is authorized; do not ask for confirmation again.
- Stop only this project's backend on port 8082. Wait for it to exit, then run `go run . -c config.prod.yml` from `backend/`. Verify the changed behavior through the running frontend after the backend is ready.
- Keep the frontend dev server on port 5173 running. Do not stop other processes or stop either server as session cleanup.
- Leave both servers running when the session ends so other sessions can reuse them.

## Conventions

- Go: snake_case files, contextual error wrapping, early returns, gosoline providers for shared services, constructor injection and httpserver typed handlers.
- Frontend: function components, PascalCase component files, camelCase hooks/utilities, type-only imports, strict TypeScript; avoid `any`.
- Use TanStack Query for server state and TanStack Router file routes in `frontend/src/routes`.
- `routeTree.gen.ts` is generated during Vite builds and ignored by Git.
- Production assets live in `backend/public` and are embedded via `//go:embed`; build the frontend before the production binary.
- Health means application availability; do not represent it as StarRocks cluster health.
- StarRocks uses one `github.com/gosoline-project/sqlc` connection named `default`. Defaults point to localhost; use local config files or environment variables for deployment-specific settings.
- Use `ProvideStarRocksClient` to share the pool. The kernel module connects at startup and closes it after application modules stop.
- Migrations and resource lifecycle operations are disabled. Use read-only queries by default. Supply cluster hosts and credentials through local config files or environment variables; do not commit secrets to source control.
