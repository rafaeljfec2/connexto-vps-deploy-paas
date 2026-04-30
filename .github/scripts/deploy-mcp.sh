#!/usr/bin/env bash
# Local deploy script run by the self-hosted GitHub Actions runner on the
# control-plane VPS. Pulls the MCP image (pinned to the short SHA built in
# the same workflow run) from GHCR and recreates the `flowdeploy-mcp`
# container.
#
# This script does NOT use sudo. The runner user must:
#   - belong to the `docker` group (already true for `ubuntu`);
#   - be able to reach the backend container by its docker DNS name on the
#     `paasdeploy` network (default: http://flowdeploy-backend:8080).
#
# The MCP server is stateless: no env file, no secrets at rest, no Docker
# socket mount, no host port binding. Public traffic reaches it ONLY through
# Traefik on `PathPrefix(/mcp)`; /metrics, /healthz and /readyz remain on
# the internal docker network.
set -euo pipefail

REGISTRY="${REGISTRY:?REGISTRY is required}"
IMAGE_NAME="${IMAGE_NAME:?IMAGE_NAME is required}"
IMAGE_TAG="${IMAGE_TAG:?IMAGE_TAG is required (short SHA from build-and-push)}"
GHCR_USER="${GHCR_USER:?GHCR_USER is required}"
GHCR_PAT="${GHCR_PAT:?GHCR_PAT is required}"
CONTAINER_NAME="${CONTAINER_NAME:?CONTAINER_NAME is required}"
TRAEFIK_NETWORK="${TRAEFIK_NETWORK:?TRAEFIK_NETWORK is required}"
MCP_HOST="${MCP_HOST:?MCP_HOST is required}"
FLOWDEPLOY_BACKEND_URL="${FLOWDEPLOY_BACKEND_URL:?FLOWDEPLOY_BACKEND_URL is required}"

FLOWDEPLOY_MCP_LOG_LEVEL="${FLOWDEPLOY_MCP_LOG_LEVEL:-info}"
FLOWDEPLOY_MCP_ALLOWED_CLIENTS="${FLOWDEPLOY_MCP_ALLOWED_CLIENTS:-cursor,claude-desktop,custom:*,ci:*}"
FLOWDEPLOY_MCP_READ_RPM="${FLOWDEPLOY_MCP_READ_RPM:-120}"
FLOWDEPLOY_MCP_MUTATE_RPM="${FLOWDEPLOY_MCP_MUTATE_RPM:-20}"
FLOWDEPLOY_MCP_SESSION_MAX_AGE="${FLOWDEPLOY_MCP_SESSION_MAX_AGE:-30m}"

HEALTH_TIMEOUT_SECONDS="${HEALTH_TIMEOUT_SECONDS:-120}"

IMAGE_REF="${REGISTRY}/${IMAGE_NAME}:${IMAGE_TAG}"

# retry runs the given command up to ${1} times, with a fixed backoff between
# attempts. Used only for steps whose failure mode is transient network noise
# against ghcr.io (docker login, docker pull). Deterministic failures (bad
# image, bad env, container unhealthy) MUST NOT be wrapped — masking them
# would hide real bugs. Mirrored from .github/scripts/deploy.sh; if this
# helper grows, extract to a shared lib/ file source-able by both scripts.
retry() {
  local max="$1"; shift
  local backoffs=(5 15 30)
  local attempt=1
  while true; do
    if "$@"; then
      return 0
    fi
    if [ "${attempt}" -ge "${max}" ]; then
      echo "  -> attempt ${attempt}/${max} failed; giving up" >&2
      return 1
    fi
    local idx=$((attempt - 1))
    local sleep_for="${backoffs[${idx}]:-30}"
    echo "  -> attempt ${attempt}/${max} failed; retrying in ${sleep_for}s..." >&2
    sleep "${sleep_for}"
    attempt=$((attempt + 1))
  done
}

