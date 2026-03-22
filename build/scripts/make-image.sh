#!/bin/sh
# ═══════════════════════════════════════════════════════════════════
# build/scripts/make-image.sh — Build ScoutOS disk image
# ═══════════════════════════════════════════════════════════════════
#
# Runs INSIDE the Vagrant Alpine VM where apk, mksquashfs, sgdisk,
# and grub tools work natively. No ISO extraction or Docker needed.
#
# Usage (from host):
#   vagrant ssh utm -- sudo /opt/scout-build/scripts/make-image.sh
#   vagrant ssh utm -- sudo NO_VERITY=1 /opt/scout-build/scripts/make-image.sh
#   vagrant ssh utm -- sudo /opt/scout-build/scripts/make-image.sh 1.0.0
#
# Shared folder: host build/ → VM /opt/scout-build/
# Output:        /opt/scout-build/output/scoutos-<version>.img.zst
#
set -eu

# ──────────────────────────────────────────────
# Configuration
# ──────────────────────────────────────────────
BUILD="/opt/scout-build"
ROOTFS="/tmp/scout-rootfs"
WORK="/tmp/scout-work"
INITRAMFS_STAGE="/tmp/scout-initramfs"
OUTPUT="$BUILD/output"

VERSION="${1:-0.1.0}"
NO_VERITY="${NO_VERITY:-}"

# Alpine version for the TARGET rootfs (may differ from build VM)
ALPINE_TARGET="3.23"

# Partition sizes (MiB) — matches PLAN.md §7.1
ESP_SIZE=256
ROOTFS_A_SIZE=640
ROOTFS_B_SIZE=640
CONFIG_SIZE=32
DATA_SIZE=256
IMAGE_SIZE=$((ESP_SIZE + ROOTFS_A_SIZE + ROOTFS_B_SIZE + CONFIG_SIZE + DATA_SIZE + 2))

# Runtime state
LOOP_DEV=""
ESP_MNT=""
CONFIG_MNT=""
DATA_MNT=""

# ──────────────────────────────────────────────
# Helpers
# ──────────────────────────────────────────────
log()  { echo "==> $*"; }
log2() { echo "    $*"; }
die()  { echo "ERROR: $*" >&2; exit 1; }

cleanup() {
    rc=$?
    set +e
    echo ""
    [ $rc -ne 0 ] && echo "==> Build FAILED (exit code $rc)"
    echo "==> Cleaning up..."

    # Unmount image partitions
    [ -n "$DATA_MNT" ]   && umount -l "$DATA_MNT"   2>/dev/null && rmdir "$DATA_MNT" 2>/dev/null
    [ -n "$CONFIG_MNT" ] && umount -l "$CONFIG_MNT" 2>/dev/null && rmdir "$CONFIG_MNT" 2>/dev/null
    [ -n "$ESP_MNT" ]    && umount -l "$ESP_MNT"    2>/dev/null && rmdir "$ESP_MNT" 2>/dev/null

    # Unmount chroot
    umount -l "$ROOTFS/dev/pts" 2>/dev/null
    umount -l "$ROOTFS/dev"     2>/dev/null
    umount -l "$ROOTFS/sys"     2>/dev/null
    umount -l "$ROOTFS/proc"    2>/dev/null

    # Detach loop device
    [ -n "$LOOP_DEV" ] && losetup -d "$LOOP_DEV" 2>/dev/null

    # Remove temp dirs
    rm -rf "$ROOTFS" "$WORK" "$INITRAMFS_STAGE" 2>/dev/null
}
trap cleanup EXIT

# ──────────────────────────────────────────────
# Phase 0: Install build tools
# ──────────────────────────────────────────────
install_build_deps() {
    log "Installing build tools..."
    apk add --no-cache \
        squashfs-tools \
        e2fsprogs \
        dosfstools \
        sgdisk \
        grub-efi \
        cpio \
        gzip \
        zstd \
        util-linux \
        losetup \
        cryptsetup 2>/dev/null || true  # cryptsetup for veritysetup

    # Verify critical tools
    for cmd in mksquashfs sgdisk mkfs.vfat mkfs.ext4 grub-mkstandalone cpio losetup; do
        command -v "$cmd" >/dev/null 2>&1 || die "Missing: $cmd (apk add failed?)"
    done

    if [ -z "$NO_VERITY" ]; then
        command -v veritysetup >/dev/null 2>&1 || die "Missing: veritysetup (apk add cryptsetup)"
    fi
}

