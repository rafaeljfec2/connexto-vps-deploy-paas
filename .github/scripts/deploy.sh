#!/usr/bin/env bash
set -euo pipefail

HOST="${SERVER_HOST:?SERVER_HOST is required}"
USER="${SERVER_USER:?SERVER_USER is required}"
PORT="${SERVER_PORT:-22}"

REGISTRY="${REGISTRY:?REGISTRY is required}"
IMAGE_NAME="${IMAGE_NAME:?IMAGE_NAME is required}"
GHCR_USER="${GHCR_USER:?GHCR_USER is required}"
GHCR_PAT="${GHCR_PAT:?GHCR_PAT is required}"
CONTAINER_NAME="${CONTAINER_NAME:?CONTAINER_NAME is required}"

APP_ENV="${APP_ENV:?APP_ENV is required}"
LOG_LEVEL="${LOG_LEVEL:-info}"
DATABASE_URL="${DATABASE_URL:?DATABASE_URL is required}"
DEPLOY_WORKERS="${DEPLOY_WORKERS:-2}"
DEPLOY_TIMEOUT="${DEPLOY_TIMEOUT:-600}"
HEALTH_CHECK_TIMEOUT="${HEALTH_CHECK_TIMEOUT:-60}"
HEALTH_CHECK_RETRIES="${HEALTH_CHECK_RETRIES:-3}"

GITHUB_PAT="${GITHUB_PAT:-}"
GITHUB_WEBHOOK_SECRET="${GITHUB_WEBHOOK_SECRET:?GITHUB_WEBHOOK_SECRET is required}"
GITHUB_WEBHOOK_URL="${GITHUB_WEBHOOK_URL:?GITHUB_WEBHOOK_URL is required}"
GITHUB_CLIENT_ID="${GITHUB_CLIENT_ID:?GITHUB_CLIENT_ID is required}"
GITHUB_CLIENT_SECRET="${GITHUB_CLIENT_SECRET:?GITHUB_CLIENT_SECRET is required}"
GITHUB_OAUTH_CALLBACK_URL="${GITHUB_OAUTH_CALLBACK_URL:?GITHUB_OAUTH_CALLBACK_URL is required}"
GITHUB_APP_ID="${GITHUB_APP_ID:?GITHUB_APP_ID is required}"
GITHUB_APP_PRIVATE_KEY_BASE64="${GITHUB_APP_PRIVATE_KEY_BASE64:?GITHUB_APP_PRIVATE_KEY_BASE64 is required}"
GITHUB_APP_INSTALL_URL="${GITHUB_APP_INSTALL_URL:?GITHUB_APP_INSTALL_URL is required}"

TOKEN_ENCRYPTION_KEY="${TOKEN_ENCRYPTION_KEY:?TOKEN_ENCRYPTION_KEY is required}"
FRONTEND_URL="${FRONTEND_URL:?FRONTEND_URL is required}"
COOKIE_DOMAIN="${COOKIE_DOMAIN:-}"
SESSION_SECURE="${SESSION_SECURE:-true}"
CORS_ORIGINS="${CORS_ORIGINS:?CORS_ORIGINS is required}"

CLOUDFLARE_SERVER_IP="${CLOUDFLARE_SERVER_IP:-}"
API_DOMAIN="${API_DOMAIN:?API_DOMAIN is required}"
TRAEFIK_NETWORK="${TRAEFIK_NETWORK:?TRAEFIK_NETWORK is required}"
TRAEFIK_API_URL="${TRAEFIK_API_URL:-http://traefik:8081}"
GRPC_SERVER_ADDR="${GRPC_SERVER_ADDR:-${API_DOMAIN}:50051}"
AGENT_TLS_INSECURE_SKIP_VERIFY="${AGENT_TLS_INSECURE_SKIP_VERIFY:-false}"

SUDO_PASS="${SERVER_SUDO_PASSWORD:?SERVER_SUDO_PASSWORD is required (used only for writing the GitHub App private key and creating /opt/flowdeploy)}"

echo "Starting deployment to ${HOST}:${PORT}"
echo "Image: ${REGISTRY}/${IMAGE_NAME}:latest"

SSH_OPTS=(
  -p "${PORT}"
  -o StrictHostKeyChecking=accept-new
  -o ServerAliveInterval=30
  -o ConnectTimeout=15
  -o BatchMode=yes
)

q() { printf '%q' "$1"; }

PROLOG=$(mktemp)
BODY=$(mktemp)
trap 'rm -f "${PROLOG}" "${BODY}"' EXIT

