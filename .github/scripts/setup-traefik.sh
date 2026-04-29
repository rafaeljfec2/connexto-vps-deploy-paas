#!/usr/bin/env bash
set -euo pipefail

HOST="${SERVER_HOST:?SERVER_HOST is required}"
USER="${SERVER_USER:?SERVER_USER is required}"
PORT="${SERVER_PORT:-22}"
TRAEFIK_NETWORK="${TRAEFIK_NETWORK:?TRAEFIK_NETWORK is required}"
TRAEFIK_NETWORK_SECONDARY="${TRAEFIK_NETWORK_SECONDARY:-}"
LETSENCRYPT_EMAIL="${LETSENCRYPT_EMAIL:?LETSENCRYPT_EMAIL is required}"
SUDO_PASS="${SERVER_SUDO_PASSWORD:?SERVER_SUDO_PASSWORD is required}"

echo "Setting up Traefik on ${HOST}..."

SSH_OPTS=(
  -p "${PORT}"
  -o StrictHostKeyChecking=accept-new
  -o ServerAliveInterval=30
  -o ConnectTimeout=15
  -o BatchMode=yes
)

TN_Q=$(printf '%q' "${TRAEFIK_NETWORK}")
TNS_Q=$(printf '%q' "${TRAEFIK_NETWORK_SECONDARY}")
EMAIL_Q=$(printf '%q' "${LETSENCRYPT_EMAIL}")
PASS_Q=$(printf '%q' "${SUDO_PASS}")

PROLOG=$(mktemp)
BODY=$(mktemp)
trap 'rm -f "${PROLOG}" "${BODY}"' EXIT

cat > "${PROLOG}" <<PROLOG_END
set -euo pipefail
TRAEFIK_NETWORK=${TN_Q}
TRAEFIK_NETWORK_SECONDARY=${TNS_Q}
LETSENCRYPT_EMAIL=${EMAIL_Q}
SUDO_PASS=${PASS_Q}
PROLOG_END

cat > "${BODY}" <<'BODY_END'
docker network create "${TRAEFIK_NETWORK}" 2>/dev/null || true
if [ -n "${TRAEFIK_NETWORK_SECONDARY}" ]; then
  docker network create "${TRAEFIK_NETWORK_SECONDARY}" 2>/dev/null || true
fi

echo "${SUDO_PASS}" | sudo -S -p '' install -d -m 0755 -o root -g root /opt/traefik
echo "${SUDO_PASS}" | sudo -S -p '' install -d -m 0755 -o root -g root /opt/traefik/letsencrypt

TRAEFIK_YAML="$(mktemp)"
cat > "${TRAEFIK_YAML}" <<TRAEFIK_CONFIG
api:
  dashboard: true
  insecure: false

entryPoints:
  web:
    address: ":80"
  websecure:
    address: ":443"
  grpc:
    address: ":50051"
  traefik:
    address: ":8081"

providers:
  docker:
    endpoint: "unix:///var/run/docker.sock"
    exposedByDefault: false
    network: ${TRAEFIK_NETWORK}

certificatesResolvers:
  letsencrypt:
    acme:
      email: ${LETSENCRYPT_EMAIL}
      storage: /letsencrypt/acme.json
      tlsChallenge: {}

log:
  level: INFO
TRAEFIK_CONFIG

echo "${SUDO_PASS}" | sudo -S -p '' install -m 0644 -o root -g root "${TRAEFIK_YAML}" /opt/traefik/traefik.yml
rm -f "${TRAEFIK_YAML}"

if docker inspect traefik >/dev/null 2>&1; then
  echo "Traefik container already exists; recreating to apply config..."
  docker stop traefik >/dev/null 2>&1 || true
  docker rm   traefik >/dev/null 2>&1 || true
fi

echo "Starting Traefik (80, 443, 50051, 8081)..."
docker run -d \
  --name traefik \
  --network "${TRAEFIK_NETWORK}" \
  --restart unless-stopped \
  -p 80:80 \
  -p 443:443 \
  -p 50051:50051 \
  -p 8081:8081 \
  -v /var/run/docker.sock:/var/run/docker.sock:ro \
  -v /opt/traefik/traefik.yml:/etc/traefik/traefik.yml:ro \
  -v /opt/traefik/letsencrypt:/letsencrypt \
  traefik:latest

if [ -n "${TRAEFIK_NETWORK_SECONDARY}" ]; then
  docker network connect "${TRAEFIK_NETWORK_SECONDARY}" traefik 2>/dev/null || true
fi

docker ps --filter "name=^traefik$" --format '{{.Status}}'
BODY_END

cat "${PROLOG}" "${BODY}" | ssh "${SSH_OPTS[@]}" "${USER}@${HOST}" bash -s

echo "Traefik setup completed."
