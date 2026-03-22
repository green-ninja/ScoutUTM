#!/usr/bin/env bash
# ═══════════════════════════════════════════════════════════════════
# tools/build-image.sh — Build ScoutOS disk image via Vagrant VM
# ═══════════════════════════════════════════════════════════════════
#
# Host-side wrapper. The actual build runs inside the Vagrant Alpine
# VM where apk, mksquashfs, sgdisk, and grub work natively.
#
# Usage:
#   ./tools/build-image.sh                  # default (version 0.1.0)
#   ./tools/build-image.sh 1.0.0            # specific version
#   NO_VERITY=1 ./tools/build-image.sh      # skip dm-verity
#
# Output: build/output/scoutos-<version>.img.zst
#
set -euo pipefail

VERSION="${1:-0.1.0}"

echo ""
echo "==> ScoutOS Image Build"
echo ""

# Check scoutd binary
if [[ ! -f build/scoutd ]]; then
    echo "ERROR: build/scoutd not found."
    echo "  Build it first:  cd scoutd && /usr/local/go/bin/go build -o ../build/scoutd ./cmd/scoutd"
    exit 1
fi

# Ensure Vagrant VM is running
if ! vagrant status utm 2>/dev/null | grep -q running; then
    echo "==> Starting Vagrant VM..."
    vagrant up utm
fi

# Run the build inside the VM
echo "==> Running build inside Vagrant VM..."
vagrant ssh utm -- sudo ${NO_VERITY:+NO_VERITY=1} /opt/scout-build/scripts/make-image.sh "$VERSION"

echo ""
echo "==> Image ready: build/output/scoutos-${VERSION}.img.zst"
echo ""
echo "  Test:  .\\tools\\test-vm.ps1"
echo "  Flash: zstd -d build/output/scoutos-${VERSION}.img.zst --stdout | sudo dd of=/dev/sdX bs=4M"
echo ""
