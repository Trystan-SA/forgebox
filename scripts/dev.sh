#!/usr/bin/env bash
# dev.sh — Development launcher for ForgeBox
# =============================================
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
cd "${PROJECT_ROOT}"

# Load .env if present
if [ -f .env ]; then
    echo "==> Loading .env"
    set -a
    # shellcheck disable=SC1091
    source .env
    set +a
fi

# Check prerequisites
echo "==> Checking prerequisites..."
command -v go >/dev/null 2>&1 || { echo "ERROR: go is not installed"; exit 1; }

GO_VERSION=$(go version | grep -oP 'go\d+\.\d+')
echo "    Go: ${GO_VERSION}"

if [ -e /dev/kvm ]; then
    echo "    KVM: available"
else
    echo "    KVM: not found (VM features will not work)"
fi

# Build
echo "==> Building..."
make build

# Run
echo "==> Starting ForgeBox (dev mode)..."
CONFIG="${FORGEBOX_CONFIG:-forgebox.yaml}"
if [ ! -f "${CONFIG}" ]; then
    echo "    Config not found at ${CONFIG}, copying example..."
    cp forgebox.example.yaml "${CONFIG}"
fi

exec ./bin/forgebox serve --config "${CONFIG}"