# ──────────────────────────────────────────────
# Phase 1: Create Alpine rootfs
# ──────────────────────────────────────────────
create_rootfs() {
    log "Creating Alpine $ALPINE_TARGET rootfs..."

    rm -rf "$ROOTFS"
    mkdir -p "$ROOTFS"

    # Bootstrap base system from online repos (no ISO needed — we're on Alpine)
    log2 "Installing alpine-base..."
    apk --root "$ROOTFS" --initdb \
        --repository "https://dl-cdn.alpinelinux.org/alpine/v${ALPINE_TARGET}/main" \
        --repository "https://dl-cdn.alpinelinux.org/alpine/v${ALPINE_TARGET}/community" \
        --no-cache --allow-untrusted \
        add alpine-base

    # Configure repos inside the new rootfs
    mkdir -p "$ROOTFS/etc/apk"
    cat > "$ROOTFS/etc/apk/repositories" <<EOF
https://dl-cdn.alpinelinux.org/alpine/v${ALPINE_TARGET}/main
https://dl-cdn.alpinelinux.org/alpine/v${ALPINE_TARGET}/community
EOF

    # DNS for network access during package installation
    cp /etc/resolv.conf "$ROOTFS/etc/resolv.conf"

    # Mount for chroot
    mkdir -p "$ROOTFS"/{proc,sys,dev/pts}
    mount -t proc     proc     "$ROOTFS/proc"
    mount -t sysfs    sysfs    "$ROOTFS/sys"
    mount --bind      /dev     "$ROOTFS/dev"
    mount --bind      /dev/pts "$ROOTFS/dev/pts"

    # Install all ScoutOS packages
    log2 "Installing packages (this may take a few minutes)..."
    chroot "$ROOTFS" apk update
    packages=$(grep -v '^#' "$BUILD/packages.txt" | grep -v '^\s*$' | tr '\n' ' ')
    chroot "$ROOTFS" apk add --allow-untrusted $packages

    # Install scoutd binary
    log2 "Installing scoutd..."
    [ -f "$BUILD/scoutd" ] || die "scoutd binary not found at $BUILD/scoutd"
    cp "$BUILD/scoutd" "$ROOTFS/usr/sbin/scoutd"
    chmod 755 "$ROOTFS/usr/sbin/scoutd"

    # Install daemon config templates
    mkdir -p "$ROOTFS/etc/scout/templates"
    cp "$BUILD/templates/"*.tmpl "$ROOTFS/etc/scout/templates/"

    log2 "Rootfs created"
}

# ──────────────────────────────────────────────
# Phase 2: Install ScoutOS overlay
# ──────────────────────────────────────────────
install_overlay() {
    log "Installing ScoutOS overlay..."

    # OpenRC init script
    cp "$BUILD/overlay/etc/init.d/scoutd" "$ROOTFS/etc/init.d/scoutd"
    chmod 755 "$ROOTFS/etc/init.d/scoutd"

    # RAUC config + post-install hook
    mkdir -p "$ROOTFS/etc/rauc"
    cp "$BUILD/overlay/etc/rauc/system.conf"    "$ROOTFS/etc/rauc/system.conf"
    cp "$BUILD/overlay/etc/rauc/post-install.sh" "$ROOTFS/etc/rauc/post-install.sh"
    chmod 755 "$ROOTFS/etc/rauc/post-install.sh"

    # Enable services
    for svc in scoutd nftables dnsmasq unbound chronyd; do
        chroot "$ROOTFS" rc-update add "$svc" default 2>/dev/null || \
            log2 "Warning: could not enable $svc"
    done

    # System config
    echo "scoututm" > "$ROOTFS/etc/hostname"
    echo "$VERSION" > "$ROOTFS/etc/scout-version"

    # Sysctl
    mkdir -p "$ROOTFS/etc/sysctl.d"
    cat > "$ROOTFS/etc/sysctl.d/scout.conf" <<'EOF'
net.ipv4.ip_forward = 1
net.ipv6.conf.all.disable_ipv6 = 1
net.ipv6.conf.default.disable_ipv6 = 1
EOF

    # First-boot data partition resize
    mkdir -p "$ROOTFS/etc/local.d"
    cat > "$ROOTFS/etc/local.d/01-resize-data.start" <<'RESIZEOF'
#!/bin/sh
MARKER=/etc/scout/.data-resized
[ -f "$MARKER" ] && exit 0
DATA_DEV=$(findfs LABEL=scout-data 2>/dev/null) || exit 0
case "$DATA_DEV" in
    *nvme*|*mmcblk*) DISK="${DATA_DEV%p[0-9]*}"; PARTNUM="${DATA_DEV##*p}" ;;
    *)               DISK="${DATA_DEV%[0-9]*}";   PARTNUM="${DATA_DEV##*[a-z]}" ;;
