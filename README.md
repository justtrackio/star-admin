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

The backend listens on port **8082**. Vite serves the frontend on port **5173** and proxies `/api` to the backend. The dashboard calls `GET /api/health` through TanStack Query and displays application availability. This endpoint does not check StarRocks connectivity.

The **Cluster status** dashboard at `/` shows live health cards for nodes, Kafka routines, dictionaries, and materialized views. Green requires every reported resource to pass its check. Nodes must be online without decommissioning or errors; routines must be running or awaiting scheduling; dictionaries must be finished, refreshing, or committing without errors; materialized views must be active with a known non-failed refresh state and no errors. Loading and empty optional resources are neutral; an empty node list is red. Failed updates turn the indicator red while retaining the previous counts and offering Retry. Each card links to its resource list. These metadata checks do not measure data freshness or detect nodes removed from cluster registration. Dashboard backend availability is shown separately.

The **Nodes** page at `/nodes` lists frontend, backend, and compute nodes in separate tables with basic status and resource details. Click a row or node link to open `/nodes/<type>/<id>` for heartbeat, version, errors, ports, and all metadata reported for that node. Rows do not expand. Both pages use `GET /api/nodes` SSE snapshots every five seconds, with automatic reconnection and stale-data warnings. Node type and ID identify nodes together. Missing metadata displays a dash, and removed nodes show a not-found page. Reads use `SHOW FRONTENDS`, `SHOW BACKENDS`, and `SHOW COMPUTE NODES`; the account needs SYSTEM OPERATE or `cluster_admin`. Frontend JVM heap usage comes from `information_schema.fe_metrics`, joined by the frontend name. This measures heap usage against the JVM heap maximum, excluding off-heap memory. Frontend CPU measurements are not available from the cluster SQL metadata. Frontend detail pages also list every setting and returned metadata column from `ADMIN SHOW FRONTEND CONFIG`, with search, pagination, and five-second SSE updates from `GET /api/frontend-config`. The source notice explains that these values belong to the FE serving the shared SQL connection, which can differ from the selected node. Credential values are masked before transmission. The existing JSON endpoint `/api/overview` remains available for compute nodes. No node operations are performed.

The **Materialized views** page at `/materialized-views` lists views across databases in the configured cluster's internal catalog. The SSE snapshot reads `information_schema.materialized_views` through the shared connection pool. The page supports name/database search, database filtering, sorting and pagination. The card header shows when the table last received a successful data update. The list loads snapshots over `GET /api/materialized-views` using SSE; the detail page uses `GET /api/materialized-views/:database/:id`, which filters metadata by database and view ID and returns only that view. The refresh tasks table shows reported partition counts and names from `EXTRA_MESSAGE`, plus queue and processing durations. Pending queue time and running processing time update with each snapshot; unavailable timings display a dash. Partition counts reflect retained metadata, which StarRocks may truncate. The refresh tasks tab streams the latest 100 retained runs over `GET /api/materialized-views/:database/:id/tasks`, with an immediate snapshot and updates every five seconds while connected. Error retries reconnect the task stream; switching tabs disconnects it. A missing view produces an empty snapshot so the detail page can show its not-found state. Each connected client receives an immediate snapshot followed by a metadata read every five seconds. Disconnecting stops that client’s refresh loop and cancels any in-flight query. Transient query failures keep the last displayed data and retry at the next interval; interrupted connections reconnect automatically. The frontend stores snapshots in TanStack Query, and error retries reconnect the stream to request an immediate snapshot. HTTP compression is disabled to avoid buffering SSE updates. Click a view name to open its detail page at `/materialized-views/<database>/<id>`, showing its overview, refresh tasks and SQL definition tabs, and any activation or refresh errors. Detail links work on direct navigation and show a not-found message when a view is absent. The last successful refresh timestamp uses the latest completed SUCCESS job in `information_schema.materialized_view_refresh_jobs`, or the latest attempt timestamp when that attempt succeeded. Pending and skipped attempts do not overwrite it. If successful jobs have expired from retained history and the latest attempt is not successful, this timestamp is unavailable. Row counts are approximate metadata statistics; unavailable metadata displays a dash.

