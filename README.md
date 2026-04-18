# Agrafa

Agrafa is a lightweight observability stack for small deployments, self-hosted apps, and side projects. It combines a Go backend, a React frontend, and a small Go agent that reports heartbeats, host metrics, and health check results.

The backend is the source of truth. The agent reports raw signals, the backend interprets them into current state and events, and the frontend presents that operational view to users.

## Repository Structure

```text
agrafa/
├── frontend/   # React + Vite UI
├── backend/    # Go API, state engine, jobs, database layer
└── agent/      # Go node agent for heartbeats, metrics, and health checks
```

## What Each Part Does

| Folder | Purpose | Main stack |
| --- | --- | --- |
| `frontend` | Dashboard UI for auth, overview, nodes, services, alerts, and settings | React 19, Vite, TypeScript, Tailwind |
| `backend` | API, auth, ingestion, state evaluation, events, alerts, and read models | Go, Chi, PostgreSQL, sqlc |
| `agent` | Runs on a monitored machine and reports data back to the backend | Go, gopsutil |

## How It Fits Together

1. The `agent` sends heartbeats, metrics, and health check results to the `backend`.
2. The `backend` stores observations, evaluates node and service state, and records meaningful events.
3. The `frontend` reads that backend state and exposes it through the UI.

## Prerequisites

- Go `1.24.x`
- Node.js and `pnpm` for the frontend
- PostgreSQL for the backend
- `psql` for running migrations and seeds via the backend `Makefile`

## Local Development

### 1. Start the backend

The backend needs PostgreSQL before it can run.

```bash
cd backend
cp .env.example .env
```

Update `POSTGRES_URI` in `backend/.env` to point at your local database, then run:

```bash
make migrate-up
make seed
make run
```

Useful local endpoints:

- API base: `http://localhost:8080/v1`
- Swagger UI: `http://localhost:8080/docs`
- OpenAPI JSON: `http://localhost:8080/openapi/swagger.json`

### 2. Start the agent

The agent talks to the backend and uses a node token for authentication.

```bash
cd agent
cp .env.example .env
make run
```

Important values in `agent/.env`:

- `AGRAFA_API_BASE_URL=http://localhost:8080/v1`
- `AGRAFA_AGENT_TOKEN=...`
- `AGRAFA_NODE_ID=...`

More agent details live in [agent/README.md](agent/README.md).

### 3. Start the frontend

```bash
cd frontend
cp .env.example .env
pnpm install
pnpm dev
```

The frontend uses `VITE_API_URL` from `frontend/.env` and proxies `/v1` requests to the backend during development.

## Self-Hosting MVP

### Zero-Question Install

The simplest self-hosted install is the dedicated `install/` path. It is fully non-interactive and brings Agrafa up on the server IP without domain setup, TLS, or Traefik.

```bash
curl -fsSL https://raw.githubusercontent.com/<owner>/agrafa/main/install/install.sh | bash
```

Result:

- Frontend: `http://server_ip:8080`
- Backend API: `http://server_ip:8081`
- Agent API base URL: `http://server_ip:8081`

The installer generates `.env`, randomizes `POSTGRES_PASSWORD` and `APP_SECRET`, detects the public IP when possible, and starts the stack with `docker compose up -d`.

If you want a domain, TLS, or a single public entrypoint later, place your own reverse proxy in front after installation.

This repo includes two Compose setups for `postgres`, `backend`, and `frontend`:

- `docker-compose.yml` pulls released backend and frontend images from GitHub Container Registry.
- `docker-compose.local.yml` builds backend and frontend locally from this checkout.

1. Copy the root env file:

```bash
cp .env.example .env
```

2. Set the required values in `.env`:

- `POSTGRES_PASSWORD`
- `APP_SECRET`
- `APP_BASE_URL`
- `VITE_API_URL`

Keep the email variables blank unless you want outbound email. Agrafa boots without email configured.

3. Set the published image owner and tags in `.env` when using released images:

- `GHCR_OWNER`
- `AGRAFA_BACKEND_TAG`
- `AGRAFA_FRONTEND_TAG`

4. Start the released stack:

```bash
docker compose up -d
```

5. Or start the local-build stack:

```bash
docker compose -f docker-compose.local.yml up -d --build
```

6. Access the app:

- Frontend: `http://localhost:5173`
- Backend API: `http://localhost:8080/v1`
- Swagger UI: `http://localhost:8080/docs`

Notes:

- The backend connects to PostgreSQL over the Compose network using the `postgres` service hostname from `POSTGRES_URI`.
- The backend runs app migrations on startup using the app's fixed `agrafa_meta.schema_migrations` metadata table.
- The released-image Compose file defaults to `latest` tags; pin `AGRAFA_BACKEND_TAG` and `AGRAFA_FRONTEND_TAG` in `.env` if you want a specific release.
- Reverse proxying is not bundled. If you want a single public entrypoint or TLS, place your own nginx, Traefik, or Caddy in front.

## Common Commands

| Area | Command |
| --- | --- |
| Frontend dev server | `cd frontend && pnpm dev` |
| Frontend production build | `cd frontend && pnpm build` |
| Backend run | `cd backend && make run` |
| Backend migrate up | `cd backend && make migrate-up` |
| Backend seed sample data | `cd backend && make seed` |
| Agent run | `cd agent && make run` |
| Backend tests | `cd backend && go test ./...` |
| Agent tests | `cd agent && go test ./...` |

## Notes

- The backend uses a single PostgreSQL database configured with `POSTGRES_URI`.
- Backend migrations live under `backend/src/db/migrations/app`.
- The agent can load fallback health checks from `agent/health_checks.json`, but backend-provided config is the normal path.
- The agent can also be published as a container image to GitHub Container Registry via `.github/workflows/publish-agent-container.yml`.

## Status

This repo is organized as a multi-part application rather than a single root package. Development commands are run from `frontend`, `backend`, and `agent` individually.
