#!/usr/bin/env bash

set -euo pipefail

APP_NAME="agrafa"
INSTALL_DIR="${APP_NAME}"
REPO_OWNER="mariusbobitiu"
RAW_BASE="https://raw.githubusercontent.com/${REPO_OWNER}/agrafa/main/install"
DEFAULT_PUBLIC_IP="127.0.0.1"
FRONTEND_PORT="8080"
BACKEND_PORT="8081"

need_cmd() {
  command -v "$1" >/dev/null 2>&1 || {
    echo "Missing required command: $1"
    exit 1
  }
}

need_cmd docker
need_cmd curl
need_cmd openssl

if ! docker compose version >/dev/null 2>&1; then
  echo "Docker Compose plugin is required."
  exit 1
fi

detect_public_ip() {
  local ip=""
  local url

  for url in \
    "https://icanhazip.com" \
    "https://api.ipify.org" \
    "https://ifconfig.me"
  do
    ip="$(curl -fsSL --max-time 5 "$url" 2>/dev/null | tr -d '[:space:]' || true)"
    if [[ "$ip" =~ ^([0-9]{1,3}\.){3}[0-9]{1,3}$ ]]; then
      printf '%s\n' "$ip"
      return 0
    fi
  done

  hostname -I 2>/dev/null | awk '{print $1}' || true
}

GHCR_OWNER="${GHCR_OWNER:-$REPO_OWNER}"

mkdir -p "$INSTALL_DIR"
cd "$INSTALL_DIR"

POSTGRES_DB="agrafa"
POSTGRES_USER="agrafa"
POSTGRES_PASSWORD="$(openssl rand -base64 24 | tr -d '\n' | tr '/+' 'ab' | cut -c1-24)"
APP_SECRET="$(openssl rand -base64 48 | tr -d '\n')"
SERVER_IP="$(detect_public_ip | tr -d '[:space:]')"
if [ -z "${SERVER_IP:-}" ]; then
  SERVER_IP="$DEFAULT_PUBLIC_IP"
fi

APP_BASE_URL="http://${SERVER_IP}:${FRONTEND_PORT}"
APP_ALLOWED_ORIGINS="${APP_BASE_URL}"

curl -fsSL "${RAW_BASE}/docker-compose.yml" -o docker-compose.yml

cat > .env <<EOF
GHCR_OWNER=${GHCR_OWNER}

POSTGRES_DB=${POSTGRES_DB}
POSTGRES_USER=${POSTGRES_USER}
POSTGRES_PASSWORD=${POSTGRES_PASSWORD}

POSTGRES_URI=postgres://${POSTGRES_USER}:${POSTGRES_PASSWORD}@postgres:5432/${POSTGRES_DB}?sslmode=disable
POSTGRES_MIGRATIONS_SCHEMA=agrafa_meta
POSTGRES_MIGRATIONS_TABLE=schema_migrations

APP_ENV=production
PORT=8080
APP_BASE_URL=${APP_BASE_URL}
APP_ALLOWED_ORIGINS=${APP_ALLOWED_ORIGINS}
VITE_API_BASE_URL=http://${SERVER_IP}:${BACKEND_PORT}
APP_SECRET=${APP_SECRET}

NODE_HEARTBEAT_TTL_SECONDS=60
NODE_EXPIRY_CHECK_INTERVAL_SECONDS=15
MANAGED_SERVICE_CHECK_INTERVAL_SECONDS=15
MANAGED_SERVICE_CHECK_TIMEOUT_SECONDS=10
SESSION_TTL_DAYS=7
SESSION_REMEMBER_TTL_DAYS=30

FRONTEND_HOST_PORT=${FRONTEND_PORT}
BACKEND_HOST_PORT=${BACKEND_PORT}
EOF

docker compose up -d

echo "Agrafa installed."
echo "Frontend: ${APP_BASE_URL}"
echo "Backend/API: http://${SERVER_IP}:${BACKEND_PORT}"
echo "Agent API base URL: http://${SERVER_IP}:${BACKEND_PORT}"