esac
[ -b "$DISK" ] && [ -n "$PARTNUM" ] || exit 0
logger -t scout "Expanding data partition ($DATA_DEV)..."
sgdisk -e "$DISK" 2>/dev/null
sgdisk -d "$PARTNUM" -n "${PARTNUM}:0:0" -t "${PARTNUM}:8300" -c "${PARTNUM}:data" "$DISK" 2>/dev/null
partprobe "$DISK" 2>/dev/null || true
sleep 1
resize2fs "$DATA_DEV" 2>/dev/null
touch "$MARKER"
logger -t scout "Data partition expanded"
RESIZEOF
    chmod 755 "$ROOTFS/etc/local.d/01-resize-data.start"
    chroot "$ROOTFS" rc-update add local default 2>/dev/null || true

    # WiFi/BT module blacklist (wired appliance only)
    mkdir -p "$ROOTFS/etc/modprobe.d"
    cat > "$ROOTFS/etc/modprobe.d/scout-blacklist.conf" <<'EOF'
blacklist iwlwifi
blacklist iwlmvm
blacklist mt7921e
blacklist mt7921_common
blacklist btusb
blacklist bluetooth
EOF

    # Mount points for persistent partitions
    mkdir -p "$ROOTFS/etc/scout" "$ROOTFS/var/lib/scout" "$ROOTFS/boot/efi"
    mkdir -p "$ROOTFS/var/lib/scout/suricata/rules"
    mkdir -p "$ROOTFS/var/lib/scout/logs"
    mkdir -p "$ROOTFS/var/lib/scout/blocklists"

    # Clean up
    rm -rf "$ROOTFS/var/cache/apk/"*
    rm -rf "$ROOTFS/usr/share/man" "$ROOTFS/usr/share/doc"
    rm -f  "$ROOTFS/etc/resolv.conf"

    log2 "Overlay installed"
}

