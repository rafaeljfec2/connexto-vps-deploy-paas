#!/usr/bin/env bash
set -euo pipefail

REPO_ROOT="$(cd "$(dirname "$0")/.." && pwd)"
GIT="git -C $REPO_ROOT"

bump_patch() {
  local version="$1"
  local major minor patch
  IFS='.' read -r major minor patch <<< "$version"
  echo "${major}.${minor}.$((patch + 1))"
}

bump_package_json() {
  local file="$1"
  local current
  current=$(sed -n 's/.*"version": "\([0-9]*\.[0-9]*\.[0-9]*\)".*/\1/p' "$file" | head -1)
  if [ -z "$current" ]; then
    echo "  WARNING: could not read version from $file"
    return
  fi
  local next
  next=$(bump_patch "$current")
  sed -i "s/\"version\": \"${current}\"/\"version\": \"${next}\"/" "$file"
  echo "  $file: $current -> $next"
}

bump_version_file() {
  local file="$1"
  local current
  current=$(tr -d '[:space:]' < "$file")
  if [ -z "$current" ]; then
    echo "  WARNING: could not read version from $file"
    return
  fi
  local next
  next=$(bump_patch "$current")
  printf '%s\n' "$next" > "$file"
  echo "  $file: $current -> $next"
}

CHANGED_FILES=""
if $GIT diff --cached --name-only --diff-filter=d 2>/dev/null | grep -q .; then
  CHANGED_FILES=$($GIT diff --cached --name-only --diff-filter=d)
elif $GIT diff --name-only HEAD --diff-filter=d 2>/dev/null | grep -q .; then
  CHANGED_FILES=$($GIT diff --name-only HEAD --diff-filter=d)
fi

if [ -z "$CHANGED_FILES" ]; then
  echo "[bump-versions] No changed files detected, skipping."
  exit 0
fi

SKIP_PATTERNS="package.json AGENT_VERSION"
FILTERED_FILES=""
while IFS= read -r file; do
  [ -z "$file" ] && continue
  base=$(basename "$file")
  skip=false
  for pattern in $SKIP_PATTERNS; do
    if [ "$base" = "$pattern" ]; then
      skip=true
      break
    fi
  done
  if [ "$skip" = false ]; then
    FILTERED_FILES="${FILTERED_FILES}${file}
"
  fi
done <<< "$CHANGED_FILES"

if [ -z "$(echo "$FILTERED_FILES" | tr -d '[:space:]')" ]; then
  echo "[bump-versions] Only version files changed, skipping."
  exit 0
fi

BUMP_FRONTEND=false
BUMP_BACKEND=false
BUMP_AGENT=false
BUMP_ROOT=false

while IFS= read -r file; do
  [ -z "$file" ] && continue
  case "$file" in
    apps/frontend/*)            BUMP_FRONTEND=true ;;
    apps/backend/*)             BUMP_BACKEND=true ;;
    apps/agent/*)               BUMP_AGENT=true ;;
    apps/shared/*)              BUMP_ROOT=true ;;
    apps/proto/*)               BUMP_ROOT=true ;;
    paasdeploy.schema.json)     BUMP_ROOT=true ;;
    deploy/*)                   BUMP_ROOT=true ;;
    Dockerfile*)                BUMP_ROOT=true ;;
  esac
done <<< "$FILTERED_FILES"

BUMPED=false

if [ "$BUMP_FRONTEND" = true ]; then
  echo "[bump-versions] Bumping frontend..."
  bump_package_json "$REPO_ROOT/apps/frontend/package.json"
  $GIT add "apps/frontend/package.json" 2>/dev/null || true
  BUMPED=true
fi

if [ "$BUMP_BACKEND" = true ]; then
  echo "[bump-versions] Bumping backend..."
  bump_package_json "$REPO_ROOT/apps/backend/package.json"
  $GIT add "apps/backend/package.json" 2>/dev/null || true
  BUMPED=true
fi

if [ "$BUMP_AGENT" = true ]; then
  echo "[bump-versions] Bumping agent..."
  bump_version_file "$REPO_ROOT/AGENT_VERSION"
  $GIT add "AGENT_VERSION" 2>/dev/null || true
  BUMPED=true
fi

if [ "$BUMP_ROOT" = true ]; then
  echo "[bump-versions] Bumping root package..."
  bump_package_json "$REPO_ROOT/package.json"
  $GIT add "package.json" 2>/dev/null || true
  BUMPED=true
fi

if [ "$BUMPED" = false ]; then
  echo "[bump-versions] No versionable packages affected, skipping."
fi
