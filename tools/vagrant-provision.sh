#!/bin/sh
set -e

echo "=== ScoutUTM Dev VM Provisioning ==="

# Install ScoutOS packages.
apk update
apk add --no-cache \
  nftables \
  suricata \
  unbound \
  dnsmasq \
  wireguard-tools \
  chrony \
  curl \
  jq

# Create directories.
mkdir -p /etc/scout/tls
mkdir -p /var/lib/scout
mkdir -p /var/log/scout

# Install scoutd binary (if present in build artifacts).
if [ -f /opt/scout-build/scoutd ]; then
  cp /opt/scout-build/scoutd /usr/sbin/scoutd
  chmod +x /usr/sbin/scoutd
  echo "scoutd binary installed"
else
  echo "WARNING: scoutd binary not found in /opt/scout-build/"
fi

# Copy default config if no config exists.
if [ ! -f /etc/scout/config.json ]; then
  if [ -f /opt/scout-build/../build/defaults/config.json ]; then
    cp /opt/scout-build/../build/defaults/config.json /etc/scout/config.json
  fi
  echo "default config installed"
fi

# Create OpenRC service script for scoutd.
cat > /etc/init.d/scoutd << 'EOF'
#!/sbin/openrc-run

name="scoutd"
description="ScoutUTM Management Daemon"
command="/usr/sbin/scoutd"
command_args="--config /etc/scout/config.json"
command_background=true
pidfile="/run/${RC_SVCNAME}.pid"
output_log="/var/log/scout/scoutd.log"
error_log="/var/log/scout/scoutd.log"

depend() {
    need net
    after firewall
}
EOF
chmod +x /etc/init.d/scoutd

# Enable services.
rc-update add nftables default 2>/dev/null || true
rc-update add dnsmasq default 2>/dev/null || true
rc-update add unbound default 2>/dev/null || true
rc-update add chronyd default 2>/dev/null || true
rc-update add scoutd default 2>/dev/null || true

# Start services (best-effort — some may fail without config).
rc-service nftables start 2>/dev/null || echo "nftables: will start after config"
rc-service dnsmasq start 2>/dev/null || echo "dnsmasq: will start after config"
rc-service chronyd start 2>/dev/null || echo "chronyd: will start after config"

# Start scoutd if binary exists.
if [ -x /usr/sbin/scoutd ]; then
  rc-service scoutd start 2>/dev/null || echo "scoutd: will start after config"
fi

echo "=== Provisioning complete ==="
