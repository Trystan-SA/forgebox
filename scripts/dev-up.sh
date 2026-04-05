#!/usr/bin/env bash
# Start the full ForgeBox dev environment using devcontainers
# Usage: ./scripts/dev-up.sh

set -euo pipefail
echo "Starting ForgeBox development environment..."
docker compose -f .devcontainer/docker-compose.yml up -d --build
echo ""
echo "Environment ready. Open this folder in VS Code and use 'Reopen in Container'."
echo "Or attach directly:"
echo "  docker compose -f .devcontainer/docker-compose.yml exec forgebox bash"
echo ""
echo "Services:"
echo "  Backend API:   http://localhost:8420"
echo "  gRPC:          localhost:8421"
echo "  Dashboard:     http://localhost:3000"
echo "  PostgreSQL:    localhost:5432"
