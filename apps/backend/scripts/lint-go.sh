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
    echo "golangci-lint not found. Install with: GOBIN=$ROOT/bin go install github.com/golangci/golangci-lint/cmd/golangci-lint@v1.61.0"
    echo "Or run in CI (lint runs there with Go 1.24)."
    exit 1
  fi
  local err
  err=$(mktemp)
  if ! $lint_cmd run --timeout=5m --concurrency=4 ./... 2>"$err"; then
    if grep -q "lower than the targeted Go version" "$err" 2>/dev/null; then
      echo ""
      echo "Seu golangci-lint foi compilado com Go 1.23; o projeto usa Go 1.24."
      echo "O lint roda no CI com Go 1.24. Para rodar local com Docker: pnpm run lint:go -- --docker"
    fi
    rm -f "$err"
    exit 3
  fi
  rm -f "$err"
}

LINT_IMAGE="paasdeploy-golangci-lint:1.61.0"

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
