#!/usr/bin/env bash
# Local deploy script run by the self-hosted GitHub Actions runner on the
# control-plane VPS. Pulls the backend image (pinned to the short SHA built
# in the same workflow run) from GHCR and recreates the
# `flowdeploy-backend` container using the env file that lives on the VPS
# (`~/flowdeploy/backend.env`).
#
# This script does NOT use sudo. The runner user must:
#   - belong to the `docker` group (already true for `ubuntu`);
#   - have the env file at ${HOME}/flowdeploy/backend.env (0600);
#   - be able to read /opt/flowdeploy/github-app-private-key.pem (mounted
#     read-only into the container, written once during VPS provisioning).
set -euo pipefail

REGISTRY="${REGISTRY:?REGISTRY is required}"
IMAGE_NAME="${IMAGE_NAME:?IMAGE_NAME is required}"
IMAGE_TAG="${IMAGE_TAG:?IMAGE_TAG is required (short SHA from build-and-push)}"
GHCR_USER="${GHCR_USER:?GHCR_USER is required}"
GHCR_PAT="${GHCR_PAT:?GHCR_PAT is required}"
CONTAINER_NAME="${CONTAINER_NAME:?CONTAINER_NAME is required}"
TRAEFIK_NETWORK="${TRAEFIK_NETWORK:?TRAEFIK_NETWORK is required}"
API_DOMAIN="${API_DOMAIN:?API_DOMAIN is required}"

ENV_FILE="${ENV_FILE:-${HOME}/flowdeploy/backend.env}"
GITHUB_APP_KEY_PATH="${GITHUB_APP_KEY_PATH:-/opt/flowdeploy/github-app-private-key.pem}"
HEALTH_TIMEOUT_SECONDS="${HEALTH_TIMEOUT_SECONDS:-180}"

IMAGE_REF="${REGISTRY}/${IMAGE_NAME}:${IMAGE_TAG}"

echo "===== Deploying ${IMAGE_REF} ====="

if [ ! -r "${ENV_FILE}" ]; then
  echo "ERROR: env file not readable at ${ENV_FILE}" >&2
  exit 1
fi
if [ ! -r "${GITHUB_APP_KEY_PATH}" ]; then
  echo "ERROR: GitHub App private key not readable at ${GITHUB_APP_KEY_PATH}" >&2
  exit 1
fi

# Defensive perm guard for the GitHub App private key. Loose perms here mean
# any local user on the VPS can sign JWTs as the FlowDeploy GitHub App.
KEY_PERM=$(stat -c '%a' "${GITHUB_APP_KEY_PATH}")
case "${KEY_PERM}" in
  600|640|660) ;;
  *)
    echo "ERROR: ${GITHUB_APP_KEY_PATH} has overly permissive mode ${KEY_PERM}; expected 0600/0640/0660. Fix on the VPS:" >&2
    echo "  sudo chown root:docker ${GITHUB_APP_KEY_PATH} && sudo chmod 0640 ${GITHUB_APP_KEY_PATH}" >&2
    exit 1
    ;;
esac

echo "===== Login to ${REGISTRY} ====="
echo "${GHCR_PAT}" | docker login "${REGISTRY}" -u "${GHCR_USER}" --password-stdin >/dev/null

echo "===== Pull image ====="
docker pull "${IMAGE_REF}"

echo "===== Ensure network ====="
docker network create "${TRAEFIK_NETWORK}" 2>/dev/null || true

PREV_IMAGE_ID=""
if docker inspect "${CONTAINER_NAME}" >/dev/null 2>&1; then
  PREV_IMAGE_ID=$(docker inspect --format '{{.Image}}' "${CONTAINER_NAME}" || true)
  echo "===== Previous image id: ${PREV_IMAGE_ID:-<none>} ====="
fi

echo "===== Stop previous container ====="
docker stop "${CONTAINER_NAME}" 2>/dev/null || true
docker rm   "${CONTAINER_NAME}" 2>/dev/null || true

LABELS_FILE="$(mktemp)"
trap 'rm -f "${LABELS_FILE}"' EXIT

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
echo "===== Run new container (docker socket gid=${DOCKER_GID}) ====="

