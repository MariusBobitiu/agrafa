#!/usr/bin/env bash

set -euo pipefail

APP_NAME="agrafa"
INSTALL_DIR="${APP_NAME}"
REPO_OWNER="MariusBobitiu"
RAW_BASE="https://raw.githubusercontent.com/${REPO_OWNER}/agrafa/main/install"

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

echo "Agrafa installer"
echo

read -rp "Install directory [agrafa]: " input_install_dir
if [ -n "${input_install_dir:-}" ]; then
  INSTALL_DIR="$input_install_dir"
fi

read -rp "GitHub Container Registry owner/user [${REPO_OWNER}]: " input_ghcr_owner
GHCR_OWNER="${input_ghcr_owner:-$REPO_OWNER}"

mkdir -p "$INSTALL_DIR"
cd "$INSTALL_DIR"

POSTGRES_DB="agrafa"
POSTGRES_USER="agrafa"
POSTGRES_PASSWORD="$(openssl rand -base64 24 | tr -d '\n' | tr '/+' 'ab' | cut -c1-24)"
APP_SECRET="$(openssl rand -base64 48 | tr -d '\n')"

read -rp "Do you want to use a domain with HTTPS? [y/N]: " HAS_DOMAIN
HAS_DOMAIN="${HAS_DOMAIN:-N}"

SERVER_IP="$(curl -fsSL https://icanhazip.com 2>/dev/null | tr -d '\n' || true)"

if [[ "$HAS_DOMAIN" =~ ^[Yy]$ ]]; then
  read -rp "Domain (e.g. agrafa.example.com): " DOMAIN
  if [ -z "${DOMAIN:-}" ]; then
    echo "Domain is required."
    exit 1
  fi

  read -rp "Email for Let's Encrypt: " LETSENCRYPT_EMAIL
  if [ -z "${LETSENCRYPT_EMAIL:-}" ]; then
    echo "Let's Encrypt email is required."
    exit 1
  fi

  TRAEFIK_ENABLED="true"
  APP_BASE_URL="https://${DOMAIN}"
  APP_ALLOWED_ORIGINS="https://${DOMAIN}"
  FRONTEND_HOST_PORT=8080
  BACKEND_HOST_PORT=8081
  COMPOSE_PROFILES="domain"
else
  TRAEFIK_ENABLED="false"
  DOMAIN="localhost"
  LETSENCRYPT_EMAIL=""
  APP_BASE_URL="http://${SERVER_IP:-localhost}:8080"
  APP_ALLOWED_ORIGINS="${APP_BASE_URL}"
  FRONTEND_HOST_PORT=8080
  BACKEND_HOST_PORT=8081
  COMPOSE_PROFILES=""
fi

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
APP_SECRET=${APP_SECRET}

NODE_HEARTBEAT_TTL_SECONDS=60
NODE_EXPIRY_CHECK_INTERVAL_SECONDS=15
MANAGED_SERVICE_CHECK_INTERVAL_SECONDS=15
MANAGED_SERVICE_CHECK_TIMEOUT_SECONDS=10
SESSION_TTL_DAYS=7
SESSION_REMEMBER_TTL_DAYS=30

TRAEFIK_ENABLED=${TRAEFIK_ENABLED}
DOMAIN=${DOMAIN}
LETSENCRYPT_EMAIL=${LETSENCRYPT_EMAIL}
FRONTEND_HOST_PORT=${FRONTEND_HOST_PORT}
BACKEND_HOST_PORT=${BACKEND_HOST_PORT}
COMPOSE_PROFILES=${COMPOSE_PROFILES}

EMAIL_ENABLED=false
EMAIL_PROVIDER=resend
EMAIL_RESEND_API_KEY=
EMAIL_RESEND_DOMAIN=
EOF

echo
echo "Starting Agrafa..."

if [ -n "${COMPOSE_PROFILES}" ]; then
  docker compose --profile "${COMPOSE_PROFILES}" up -d
else
  docker compose up -d
fi

echo
echo "Agrafa installed."

if [[ "$HAS_DOMAIN" =~ ^[Yy]$ ]]; then
  echo "Frontend: https://${DOMAIN}"
  echo "Backend/API: https://${DOMAIN}/api"
  echo "Agent API base URL: https://${DOMAIN}/api"
else
  echo "Frontend: ${APP_BASE_URL}"
  echo "Backend/API: http://${SERVER_IP:-localhost}:8081"
  echo "Agent API base URL: http://${SERVER_IP:-localhost}:8081"
fi