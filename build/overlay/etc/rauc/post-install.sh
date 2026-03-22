#!/bin/sh
# RAUC post-install hook: extract kernel + initramfs from SquashFS to ESP.
set -eu

SLOT="$1"
SLOT_DEV="$2"
SLOT_DIR=$(echo "$SLOT" | tr 'A-Z' 'a-z')
ESP="/boot/efi/EFI/scout"
MOUNT=$(mktemp -d)

mount -t squashfs -o ro "$SLOT_DEV" "$MOUNT"

# Extract to temp files first, then atomic rename (prevents partial writes on power loss).
cp "$MOUNT/boot/vmlinuz-lts" "${ESP}/slot-${SLOT_DIR}/vmlinuz.new"
cp "$MOUNT/boot/initramfs"   "${ESP}/slot-${SLOT_DIR}/initramfs.new"

# Verify extracted files are non-zero.
[ -s "${ESP}/slot-${SLOT_DIR}/vmlinuz.new" ]   || { echo "ERROR: vmlinuz extraction failed"; exit 1; }
[ -s "${ESP}/slot-${SLOT_DIR}/initramfs.new" ] || { echo "ERROR: initramfs extraction failed"; exit 1; }

# Atomic rename (as atomic as FAT32 allows).
mv "${ESP}/slot-${SLOT_DIR}/vmlinuz.new"   "${ESP}/slot-${SLOT_DIR}/vmlinuz"
mv "${ESP}/slot-${SLOT_DIR}/initramfs.new" "${ESP}/slot-${SLOT_DIR}/initramfs"
sync

umount "$MOUNT"
rmdir "$MOUNT"
