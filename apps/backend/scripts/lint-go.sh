#!/usr/bin/env bash
set -e
cd "$(dirname "$0")/.."
ROOT="$(pwd)"

run_local() {
  local lint_cmd=
  if command -v golangci-lint >/dev/null 2>&1; then
    lint_cmd="golangci-lint"
  elif [ -x "./bin/golangci-lint" ]; then
    lint_cmd="./bin/golangci-lint"
  else
    echo "golangci-lint not found."
    echo "Install with the project's Go toolchain: make install-lint"
    exit 1
  fi
  local err
  err=$(mktemp)
  if ! $lint_cmd run --timeout=5m --concurrency=4 ./... 2>"$err"; then
    if grep -q "lower than the targeted Go version" "$err" 2>/dev/null; then
      echo ""
      echo "Your golangci-lint was built with Go < 1.25; the project targets Go 1.25.5."
      echo "Rebuild locally: make install-lint"
      echo "Or run via Docker: pnpm run lint:go -- --docker"
    fi
    rm -f "$err"
    exit 3
  fi
  rm -f "$err"
}

LINT_IMAGE="paasdeploy-golangci-lint:2.12.2"

run_docker() {
  if ! docker image inspect "$LINT_IMAGE" >/dev/null 2>&1; then
    echo "Building lint image (one-time, ~1 min)..."
    docker build -t "$LINT_IMAGE" -f "$ROOT/scripts/Dockerfile.lint" "$ROOT/scripts"
  fi
  docker run --rm -v "$ROOT:/app" -w /app "$LINT_IMAGE" run --timeout=5m --concurrency=4 ./...
}

USE_DOCKER=
for a in "$@"; do [ "$a" = "--docker" ] && USE_DOCKER=1; done
if [ -n "$USE_DOCKER" ]; then
  run_docker
else
  run_local
fi