# ──────────────────────────────────────────────
# Phase 3: Stage initramfs files (chroot still mounted)
# ──────────────────────────────────────────────
prepare_initramfs() {
    log "Staging initramfs files..."

    rm -rf "$INITRAMFS_STAGE"
    mkdir -p "$INITRAMFS_STAGE"/{bin,sbin,lib,dev,proc,sys,mnt,tmp,run,etc}
    mkdir -p "$INITRAMFS_STAGE"/mnt/{lower,upper,root}

    # BusyBox + musl
    cp "$ROOTFS/bin/busybox" "$INITRAMFS_STAGE/bin/busybox"
    chmod 755 "$INITRAMFS_STAGE/bin/busybox"
    cp "$ROOTFS"/lib/ld-musl-*.so.1 "$INITRAMFS_STAGE/lib/" 2>/dev/null || true

    # BusyBox applet symlinks
    for cmd in sh mount umount mkdir cat echo grep sleep reboot switch_root \
               modprobe insmod findfs blkid mknod ln ls cp mv rm; do
        ln -sf /bin/busybox "$INITRAMFS_STAGE/bin/$cmd"  2>/dev/null || true
        ln -sf /bin/busybox "$INITRAMFS_STAGE/sbin/$cmd" 2>/dev/null || true
    done

    # veritysetup + library dependencies
    if [ -f "$ROOTFS/sbin/veritysetup" ]; then
        cp "$ROOTFS/sbin/veritysetup" "$INITRAMFS_STAGE/sbin/"
        chmod 755 "$INITRAMFS_STAGE/sbin/veritysetup"
        log2 "Resolving veritysetup dependencies..."
        chroot "$ROOTFS" ldd /sbin/veritysetup 2>/dev/null | \
            awk '/=>/ {print $3}' | grep -v '^$' | sort -u | \
            while read lib; do
                if [ -f "$ROOTFS$lib" ]; then
                    mkdir -p "$INITRAMFS_STAGE$(dirname "$lib")"
                    cp "$ROOTFS$lib" "$INITRAMFS_STAGE$lib"
                fi
            done
    fi

    # blkid (util-linux version for PARTUUID support)
    if [ -f "$ROOTFS/sbin/blkid" ]; then
        cp "$ROOTFS/sbin/blkid" "$INITRAMFS_STAGE/sbin/blkid.real"
        chmod 755 "$INITRAMFS_STAGE/sbin/blkid.real"
        ln -sf /sbin/blkid.real "$INITRAMFS_STAGE/sbin/blkid"
        chroot "$ROOTFS" ldd /sbin/blkid 2>/dev/null | \
            awk '/=>/ {print $3}' | grep -v '^$' | sort -u | \
            while read lib; do
                if [ -f "$ROOTFS$lib" ]; then
                    mkdir -p "$INITRAMFS_STAGE$(dirname "$lib")"
                    cp "$ROOTFS$lib" "$INITRAMFS_STAGE$lib"
                fi
            done
    fi

    # findfs (util-linux)
    if [ -f "$ROOTFS/sbin/findfs" ]; then
        cp "$ROOTFS/sbin/findfs" "$INITRAMFS_STAGE/sbin/findfs.real"
        chmod 755 "$INITRAMFS_STAGE/sbin/findfs.real"
        ln -sf /sbin/findfs.real "$INITRAMFS_STAGE/sbin/findfs"
    fi

    # Kernel modules
    kver=$(ls "$ROOTFS/lib/modules/" 2>/dev/null | head -1)
    if [ -n "$kver" ]; then
        log2 "Copying kernel modules ($kver)..."
        moddir="$INITRAMFS_STAGE/lib/modules/$kver"
        mkdir -p "$moddir"

        for pattern in \
            kernel/fs/squashfs kernel/fs/overlayfs kernel/md \
            kernel/drivers/nvme kernel/fs/fat kernel/fs/nls \
            kernel/fs/ext4 kernel/fs/jbd2 kernel/lib/crc16 \
            kernel/fs/mbcache kernel/lib/crypto kernel/crypto; do
            src="$ROOTFS/lib/modules/$kver/$pattern"
            if [ -d "$src" ]; then
                mkdir -p "$(dirname "$moddir/$pattern")"
                cp -a "$src" "$(dirname "$moddir/$pattern")/"
            fi
        done

        # Module metadata
        for f in modules.dep modules.alias modules.symbols modules.order modules.builtin; do
            [ -f "$ROOTFS/lib/modules/$kver/$f" ] && \
                cp "$ROOTFS/lib/modules/$kver/$f" "$moddir/"
        done
    fi

    log2 "Initramfs files staged"
}

# ──────────────────────────────────────────────
# Phase 4: Unmount chroot
# ──────────────────────────────────────────────
unmount_chroot() {
    log "Unmounting chroot..."
    umount -l "$ROOTFS/dev/pts" 2>/dev/null || true
    umount -l "$ROOTFS/dev"     2>/dev/null || true
    umount -l "$ROOTFS/sys"     2>/dev/null || true
    umount -l "$ROOTFS/proc"    2>/dev/null || true
}

# ──────────────────────────────────────────────
# Phase 5: Build SquashFS + dm-verity
# ──────────────────────────────────────────────
build_squashfs() {
    log "Building SquashFS..."
    mkdir -p "$WORK"

    mksquashfs "$ROOTFS" "$WORK/scout-rootfs.sqfs" \
        -comp zstd -Xcompression-level 19 \
        -noappend -no-exports -no-recovery

    sqfs_size=$(stat -c %s "$WORK/scout-rootfs.sqfs")
    log2 "SquashFS size: $((sqfs_size / 1024 / 1024))MB"

    if [ -z "$NO_VERITY" ]; then
        log2 "Generating dm-verity hash tree..."
        veritysetup format \
            "$WORK/scout-rootfs.sqfs" "$WORK/scout-rootfs.sqfs" \
            --data-block-size=4096 --hash-offset="$sqfs_size" \
            | tee "$WORK/verity-output.txt"

        grep "Root hash:" "$WORK/verity-output.txt" | awk '{print $3}' > "$WORK/roothash.txt"
        echo "$sqfs_size" > "$WORK/hashoffset.txt"
        log2 "Root hash: $(cat "$WORK/roothash.txt")"
    else
        echo "" > "$WORK/roothash.txt"
        echo "0" > "$WORK/hashoffset.txt"
        log2 "dm-verity skipped (NO_VERITY=1)"
    fi

    # Free rootfs disk space (no longer needed)
    rm -rf "$ROOTFS"
}

