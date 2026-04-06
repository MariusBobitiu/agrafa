# Agrafa Agent

Minimal Go agent for Agrafa. It sends:

- backend-fetched agent config from `/v1/agent/config`
- node heartbeats to `/v1/agent/heartbeat`
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