cat > "${PROLOG}" <<PROLOG_END
set -euo pipefail
REGISTRY=$(q "${REGISTRY}")
IMAGE_NAME=$(q "${IMAGE_NAME}")
GHCR_USER=$(q "${GHCR_USER}")
GHCR_PAT=$(q "${GHCR_PAT}")
CONTAINER_NAME=$(q "${CONTAINER_NAME}")
APP_ENV=$(q "${APP_ENV}")
LOG_LEVEL=$(q "${LOG_LEVEL}")
DATABASE_URL=$(q "${DATABASE_URL}")
DEPLOY_WORKERS=$(q "${DEPLOY_WORKERS}")
DEPLOY_TIMEOUT=$(q "${DEPLOY_TIMEOUT}")
HEALTH_CHECK_TIMEOUT=$(q "${HEALTH_CHECK_TIMEOUT}")
HEALTH_CHECK_RETRIES=$(q "${HEALTH_CHECK_RETRIES}")
GITHUB_PAT=$(q "${GITHUB_PAT}")
GITHUB_WEBHOOK_SECRET=$(q "${GITHUB_WEBHOOK_SECRET}")
GITHUB_WEBHOOK_URL=$(q "${GITHUB_WEBHOOK_URL}")
GITHUB_CLIENT_ID=$(q "${GITHUB_CLIENT_ID}")
GITHUB_CLIENT_SECRET=$(q "${GITHUB_CLIENT_SECRET}")
GITHUB_OAUTH_CALLBACK_URL=$(q "${GITHUB_OAUTH_CALLBACK_URL}")
GITHUB_APP_ID=$(q "${GITHUB_APP_ID}")
GITHUB_APP_PRIVATE_KEY_BASE64=$(q "${GITHUB_APP_PRIVATE_KEY_BASE64}")
GITHUB_APP_INSTALL_URL=$(q "${GITHUB_APP_INSTALL_URL}")
TOKEN_ENCRYPTION_KEY=$(q "${TOKEN_ENCRYPTION_KEY}")
FRONTEND_URL=$(q "${FRONTEND_URL}")
COOKIE_DOMAIN=$(q "${COOKIE_DOMAIN}")
SESSION_SECURE=$(q "${SESSION_SECURE}")
CORS_ORIGINS=$(q "${CORS_ORIGINS}")
CLOUDFLARE_SERVER_IP=$(q "${CLOUDFLARE_SERVER_IP}")
API_DOMAIN=$(q "${API_DOMAIN}")
TRAEFIK_NETWORK=$(q "${TRAEFIK_NETWORK}")
TRAEFIK_API_URL=$(q "${TRAEFIK_API_URL}")
GRPC_SERVER_ADDR=$(q "${GRPC_SERVER_ADDR}")
AGENT_TLS_INSECURE_SKIP_VERIFY=$(q "${AGENT_TLS_INSECURE_SKIP_VERIFY}")
SUDO_PASS=$(q "${SUDO_PASS}")
PROLOG_END

cat > "${BODY}" <<'BODY_END'
echo "${GHCR_PAT}" | docker login "${REGISTRY}" -u "${GHCR_USER}" --password-stdin >/dev/null

docker pull "${REGISTRY}/${IMAGE_NAME}:latest"

docker network create "${TRAEFIK_NETWORK}" 2>/dev/null || true

docker stop "${CONTAINER_NAME}" 2>/dev/null || true
docker rm   "${CONTAINER_NAME}" 2>/dev/null || true

echo "${SUDO_PASS}" | sudo -S -p '' install -d -m 0755 -o root -g root /opt/flowdeploy

TMP_KEY="$(mktemp)"
trap 'rm -f "${TMP_KEY}"' EXIT
printf '%s' "${GITHUB_APP_PRIVATE_KEY_BASE64}" | base64 -d > "${TMP_KEY}"
chmod 600 "${TMP_KEY}"
if ! head -c 28 "${TMP_KEY}" | grep -q '^-----BEGIN '; then
  echo "GITHUB_APP_PRIVATE_KEY_BASE64 decoded to a non-PEM payload" >&2
  exit 1
fi
echo "${SUDO_PASS}" | sudo -S -p '' install -m 0640 -o root -g docker "${TMP_KEY}" /opt/flowdeploy/github-app-private-key.pem
rm -f "${TMP_KEY}"
trap - EXIT

