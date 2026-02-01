#!/bin/bash
set -e

NETWORK_NAME="paasdeploy"

echo "=== PaaS Deploy - Starting ==="

if ! docker network inspect "$NETWORK_NAME" >/dev/null 2>&1; then
    echo "Creating Docker network: $NETWORK_NAME"
    docker network create "$NETWORK_NAME"
else
    echo "Docker network '$NETWORK_NAME' already exists"
fi

echo "Starting services with docker-compose..."
docker compose up -d --build

echo ""
echo "=== PaaS Deploy started successfully ==="
echo "Frontend: http://localhost"
echo "API: http://localhost/api"
