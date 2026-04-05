#!/usr/bin/env bash
# setup-kvm.sh — Check and configure KVM access on Linux
# ========================================================
set -euo pipefail

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
NC='\033[0m'

ok()   { echo -e "${GREEN}[OK]${NC}   $1"; }
warn() { echo -e "${YELLOW}[WARN]${NC} $1"; }
fail() { echo -e "${RED}[FAIL]${NC} $1"; }

echo "==> Checking KVM support for ForgeBox"
echo ""

# 1. Check CPU virtualization
if grep -Eqc '(vmx|svm)' /proc/cpuinfo 2>/dev/null; then
    ok "CPU supports hardware virtualization"
else
    fail "CPU does not support hardware virtualization (vmx/svm not found in /proc/cpuinfo)"
    exit 1
fi

# 2. Check /dev/kvm exists
if [ -e /dev/kvm ]; then
    ok "/dev/kvm exists"
else
    fail "/dev/kvm not found"
    echo "    Try: sudo modprobe kvm && sudo modprobe kvm_intel  (or kvm_amd)"
    exit 1
fi

# 3. Check /dev/kvm is readable/writable by current user
if [ -r /dev/kvm ] && [ -w /dev/kvm ]; then
    ok "/dev/kvm is accessible by $(whoami)"
else
    warn "/dev/kvm is not accessible by $(whoami)"
    KVM_GROUP=$(stat -c '%G' /dev/kvm 2>/dev/null || echo "kvm")
    echo "    Fix: sudo usermod -aG ${KVM_GROUP} $(whoami) && newgrp ${KVM_GROUP}"

    if [ "$(id -u)" -eq 0 ]; then
        echo "    (Running as root — adding current SUDO_USER to kvm group)"
        if [ -n "${SUDO_USER:-}" ]; then
            usermod -aG "${KVM_GROUP}" "${SUDO_USER}"
            ok "Added ${SUDO_USER} to ${KVM_GROUP} group (log out and back in to apply)"
        fi
    fi
fi

# 4. Check nested virtualization (useful for running inside a VM)
NESTED_FILE=""
if [ -f /sys/module/kvm_intel/parameters/nested ]; then
    NESTED_FILE=/sys/module/kvm_intel/parameters/nested
elif [ -f /sys/module/kvm_amd/parameters/nested ]; then
    NESTED_FILE=/sys/module/kvm_amd/parameters/nested
fi

if [ -n "${NESTED_FILE}" ]; then
    NESTED=$(cat "${NESTED_FILE}")
    if [ "${NESTED}" = "Y" ] || [ "${NESTED}" = "1" ]; then
        ok "Nested virtualization is enabled"
    else
        warn "Nested virtualization is disabled (may be needed if running inside a VM)"
    fi
fi

echo ""
echo "==> KVM check complete."