echo "===== Deploying ${IMAGE_REF} ====="
echo "  container         : ${CONTAINER_NAME}"
echo "  network           : ${TRAEFIK_NETWORK}"
echo "  public host       : ${MCP_HOST}"
echo "  backend (internal): ${FLOWDEPLOY_BACKEND_URL}"
echo "  read_rpm          : ${FLOWDEPLOY_MCP_READ_RPM}"
echo "  mutate_rpm        : ${FLOWDEPLOY_MCP_MUTATE_RPM}"
echo "  session_max_age   : ${FLOWDEPLOY_MCP_SESSION_MAX_AGE}"

echo "===== Login to ${REGISTRY} ====="
docker_login() {
  echo "${GHCR_PAT}" | docker login "${REGISTRY}" -u "${GHCR_USER}" --password-stdin >/dev/null
}
retry 3 docker_login

echo "===== Pull image ====="
retry 3 docker pull "${IMAGE_REF}"

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

# Public router: exposes ONLY /mcp via Traefik (HTTPS + Let's Encrypt).
# /metrics, /healthz and /readyz stay reachable only inside the paasdeploy
# network (Prometheus / docker healthcheck only).
cat > "${LABELS_FILE}" <<LABELS
traefik.enable=true
traefik.http.routers.${CONTAINER_NAME}.rule=Host(\`${MCP_HOST}\`) && PathPrefix(\`/mcp\`)
traefik.http.routers.${CONTAINER_NAME}.entrypoints=websecure
traefik.http.routers.${CONTAINER_NAME}.tls=true
traefik.http.routers.${CONTAINER_NAME}.tls.certresolver=letsencrypt
traefik.http.services.${CONTAINER_NAME}.loadbalancer.server.port=3001
traefik.http.middlewares.${CONTAINER_NAME}-headers.headers.customResponseHeaders.X-Content-Type-Options=nosniff
traefik.http.middlewares.${CONTAINER_NAME}-headers.headers.customResponseHeaders.Strict-Transport-Security=max-age=63072000; includeSubDomains
traefik.http.routers.${CONTAINER_NAME}.middlewares=${CONTAINER_NAME}-headers@docker
LABELS

run_container() {
  local image_ref="$1"
  docker run -d \
    --name "${CONTAINER_NAME}" \
    --network "${TRAEFIK_NETWORK}" \
    --restart unless-stopped \
    --dns 8.8.8.8 --dns 1.1.1.1 \
    -e FLOWDEPLOY_BACKEND_URL="${FLOWDEPLOY_BACKEND_URL}" \
    -e FLOWDEPLOY_LOG_LEVEL="${FLOWDEPLOY_MCP_LOG_LEVEL}" \
    --label-file "${LABELS_FILE}" \
    "${image_ref}" \
    serve \
    --addr=:3001 \
    --allowed-clients="${FLOWDEPLOY_MCP_ALLOWED_CLIENTS}" \
    --read-rpm="${FLOWDEPLOY_MCP_READ_RPM}" \
    --mutate-rpm="${FLOWDEPLOY_MCP_MUTATE_RPM}" \
    --session-max-age="${FLOWDEPLOY_MCP_SESSION_MAX_AGE}"
}

echo "===== Run new container ====="
run_container "${IMAGE_REF}"

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
    run_container "${PREV_IMAGE_ID}" >&2 || echo "Rollback also failed; manual intervention required" >&2
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
      # The MCP Dockerfile declares HEALTHCHECK on /healthz, so we should
      # always observe healthy/unhealthy; falling back to docker exec covers
      # the rare case where HEALTHCHECK is disabled at the engine level.
      if docker exec "${CONTAINER_NAME}" wget --no-verbose --tries=1 --timeout=3 -q -O- http://127.0.0.1:3001/healthz >/dev/null 2>&1; then
        echo "/healthz OK after ${elapsed}s (no docker HEALTHCHECK declared)"
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
