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

## Status

This repo is organized as a multi-part application rather than a single root package. Development commands are run from `frontend`, `backend`, and `agent` individually.