# ──────────────────────────────────────────────
# Phase 6: Pack initramfs
# ──────────────────────────────────────────────
build_initramfs() {
    log "Building initramfs..."

    roothash=$(cat "$WORK/roothash.txt" 2>/dev/null || echo "")
    hashoffset=$(cat "$WORK/hashoffset.txt" 2>/dev/null || echo "0")

    sed \
        -e "s|@ROOTHASH@|$roothash|g" \
        -e "s|@HASHOFFSET@|$hashoffset|g" \
        "$BUILD/initramfs/init" > "$INITRAMFS_STAGE/init"
    chmod 755 "$INITRAMFS_STAGE/init"

    (cd "$INITRAMFS_STAGE" && find . | cpio -o -H newc 2>/dev/null | gzip) \
        > "$WORK/initramfs.img"

    size=$(stat -c %s "$WORK/initramfs.img")
    log2 "Initramfs size: $((size / 1024))KB"
}

# ──────────────────────────────────────────────
# Phase 7: Assemble GPT disk image
# ──────────────────────────────────────────────
assemble_image() {
    log "Assembling disk image..."

    img="$WORK/scoutos-${VERSION}.img"
    truncate -s "${IMAGE_SIZE}M" "$img"

    # GPT partition table
    sgdisk --clear \
        --new=1:0:+${ESP_SIZE}M       --typecode=1:EF00 --change-name=1:"ESP" \
        --new=2:0:+${ROOTFS_A_SIZE}M  --typecode=2:8300 --change-name=2:"rootfs-a" \
        --new=3:0:+${ROOTFS_B_SIZE}M  --typecode=3:8300 --change-name=3:"rootfs-b" \
        --new=4:0:+${CONFIG_SIZE}M    --typecode=4:8300 --change-name=4:"config" \
        --new=5:0:0                   --typecode=5:8300 --change-name=5:"data" \
        "$img"

    # Loop device
    LOOP_DEV=$(losetup --find --show --partscan "$img")
    partprobe "$LOOP_DEV" 2>/dev/null || blockdev --rereadpt "$LOOP_DEV" 2>/dev/null || true
    waited=0
    while [ ! -b "${LOOP_DEV}p1" ] && [ "$waited" -lt 10 ]; do
        sleep 1
        waited=$((waited + 1))
    done
    [ -b "${LOOP_DEV}p1" ] || die "Partition devices did not appear"

    # PARTUUIDs
    rootfs_a_partuuid=$(sgdisk -i 2 "$img" | grep "Partition unique GUID" | awk '{print tolower($NF)}')
    rootfs_b_partuuid=$(sgdisk -i 3 "$img" | grep "Partition unique GUID" | awk '{print tolower($NF)}')
    log2 "rootfs-a PARTUUID: $rootfs_a_partuuid"
    log2 "rootfs-b PARTUUID: $rootfs_b_partuuid"

    # Format partitions
    log2 "Formatting..."
    mkfs.vfat -F 32 -n SCOUT-ESP    "${LOOP_DEV}p1"
    mkfs.ext4 -q    -L scout-config "${LOOP_DEV}p4"
    mkfs.ext4 -q    -L scout-data   "${LOOP_DEV}p5"

    # Write SquashFS to rootfs-a
    log2 "Writing SquashFS to rootfs-a..."
    dd if="$WORK/scout-rootfs.sqfs" of="${LOOP_DEV}p2" bs=4M conv=fsync 2>&1 | tail -1

    # ── ESP setup ──
    ESP_MNT=$(mktemp -d)
    mount "${LOOP_DEV}p1" "$ESP_MNT"

    mkdir -p "$ESP_MNT/EFI/BOOT"
    mkdir -p "$ESP_MNT/EFI/scout/slot-a"
    mkdir -p "$ESP_MNT/EFI/scout/slot-b"

    # GRUB config with PARTUUIDs and try-counter logic
    cat > "$WORK/grub.cfg" <<GRUBEOF
set default=0
set timeout=3

set ORDER="A B"
set A_TRY=0
set B_TRY=0
set A_OK=0
set B_OK=0

load_env --file /EFI/scout/grubenv

if [ "\$ORDER" = "A B" ]; then
    if [ "\$A_TRY" != "0" ]; then
        set A_TRY=0
        save_env A_TRY
        set default=0
    elif [ "\$A_OK" = "1" ]; then
        set default=0
    elif [ "\$B_TRY" != "0" ]; then
        set B_TRY=0
        save_env B_TRY
        set default=1
    elif [ "\$B_OK" = "1" ]; then
        set default=1
    fi
elif [ "\$ORDER" = "B A" ]; then
    if [ "\$B_TRY" != "0" ]; then
        set B_TRY=0
        save_env B_TRY
        set default=1
    elif [ "\$B_OK" = "1" ]; then
        set default=1
    elif [ "\$A_TRY" != "0" ]; then
        set A_TRY=0
        save_env A_TRY
        set default=0
    elif [ "\$A_OK" = "1" ]; then
        set default=0
    fi
fi

menuentry "ScoutOS Slot A" --id slota {
    linux /EFI/scout/slot-a/vmlinuz root=PARTUUID=${rootfs_a_partuuid} rootfstype=squashfs ro rauc.slot=A net.ifnames=0 console=tty0 console=ttyS0,115200n8
    initrd /EFI/scout/slot-a/initramfs
}

menuentry "ScoutOS Slot B" --id slotb {
    linux /EFI/scout/slot-b/vmlinuz root=PARTUUID=${rootfs_b_partuuid} rootfstype=squashfs ro rauc.slot=B net.ifnames=0 console=tty0 console=ttyS0,115200n8
    initrd /EFI/scout/slot-b/initramfs
}

menuentry "ScoutOS Rescue (last known good)" --id rescue {
    if [ "\$A_OK" = "1" ]; then
        linux /EFI/scout/slot-a/vmlinuz root=PARTUUID=${rootfs_a_partuuid} rootfstype=squashfs ro rauc.slot=A scout.rescue=1 net.ifnames=0 console=tty0 console=ttyS0,115200n8
        initrd /EFI/scout/slot-a/initramfs
    elif [ "\$B_OK" = "1" ]; then
        linux /EFI/scout/slot-b/vmlinuz root=PARTUUID=${rootfs_b_partuuid} rootfstype=squashfs ro rauc.slot=B scout.rescue=1 net.ifnames=0 console=tty0 console=ttyS0,115200n8
        initrd /EFI/scout/slot-b/initramfs
    fi
}
GRUBEOF

    # Embedded GRUB config (chains to real grub.cfg)
    cat > "$WORK/grub-embed.cfg" <<'EMBEDEOF'
insmod part_gpt
insmod fat
search --set=root --file /EFI/scout/grub.cfg
set prefix=($root)/EFI/scout
configfile $prefix/grub.cfg
EMBEDEOF

    log2 "Building GRUB EFI binary..."
    grub-mkstandalone \
        --format=x86_64-efi \
        --output="$ESP_MNT/EFI/BOOT/BOOTX64.EFI" \
        --locales="" \
        --fonts="" \
        --modules="part_gpt fat normal linux echo all_video test configfile loadenv search" \
        "boot/grub/grub.cfg=$WORK/grub-embed.cfg"

    cp "$WORK/grub.cfg" "$ESP_MNT/EFI/scout/grub.cfg"

    # RAUC grubenv
    grub-editenv "$ESP_MNT/EFI/scout/grubenv" create
    grub-editenv "$ESP_MNT/EFI/scout/grubenv" set "ORDER=A B"
    grub-editenv "$ESP_MNT/EFI/scout/grubenv" set "A_TRY=0"
    grub-editenv "$ESP_MNT/EFI/scout/grubenv" set "B_TRY=0"
    grub-editenv "$ESP_MNT/EFI/scout/grubenv" set "A_OK=1"
    grub-editenv "$ESP_MNT/EFI/scout/grubenv" set "B_OK=0"

    # Kernel (saved to /tmp before rootfs was deleted)
    log2 "Copying kernel to ESP..."
    [ -f /tmp/scout-vmlinuz ] || die "Kernel not found at /tmp/scout-vmlinuz"
    cp /tmp/scout-vmlinuz "$ESP_MNT/EFI/scout/slot-a/vmlinuz"
    cp "$WORK/initramfs.img" "$ESP_MNT/EFI/scout/slot-a/initramfs"

    # ── Config partition ──
    log2 "Writing default config..."
    CONFIG_MNT=$(mktemp -d)
    mount "${LOOP_DEV}p4" "$CONFIG_MNT"
    cp "$BUILD/defaults/config.json" "$CONFIG_MNT/config.json"
    mkdir -p "$CONFIG_MNT/tls"
    chmod 700 "$CONFIG_MNT/tls"

    # ── Data partition ──
    log2 "Setting up data partition..."
    DATA_MNT=$(mktemp -d)
    mount "${LOOP_DEV}p5" "$DATA_MNT"
    mkdir -p "$DATA_MNT/suricata/rules" "$DATA_MNT/logs" "$DATA_MNT/blocklists"

    sync

    # Unmount
    umount "$DATA_MNT";   rmdir "$DATA_MNT";   DATA_MNT=""
    umount "$CONFIG_MNT"; rmdir "$CONFIG_MNT"; CONFIG_MNT=""
    umount "$ESP_MNT";    rmdir "$ESP_MNT";    ESP_MNT=""

    # Detach loop
    losetup -d "$LOOP_DEV"; LOOP_DEV=""

    log2 "Image assembled: $(du -h "$img" | cut -f1)"
}

