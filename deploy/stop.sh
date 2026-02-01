#!/bin/bash
set -e

echo "=== PaaS Deploy - Stopping ==="

docker compose down

echo "=== PaaS Deploy stopped ==="