The **Tables** page at `/tables` lists regular tables across visible databases. Table metadata comes from `information_schema.tables`, with materialized views and system databases excluded. Row counts are estimates; data size includes all replicas.

The **Kafka routines** page at `/routine-loads` lists Kafka Routine Load jobs across visible databases. Ingestion metadata comes from `information_schema.routine_load_jobs` and shows target tables, Kafka topics, states, job row totals, error rows, reported rows per second, and state change reasons. Click a routine name or row to open `/routine-loads/<database>/<id>` with live job state, ingestion totals, reported throughput, target and topic details, and consumed partition offsets. Detail links support direct navigation and show a not-found message when the job is absent. Consumed offsets do not measure consumer lag because Kafka end offsets are unavailable in this metadata. Both lists support search, pagination, and SSE snapshots every five seconds while connected. Independent SSE endpoints (`GET /api/tables` and `GET /api/routine-loads`) send an immediate snapshot, retain the last successful data on errors, and retry automatically. Disconnecting cancels in-flight metadata reads. The cluster must support the routine load metadata view, and the account must have permission to see the jobs. External connectors that use Stream Load are not represented by Routine Load metadata. The API selects only the Kafka topic from source properties and does not return custom Kafka properties or credentials. The list endpoints read metadata only. The routine details page also provides Pause and Resume controls through `POST /api/routine-loads/:database/:id/:action` (`pause`, `resume`, `resume-earliest`, or `resume-latest`). The backend resolves the current job by database and ID, checks its state, and executes the corresponding Routine Load command. Resume is available for paused jobs; Pause is available for running, scheduled, or unstable jobs. Cancelled and stopped jobs cannot be resumed. Commands require the cluster account to have INSERT permission on the target table. Failed commands remain visible, and accepted commands reload the live snapshot. When a paused routine reports a Kafka offset out of range error, its error section offers recovery from earliest retained or latest offsets. After confirmation, recovery resets all partitions reported in the job progress, then resumes the routine. Earliest can replay loaded data; latest skips the retained backlog. Deleted Kafka messages cannot be recovered. The backend rejects recovery for other errors or missing/invalid partition metadata, stops if the offset reset fails, and reports when a reset succeeds but resuming fails. The details page shows `max_routine_load_batch_size` (bytes) and `routine_load_task_consume_second` (seconds) from the connected frontend through read-only `ADMIN SHOW FRONTEND CONFIG` metadata at `GET /api/routine-load-settings`, updated every five seconds. The routine’s `task_consume_second` is shown separately from job properties. Configuration permissions may be required; failures do not block routine or task snapshots. The details page also streams current execution tasks from `SHOW ROUTINE LOAD TASK` through `GET /api/routine-loads/:database/:id/tasks` every second. Executions can be brief, so snapshots may show new queued tasks with pending transactions and node assignments while ingestion continues; snapshots can still miss short executions. Tasks show transaction status and IDs, execution node, warehouse, creation/scheduling/execution times, timeout, messages, and expandable partition progress/latest offsets. These are current tasks, not historical runs; paused jobs may have none. Task errors retain the last successful snapshot. IDs and offsets retain their original precision, and absent metadata displays a dash.

The **Resource groups** page at `/resource-groups` lists all groups in the connected cluster through read-only `SHOW RESOURCE GROUPS ALL`. The SSE endpoint `GET /api/resource-groups` sends an immediate snapshot and updates every five seconds. Search by name, ID, or classifier, and expand a row for all reported configuration fields and classifier rules. Groups with multiple classifiers appear once. Limits are configuration per execution node, not live usage. Missing values display a dash. Failed updates keep the last successful snapshot and show a warning with a retry control. No resource groups are modified.

The **Dictionaries** page at `/dictionaries` lists dictionary objects and their reported refresh status through read-only `SHOW DICTIONARY` metadata at `GET /api/dictionaries`. It streams an immediate SSE snapshot and updates every five seconds while open, with automatic reconnection, name/database/source search, and status filtering. Expand rows for source key/value columns, refresh start time, and per-node memory information. Refresh errors remain visible even when StarRocks reports `FINISHED` after rolling back to cached data. Failed updates retain the previous snapshot with a stale-data warning. Snapshot updates never trigger a dictionary refresh or change cluster data. The connected account must be permitted to run `SHOW DICTIONARY`.

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
