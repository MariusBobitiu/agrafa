# Contributing to Agrafa

Thanks for helping improve Agrafa. Bug reports, feature ideas, documentation fixes, and focused code changes are welcome.

## Prerequisites

- Go 1.25 or newer for the backend
- Go 1.24.2 or newer for the agent
- Node.js 24, pnpm 10, and Vite+ (`vp`) for the frontend
- PostgreSQL and `psql` for backend development
- Docker with Docker Compose for the container-based setup

## Local Setup

Clone the repository, then configure and run each component in a separate terminal.

### Backend

Create a PostgreSQL database, then:

```bash
cd backend
cp .env.example .env
```

Set `POSTGRES_URI` and replace `APP_SECRET` in `backend/.env`, then run:

```bash
make migrate-up
make seed
make run
```

The API is available at `http://localhost:8080/v1` and Swagger UI at `http://localhost:8080/docs`.

### Frontend

```bash
cd frontend
cp .env.example .env
pnpm install
pnpm dev
```

The development server uses `VITE_API_URL` from `frontend/.env` and is available at `http://localhost:5173`.

### Agent

Create a node in the dashboard and copy its agent token, then:

```bash
cd agent
cp .env.example .env
```

Set `AGRAFA_API_BASE_URL` and `AGRAFA_AGENT_TOKEN` in `agent/.env`, then run:

```bash
make run
```

See the [agent guide](agent/README.md) for configuration and container usage.

### Docker Compose Alternative

To build and run the backend, frontend, and PostgreSQL from the checkout:

```bash
cp .env.example .env
docker compose -f docker-compose.local.yml up -d --build
```

Replace the example secrets and review the URLs in `.env` before starting the stack.

## Checks

Run the checks relevant to your change before opening a pull request:

```bash
cd backend && go test ./... && go vet ./... && go build ./...
cd agent && go test ./... && go vet ./... && go build ./...
cd frontend && vp install --frozen-lockfile && vp lint && vp run build
```

## Pull Requests

- Keep each pull request focused and explain the reason for the change.
- Link the relevant issue when one exists.
- Add or update tests for behavior changes.
- Update documentation when setup, configuration, or user-facing behavior changes.
- Make sure the relevant checks pass and avoid committing secrets or local `.env` files.

Please report security issues privately using the process in [SECURITY.md](SECURITY.md), not through a public issue.
