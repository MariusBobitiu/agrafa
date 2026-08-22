# Agrafa

<p align="center">
  <img src="frontend/public/logo.png" alt="Agrafa" width="96" />
</p>

<p align="center">
  Lightweight, self-hosted monitoring for small deployments, personal infrastructure, and side projects.
</p>

<p align="center">
  <a href="https://github.com/MariusBobitiu/agrafa/actions/workflows/ci.yml"><img src="https://github.com/MariusBobitiu/agrafa/actions/workflows/ci.yml/badge.svg" alt="CI" /></a>
  <a href="LICENSE"><img src="https://img.shields.io/badge/license-MIT-blue.svg" alt="MIT License" /></a>
</p>

<p align="center">
  <a href="#quick-start">Quick start</a> ·
  <a href="#api-documentation">API docs</a> ·
  <a href="CONTRIBUTING.md">Contributing</a> ·
  <a href="SECURITY.md">Security</a> ·
  <a href="LICENSE">License</a>
</p>

<p align="center">
  <a href="https://res.cloudinary.com/mb-labs/image/upload/v1787432958/overview-page_a5w9eb.png">
    <img src="https://res.cloudinary.com/mb-labs/image/upload/v1787432958/overview-page_a5w9eb.png" alt="Agrafa overview dashboard" />
  </a>
</p>

Agrafa combines a Go API, a React dashboard, and a small Go agent in one focused observability stack. The agent reports heartbeats, host metrics, and health-check results; the backend turns those signals into current state, events, and alerts; and the frontend gives you one operational view of your nodes and services.

> Agrafa is currently early-stage software. Expect deployment details and APIs to evolve before a stable release.

## Highlights

- Monitor node availability, CPU, memory, disk usage, and HTTP health checks.
- Keep state evaluation and event history in a central PostgreSQL-backed API.
- Configure monitored services from the dashboard and deliver optional email alerts.
- Run the agent on Linux with host-level metrics or use it during local development.
- Self-host the stack with Docker Compose and pin every Agrafa image to one product version.

## Quick Start

The non-interactive installer creates an `agrafa/` directory, generates secrets, detects the server IP when possible, and starts PostgreSQL, the backend, and the frontend:

```bash
curl -fsSL https://raw.githubusercontent.com/MariusBobitiu/agrafa/main/install/install.sh | bash
```

It exposes:

- Frontend: `http://server_ip:8080`
- Backend API: `http://server_ip:8081`
- Agent API base URL: `http://server_ip:8081`

The installer uses the `latest` images by default. For a reproducible deployment, pin one released version across the stack:

```bash
curl -fsSL https://raw.githubusercontent.com/MariusBobitiu/agrafa/main/install/install.sh \
  | AGRAFA_VERSION=0.1.0 bash
```

Domain routing and TLS are deliberately left to your reverse proxy of choice.

