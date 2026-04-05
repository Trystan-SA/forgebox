#!/usr/bin/env bash
# Stop the ForgeBox dev environment
# Usage: ./scripts/dev-down.sh

set -euo pipefail
docker compose -f .devcontainer/docker-compose.yml down
echo "Dev environment stopped."
