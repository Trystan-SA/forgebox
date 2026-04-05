#!/usr/bin/env bash
# build-rootfs.sh — Build a minimal Alpine-based ext4 rootfs for Firecracker VMs
# ================================================================================
# Requires: root, qemu-img (or truncate), mkfs.ext4, mount
# Produces: rootfs/base/rootfs.ext4

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"

ROOTFS_DIR="${PROJECT_ROOT}/rootfs/base"
ROOTFS_IMG="${ROOTFS_DIR}/rootfs.ext4"
MOUNT_DIR=$(mktemp -d /tmp/forgebox-rootfs.XXXXXX)
AGENT_BIN="${PROJECT_ROOT}/bin/fb-agent"

ROOTFS_SIZE_MB="${ROOTFS_SIZE_MB:-2048}"
ALPINE_VERSION="${ALPINE_VERSION:-3.19}"
ALPINE_MIRROR="${ALPINE_MIRROR:-https://dl-cdn.alpinelinux.org/alpine}"
ARCH="${ARCH:-x86_64}"

cleanup() {
    echo "==> Cleaning up..."
    umount -lf "${MOUNT_DIR}" 2>/dev/null || true
    rm -rf "${MOUNT_DIR}"
}
trap cleanup EXIT

if [ "$(id -u)" -ne 0 ]; then
    echo "ERROR: This script must be run as root (need mount/chroot)."
    exit 1
fi

if [ ! -f "${AGENT_BIN}" ]; then
    echo "ERROR: fb-agent binary not found at ${AGENT_BIN}"
    echo "       Run 'make build-agent' first."
    exit 1
fi

echo "==> Creating ${ROOTFS_SIZE_MB}MB ext4 image at ${ROOTFS_IMG}"
mkdir -p "${ROOTFS_DIR}"
truncate -s "${ROOTFS_SIZE_MB}M" "${ROOTFS_IMG}"
mkfs.ext4 -F -L forgebox-rootfs "${ROOTFS_IMG}"

echo "==> Mounting image"
mount -o loop "${ROOTFS_IMG}" "${MOUNT_DIR}"

echo "==> Bootstrapping Alpine Linux ${ALPINE_VERSION}"
ALPINE_KEYS_URL="${ALPINE_MIRROR}/v${ALPINE_VERSION}/main/${ARCH}"

# Download and extract Alpine minirootfs
MINIROOTFS_URL="${ALPINE_MIRROR}/v${ALPINE_VERSION}/releases/${ARCH}/alpine-minirootfs-${ALPINE_VERSION}.0-${ARCH}.tar.gz"
echo "    Downloading ${MINIROOTFS_URL}"
curl -fsSL "${MINIROOTFS_URL}" | tar -xz -C "${MOUNT_DIR}"

# Configure Alpine repositories
cat > "${MOUNT_DIR}/etc/apk/repositories" <<EOF
${ALPINE_MIRROR}/v${ALPINE_VERSION}/main
${ALPINE_MIRROR}/v${ALPINE_VERSION}/community
EOF

# Install packages inside chroot
echo "==> Installing packages"
chroot "${MOUNT_DIR}" /bin/sh -c "
    apk update && apk add --no-cache \
        bash \
        curl \
        wget \
        git \
        openssh-client \
        python3 \
        py3-pip \
        nodejs \
        npm \
        ripgrep \
        jq \
        ca-certificates \
        tzdata \
        shadow \
        sudo
"

# Create forgebox user inside the VM
echo "==> Creating forgebox user"
chroot "${MOUNT_DIR}" /bin/sh -c "
    addgroup -S forgebox && \
    adduser -S -G forgebox -h /home/forgebox -s /bin/bash forgebox && \
    echo 'forgebox ALL=(ALL) NOPASSWD:ALL' >> /etc/sudoers.d/forgebox
"

# Install fb-agent
echo "==> Installing fb-agent"
cp "${AGENT_BIN}" "${MOUNT_DIR}/usr/local/bin/fb-agent"
chmod 755 "${MOUNT_DIR}/usr/local/bin/fb-agent"

# Configure init (simple /init script for Firecracker)
echo "==> Writing /init"
cat > "${MOUNT_DIR}/init" <<'INITEOF'
#!/bin/bash
# ForgeBox VM init — started by Firecracker as PID 1

mount -t proc proc /proc
mount -t sysfs sysfs /sys
mount -t devtmpfs devtmpfs /dev

# Set hostname
hostname forgebox-vm

# Bring up loopback
ip link set lo up

# Bring up eth0 via DHCP if present
if ip link show eth0 &>/dev/null; then
    ip link set eth0 up
    udhcpc -i eth0 -q -s /etc/udhcpc/default.script 2>/dev/null || true
fi

# Start the agent — it communicates with the host via vsock
exec /usr/local/bin/fb-agent
INITEOF
chmod 755 "${MOUNT_DIR}/init"

# Ensure /workspace exists for task execution
mkdir -p "${MOUNT_DIR}/workspace"
chown 1000:1000 "${MOUNT_DIR}/workspace"

echo "==> Unmounting"
umount "${MOUNT_DIR}"

echo "==> rootfs built successfully: ${ROOTFS_IMG}"
ls -lh "${ROOTFS_IMG}"