For configuration and alternative deployment paths, continue to [Self-Hosting with Docker Compose](#self-hosting-with-docker-compose). For API exploration, see [API Documentation](#api-documentation).

## Architecture

```text
┌──────────────┐       heartbeats, metrics, checks        ┌──────────────┐
│ Agrafa agent │ ───────────────────────────────────────▶ │  Go backend  │
└──────────────┘                                          │ + PostgreSQL │
                                                          └──────┬───────┘
                                                                 │ state, events,
                                                                 │ alerts, settings
                                                                 ▼
                                                          ┌──────────────┐
                                                          │ React UI     │
                                                          └──────────────┘
```

The backend is the source of truth. Agents report raw observations; the backend evaluates them; and the frontend reads the resulting operational state.

| Folder | Purpose | Main stack |
| --- | --- | --- |
| `frontend` | Dashboard for authentication, overview, nodes, services, alerts, and settings | React 19, Vite, TypeScript, Tailwind |
| `backend` | API, ingestion, state evaluation, events, alerts, and read models | Go, Chi, PostgreSQL, sqlc |
| `agent` | Host agent for heartbeats, metrics, and health checks | Go, gopsutil |
| `install` | Non-interactive Docker Compose installer | Bash, Docker Compose |

## Self-Hosting with Docker Compose

The root Compose files provide two paths:

- `docker-compose.yml` pulls released backend and frontend images from GitHub Container Registry.
- `docker-compose.local.yml` builds backend and frontend from the current checkout.

Copy and configure the environment file:

```bash
cp .env.example .env
```

At minimum, replace `POSTGRES_PASSWORD` and `APP_SECRET`, then review `POSTGRES_URI`, `APP_BASE_URL`, `APP_ALLOWED_ORIGINS`, and `VITE_API_URL` for your environment. Email settings are optional.

Released deployments use these image settings:

```env
GHCR_OWNER=mariusbobitiu
AGRAFA_VERSION=0.1.0
```

A product tag such as `v0.1.0` publishes matching `agrafa-backend:0.1.0`, `agrafa-frontend:0.1.0`, and `agrafa-agent:0.1.0` images. Use the same `AGRAFA_VERSION` for compatibility across the stack.

Start released images:

```bash
docker compose up -d
```

Or build locally:

```bash
docker compose -f docker-compose.local.yml up -d --build
```

The root Compose setup exposes the frontend at `http://localhost:5173`, the API at `http://localhost:8080/v1`, and Swagger UI at `http://localhost:8080/docs`.

## Local Development

### Prerequisites

- The Go versions declared in `backend/go.mod` and `agent/go.mod`
- Node.js 24 and pnpm 10 for the frontend
- PostgreSQL and `psql` for backend migrations and seed data

### Backend

```bash
cd backend
cp .env.example .env
```

Set `POSTGRES_URI` in `backend/.env`, then run:

```bash
make migrate-up
make seed
make run
```

Local endpoints:

- API: `http://localhost:8080/v1`
- Swagger UI: `http://localhost:8080/docs`
- OpenAPI JSON: `http://localhost:8080/openapi/swagger.json`

### Agent

```bash
cd agent
cp .env.example .env
make run
```

Set `AGRAFA_API_BASE_URL` and `AGRAFA_AGENT_TOKEN` in `agent/.env`. `AGRAFA_NODE_ID` is optional compatibility configuration; the normal flow learns the node ID from the backend. See the [agent guide](agent/README.md) for runtime and container details.

### Frontend

```bash
cd frontend
cp .env.example .env
pnpm install
pnpm dev
```

The frontend uses `VITE_API_URL` from `frontend/.env` and proxies `/v1` requests to the backend during development.

## API Documentation

The backend serves interactive Swagger UI at `http://localhost:8080/docs` and the OpenAPI document at `http://localhost:8080/openapi/swagger.json` while it is running. The generated specification is also kept in [`backend/docs`](backend/docs).

## Common Commands

| Area | Command |
| --- | --- |
| Frontend development | `cd frontend && pnpm dev` |
| Frontend validation | `cd frontend && vp lint && vp run build` |
| Backend tests | `cd backend && go test ./...` |
| Backend static checks | `cd backend && go vet ./...` |
| Backend build | `cd backend && go build ./...` |
| Agent tests | `cd agent && go test ./...` |
| Agent static checks | `cd agent && go vet ./...` |
| Agent build | `cd agent && go build ./...` |
| Backend migrations | `cd backend && make migrate-up` |
| Backend sample data | `cd backend && make seed` |

## Releases

Agrafa has one product version. Pushing a SemVer-style tag such as `v0.1.0` or `v0.1.0-rc.1` runs the unified publish workflow and publishes backend, frontend, and agent images with the same version. Component-specific release tags are not used.

## Contributing and Security

Issues and focused pull requests are welcome. See the [contributing guide](CONTRIBUTING.md) for setup, checks, and pull request expectations.

Report vulnerabilities privately according to the [security policy](SECURITY.md). Agrafa is available under the [MIT License](LICENSE).