# 9005 is bound to loopback so health probes can hit the API directly on the
# host without exposing it on the public NIC. External traffic always goes
# through Traefik (HTTPS + Let's Encrypt + middlewares).
docker run -d \
  --name "${CONTAINER_NAME}" \
  --network "${TRAEFIK_NETWORK}" \
  --restart unless-stopped \
  --group-add "${DOCKER_GID}" \
  -p 127.0.0.1:9005:8080 \
  --dns 8.8.8.8 --dns 1.1.1.1 \
  --env-file "${ENV_FILE}" \
  -v /var/run/docker.sock:/var/run/docker.sock \
  -v /opt/flowdeploy:/opt/flowdeploy:ro \
  -v flowdeploy-data:/data/apps \
  -v /etc/nginx:/etc/nginx:ro \
  -v /etc/letsencrypt:/etc/letsencrypt:ro \
  --label-file "${LABELS_FILE}" \
  "${IMAGE_REF}"

ACTIVE_IMAGE_ID="$(docker inspect --format '{{.Image}}' "${CONTAINER_NAME}")"
echo "===== Active image id: ${ACTIVE_IMAGE_ID} ====="

rollback() {
  local reason="$1"
  echo "ERROR: ${reason}" >&2
  docker inspect --format '{{json .State}}' "${CONTAINER_NAME}" >&2 2>/dev/null || true
  docker logs --tail 200 "${CONTAINER_NAME}" 2>&1 >&2 || true
  if [ -n "${PREV_IMAGE_ID}" ]; then
    echo "Attempting rollback to previous image ${PREV_IMAGE_ID}" >&2
    docker stop "${CONTAINER_NAME}" 2>/dev/null || true
    docker rm   "${CONTAINER_NAME}" 2>/dev/null || true
    docker run -d \
      --name "${CONTAINER_NAME}" \
      --network "${TRAEFIK_NETWORK}" \
      --restart unless-stopped \
      --group-add "${DOCKER_GID}" \
      -p 127.0.0.1:9005:8080 \
      --dns 8.8.8.8 --dns 1.1.1.1 \
      --env-file "${ENV_FILE}" \
      -v /var/run/docker.sock:/var/run/docker.sock \
      -v /opt/flowdeploy:/opt/flowdeploy:ro \
      -v flowdeploy-data:/data/apps \
      -v /etc/nginx:/etc/nginx:ro \
      -v /etc/letsencrypt:/etc/letsencrypt:ro \
      --label-file "${LABELS_FILE}" \
      "${PREV_IMAGE_ID}" >&2 || echo "Rollback also failed; manual intervention required" >&2
  else
    echo "No previous image captured; cannot rollback. Manual intervention required." >&2
  fi
  exit 1
}

echo "===== Waiting for container to become healthy (timeout=${HEALTH_TIMEOUT_SECONDS}s) ====="
elapsed=0
while [ "${elapsed}" -lt "${HEALTH_TIMEOUT_SECONDS}" ]; do
  STATE="$(docker inspect --format '{{.State.Status}}' "${CONTAINER_NAME}" 2>/dev/null || echo unknown)"
  if [ "${STATE}" != "running" ]; then
    rollback "Container state: ${STATE} after ${elapsed}s"
  fi
  HEALTH="$(docker inspect --format '{{if .State.Health}}{{.State.Health.Status}}{{else}}none{{end}}' "${CONTAINER_NAME}" 2>/dev/null || echo unknown)"
  case "${HEALTH}" in
    healthy)
      echo "Health: healthy after ${elapsed}s"
      break
      ;;
    unhealthy)
      rollback "Container reported unhealthy after ${elapsed}s"
      ;;
    none)
      if curl -fsS -m 2 -o /dev/null http://127.0.0.1:9005/health; then
        echo "HTTP /health OK after ${elapsed}s (no docker HEALTHCHECK declared)"
        break
      fi
      ;;
  esac
  sleep 3
  elapsed=$((elapsed + 3))
done

if [ "${elapsed}" -ge "${HEALTH_TIMEOUT_SECONDS}" ]; then
  rollback "container did not become healthy within ${HEALTH_TIMEOUT_SECONDS}s"
fi

echo "===== Container status: $(docker ps --filter "name=^${CONTAINER_NAME}$" --format '{{.Status}}') ====="
docker logs --tail 30 "${CONTAINER_NAME}" || true

echo "===== Deployment completed ====="