# ──────────────────────────────────────────────
# Phase 8: Compress and copy to shared folder
# ──────────────────────────────────────────────
finalize() {
    log "Compressing image..."

    img="$WORK/scoutos-${VERSION}.img"
    mkdir -p "$OUTPUT"

    zstd -T0 -19 "$img" -o "$WORK/scoutos-${VERSION}.img.zst" --rm
    log2 "Compressed: $(du -h "$WORK/scoutos-${VERSION}.img.zst" | cut -f1)"

    # Copy to shared folder (accessible from host)
    cp "$WORK/scoutos-${VERSION}.img.zst" "$OUTPUT/"
    log2 "Output: $OUTPUT/scoutos-${VERSION}.img.zst"
}

# ──────────────────────────────────────────────
# Main
# ──────────────────────────────────────────────
main() {
    echo ""
    echo "╔══════════════════════════════════════╗"
    echo "║     ScoutOS Image Builder            ║"
    echo "╚══════════════════════════════════════╝"
    echo ""

    install_build_deps
    create_rootfs
    install_overlay
    prepare_initramfs

    # Save kernel before rootfs is deleted
    vmlinuz=$(ls "$ROOTFS"/boot/vmlinuz-lts 2>/dev/null || \
              ls "$ROOTFS"/boot/vmlinuz*     2>/dev/null | head -1)
    [ -n "$vmlinuz" ] || die "Kernel not found in rootfs"
    cp "$vmlinuz" "/tmp/scout-vmlinuz"

    unmount_chroot
    build_squashfs       # deletes rootfs after SquashFS is built
    build_initramfs
    assemble_image
    finalize

    rm -rf "$WORK" "$INITRAMFS_STAGE" /tmp/scout-vmlinuz

    echo ""
    echo "╔══════════════════════════════════════╗"
    echo "║     Build complete!                  ║"
    echo "╚══════════════════════════════════════╝"
    echo ""
    echo "  Image: $OUTPUT/scoutos-${VERSION}.img.zst"
    echo ""
    echo "  Test with VirtualBox (from host PowerShell):"
    echo "    .\\tools\\test-vm.ps1"
    echo ""
    echo "  Flash to hardware:"
    echo "    zstd -d scoutos-${VERSION}.img.zst --stdout | sudo dd of=/dev/sdX bs=4M"
    echo ""
}

main
