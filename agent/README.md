# Agrafa Agent

Minimal Go agent for Agrafa. It sends:

- backend-fetched agent config from `/v1/agent/config`
- node heartbeats to `/v1/agent/heartbeat`
- best-effort shutdown signals to `/v1/agent/shutdown`
- node CPU, memory, and disk metrics to `/v1/agent/metrics`
- backend-configured HTTP health check results to `/v1/agent/health`

## Config

Runtime config is environment-based. The agent will read `.env` or `.env.local` if present.

Required:

- `AGRAFA_API_BASE_URL`
- `AGRAFA_AGENT_TOKEN`
- `AGRAFA_NODE_ID`

The agent sends `AGRAFA_AGENT_TOKEN` as the `X-Agent-Token` header on every request to the backend. `AGRAFA_NODE_ID` can still be present in heartbeat and metrics payloads for compatibility, but the backend authenticates the node from the token.

Common optional values:

- `AGRAFA_API_TIMEOUT_SECONDS`
- `AGRAFA_API_RETRY_COUNT`
- `AGRAFA_HEARTBEAT_INTERVAL_SECONDS`
- `AGRAFA_METRICS_INTERVAL_SECONDS`
- `AGRAFA_HEALTH_INTERVAL_SECONDS`
- `AGRAFA_CONFIG_REFRESH_SECONDS`
- `AGRAFA_HTTP_TIMEOUT_SECONDS`
- `AGRAFA_DISK_PATH`
- `AGRAFA_HEALTH_CHECKS_FILE`

Transport behavior stays intentionally small:

- backend API requests use `AGRAFA_API_TIMEOUT_SECONDS`
- transient failures retry up to `AGRAFA_API_RETRY_COUNT` additional times
- retries apply only to network errors, timeouts, and `5xx` backend responses
- `4xx` responses, including `401`, do not retry
- auth failures are logged clearly and repeated identical auth errors are suppressed
- on `SIGINT` or `SIGTERM`, the agent makes one best-effort shutdown request with a reason and stops waiting after 5 seconds

`AGRAFA_HTTP_TIMEOUT_SECONDS` remains the default timeout for outbound HTTP health checks run by the agent.

Assigned health checks now come from the backend by default through `/v1/agent/config`. The agent refreshes that in-memory config every `AGRAFA_CONFIG_REFRESH_SECONDS` seconds and keeps the last known config if a refresh fails.

`AGRAFA_HEALTH_CHECKS_FILE` is still available as an optional fallback/dev-only source. If present, those checks are loaded at startup and used only until backend config replaces them, or while backend config fetches are failing.

## Run

```bash
cp .env.example .env
go run .
```

Or:

```bash
make run
```

## Docker

Build locally:

```bash
docker build -t agrafa-agent ./agent
```

Pull from GitHub Container Registry after the publish workflow runs:

```bash
docker pull ghcr.io/mariusbobitiu/agrafa-agent:latest
```

Recommended Linux runtime when you want host-level metrics instead of container-only metrics:

```bash
docker run --rm \
  --name agrafa-agent \
  --pid=host \
  -e AGRAFA_API_BASE_URL=https://your-backend.example.com/v1 \
  -e AGRAFA_AGENT_TOKEN=your-agent-token \
  -e AGRAFA_NODE_ID=123 \
  -e HOST_PROC=/host/proc \
  -e HOST_SYS=/host/sys \
  -e HOST_ETC=/host/etc \
  -e HOST_ROOT=/host \
  -e AGRAFA_DISK_PATH=/host \
  -v /proc:/host/proc:ro \
  -v /sys:/host/sys:ro \
  -v /etc:/host/etc:ro \
  -v /:/host:ro \
  ghcr.io/mariusbobitiu/agrafa-agent:latest
```

Add `--network host` if your health checks target services bound to `localhost` on the monitored machine.

The GitHub Actions workflow publishes the image to GHCR on tags that match `agent-v*.*.*`, for example `agent-v0.1.0`. You can also trigger the workflow manually from the Actions tab.
