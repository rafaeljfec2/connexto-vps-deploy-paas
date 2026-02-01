#!/bin/bash
set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
NETWORK_NAME="paasdeploy"

echo "=== PaaS Deploy - Starting ==="

if [ ! -f "$SCRIPT_DIR/.env" ]; then
    echo "Creating .env from .env.example..."
    cp "$SCRIPT_DIR/.env.example" "$SCRIPT_DIR/.env"
fi

if ! docker network inspect "$NETWORK_NAME" >/dev/null 2>&1; then
    echo "Creating Docker network: $NETWORK_NAME"
    docker network create "$NETWORK_NAME"
else
    echo "Docker network '$NETWORK_NAME' already exists"
fi

echo "Starting services with docker-compose..."
docker compose --env-file "$SCRIPT_DIR/.env" up -d --build

echo ""
echo "=== PaaS Deploy started successfully ==="
echo "Frontend: http://localhost"
echo "API: http://localhost/api"