LABELS_FILE="$(mktemp)"
cat > "${LABELS_FILE}" <<LABELS
traefik.enable=true
traefik.http.routers.${CONTAINER_NAME}.rule=Host(\`${API_DOMAIN}\`)
traefik.http.routers.${CONTAINER_NAME}.entrypoints=websecure
traefik.http.routers.${CONTAINER_NAME}.tls=true
traefik.http.routers.${CONTAINER_NAME}.tls.certresolver=letsencrypt
traefik.http.services.${CONTAINER_NAME}.loadbalancer.server.port=8080
traefik.http.routers.${CONTAINER_NAME}-http.rule=Host(\`${API_DOMAIN}\`)
traefik.http.routers.${CONTAINER_NAME}-http.entrypoints=web
traefik.http.routers.${CONTAINER_NAME}-http.middlewares=redirect-to-https
traefik.http.middlewares.redirect-to-https.redirectscheme.scheme=https
traefik.tcp.routers.${CONTAINER_NAME}-grpc.rule=HostSNI(\`*\`)
traefik.tcp.routers.${CONTAINER_NAME}-grpc.entrypoints=grpc
traefik.tcp.routers.${CONTAINER_NAME}-grpc.tls.passthrough=true
traefik.tcp.services.${CONTAINER_NAME}-grpc.loadbalancer.server.port=50051
LABELS

DOCKER_GID=$(stat -c '%g' /var/run/docker.sock)
echo "Docker socket GID: ${DOCKER_GID}"

DOCKER_RUN_ARGS=(
  -d
  --name "${CONTAINER_NAME}"
  --network "${TRAEFIK_NETWORK}"
  --restart unless-stopped
  --group-add "${DOCKER_GID}"
  -p 9005:8080
  --dns 8.8.8.8 --dns 1.1.1.1
  -v /var/run/docker.sock:/var/run/docker.sock
  -v /opt/flowdeploy:/opt/flowdeploy
  -v flowdeploy-data:/data/apps
  -v /etc/nginx:/etc/nginx:ro
  -v /etc/letsencrypt:/etc/letsencrypt:ro
  -e APP_ENV="${APP_ENV}"
  -e HOST=0.0.0.0
  -e PORT=8080
  -e LOG_LEVEL="${LOG_LEVEL}"
  -e DATABASE_URL="${DATABASE_URL}"
  -e DEPLOY_DATA_DIR=/data/apps
  -e DEPLOY_WORKERS="${DEPLOY_WORKERS}"
  -e DEPLOY_TIMEOUT="${DEPLOY_TIMEOUT}"
  -e HEALTH_CHECK_TIMEOUT="${HEALTH_CHECK_TIMEOUT}"
  -e HEALTH_CHECK_RETRIES="${HEALTH_CHECK_RETRIES}"
  -e DOCKER_HOST=unix:///var/run/docker.sock
  -e GIT_HUB_PAT="${GITHUB_PAT}"
  -e GIT_HUB_WEBHOOK_SECRET="${GITHUB_WEBHOOK_SECRET}"
  -e GIT_HUB_WEBHOOK_URL="${GITHUB_WEBHOOK_URL}"
  -e GIT_HUB_CLIENT_ID="${GITHUB_CLIENT_ID}"
  -e GIT_HUB_CLIENT_SECRET="${GITHUB_CLIENT_SECRET}"
  -e GIT_HUB_OAUTH_CALLBACK_URL="${GITHUB_OAUTH_CALLBACK_URL}"
  -e GIT_HUB_APP_ID="${GITHUB_APP_ID}"
  -e GIT_HUB_APP_PRIVATE_KEY_PATH=/opt/flowdeploy/github-app-private-key.pem
  -e GIT_HUB_APP_INSTALL_URL="${GITHUB_APP_INSTALL_URL}"
  -e TOKEN_ENCRYPTION_KEY="${TOKEN_ENCRYPTION_KEY}"
  -e FRONTEND_URL="${FRONTEND_URL}"
  -e COOKIE_DOMAIN="${COOKIE_DOMAIN}"
  -e SESSION_SECURE="${SESSION_SECURE}"
  -e CORS_ORIGINS="${CORS_ORIGINS}"
  -e GRPC_ENABLED=true
  -e AGENT_BINARY_PATH=/app/agent
  -e API_BASE_URL=https://${API_DOMAIN}
  -e GRPC_PORT=50051
  -e AGENT_GRPC_PORT=50052
  -e GRPC_SERVER_ADDR="${GRPC_SERVER_ADDR}"
  -e TRAEFIK_API_URL="${TRAEFIK_API_URL}"
  -e AGENT_TLS_INSECURE_SKIP_VERIFY="${AGENT_TLS_INSECURE_SKIP_VERIFY}"
)

if [ -n "${CLOUDFLARE_SERVER_IP}" ]; then
  DOCKER_RUN_ARGS+=( -e CLOUDFLARE_SERVER_IP="${CLOUDFLARE_SERVER_IP}" )
fi

DOCKER_RUN_ARGS+=( --label-file "${LABELS_FILE}" "${REGISTRY}/${IMAGE_NAME}:latest" )

docker run "${DOCKER_RUN_ARGS[@]}"

rm -f "${LABELS_FILE}"

sleep 4
docker ps --filter "name=${CONTAINER_NAME}" --format '{{.Status}}'
BODY_END

cat "${PROLOG}" "${BODY}" | ssh "${SSH_OPTS[@]}" "${USER}@${HOST}" bash -s

echo "Deployment completed."
