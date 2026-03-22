#!/bin/sh
# ================================================================
# ScoutUTM Hardware Install
# ================================================================
#
# Run this on a fresh Alpine Linux install on the GMKtec Mini PC.
#
# Prerequisites:
#   1. Boot GMKtec from Alpine USB (Rufus to write the ISO)
#   2. Run: setup-alpine  (install to NVMe, pick sda/nvme0n1, sys mode)
#   3. Reboot into installed Alpine
#   4. Copy this entire build/ folder to the machine:
#        scp -r build/ root@<ip>:/opt/scout-build/
#   5. SSH in and run:
#        sh /opt/scout-build/scripts/hardware-install.sh
#
set -e

BUILD="/opt/scout-build"

echo ""
echo "=== ScoutUTM Hardware Install ==="
echo ""

# -- Preflight checks --

if [ ! -f "$BUILD/scoutd" ]; then
  echo "ERROR: $BUILD/scoutd not found"
  echo "Copy the build/ folder to /opt/scout-build/ first."
  exit 1
fi

if ! ls "$BUILD/templates/"*.tmpl >/dev/null 2>&1; then
  echo "ERROR: templates not found in $BUILD/templates/"
  exit 1
fi

if [ ! -f "$BUILD/defaults/config.json" ]; then
  echo "ERROR: $BUILD/defaults/config.json not found"
  exit 1
fi

# -- System setup --

echo "Enabling IP forwarding..."
sysctl -w net.ipv4.ip_forward=1
mkdir -p /etc/sysctl.d
echo "net.ipv4.ip_forward = 1" > /etc/sysctl.d/99-scout.conf

# Ensure repos are configured
ALPINE_VER=$(cat /etc/alpine-release | cut -d. -f1,2)
cat > /etc/apk/repositories <<EOF
https://dl-cdn.alpinelinux.org/alpine/v${ALPINE_VER}/main
https://dl-cdn.alpinelinux.org/alpine/v${ALPINE_VER}/community
EOF

# -- Install packages --

echo "Installing packages..."
apk update
apk add \
  nftables \
  suricata \
  unbound \
  dnsmasq \
  wireguard-tools \
  chrony \
  iproute2 \
  conntrack-tools \
  openssl \
  curl \
  jq

# -- Install scoutd --

echo "Installing scoutd..."
mkdir -p /etc/scout/tls
mkdir -p /var/lib/scout/logs
mkdir -p /var/lib/scout/suricata
mkdir -p /var/lib/scout/blocklists
mkdir -p /var/log/scout
mkdir -p /usr/share/scoutd/templates

install -m 755 "$BUILD/scoutd" /usr/sbin/scoutd
cp "$BUILD/templates/"*.tmpl /usr/share/scoutd/templates/
echo "OK: scoutd + templates installed"

# -- Default config (only if none exists) --

if [ ! -f /etc/scout/config.json ]; then
  cp "$BUILD/defaults/config.json" /etc/scout/config.json
  echo "OK: default config installed"
else
  echo "OK: existing config preserved"
fi

# -- OpenRC service (production mode, no --dev-mode) --

cat > /etc/init.d/scoutd <<'INITEOF'
#!/sbin/openrc-run

name="scoutd"
description="ScoutUTM Management Daemon"
command="/usr/sbin/scoutd"
command_args="--config /etc/scout/config.json --template-dir /usr/share/scoutd/templates --output-dir /etc --tls-dir /etc/scout/tls --addr :443 --http-addr :80"
command_background=true
pidfile="/run/${RC_SVCNAME}.pid"
output_log="/var/log/scout/scoutd.log"
error_log="/var/log/scout/scoutd.log"

depend() {
    need net
    after firewall
}
INITEOF
chmod +x /etc/init.d/scoutd

# -- Enable services --

rc-update add nftables default 2>/dev/null || true
rc-update add dnsmasq default 2>/dev/null || true
rc-update add unbound default 2>/dev/null || true
rc-update add suricata default 2>/dev/null || true
rc-update add chronyd default 2>/dev/null || true
rc-update add scoutd default 2>/dev/null || true

# -- Prepare Unbound --

mkdir -p /var/lib/unbound
unbound-anchor -a /var/lib/unbound/root.key 2>/dev/null || true
chown -R unbound:unbound /var/lib/unbound
touch /var/lib/scout/logs/dns-query.log
chown unbound:unbound /var/lib/scout/logs 2>/dev/null || true
chown unbound:unbound /var/lib/scout/logs/dns-query.log 2>/dev/null || true

# Stub RPZ zones (real blocklists fetched by scheduler)
for zone in rpz-malware rpz-phishing rpz-ads rpz-parental-adult rpz-parental-gambling rpz-parental-gaming rpz-parental-social_media rpz-parental-streaming; do
  cat > "/var/lib/scout/blocklists/${zone}.zone" <<RPZEOF
\$SOA 0 0 IN SOA localhost. root.localhost. 1 86400 7200 2592000 86400
\$NS 0 0 IN NS localhost.
RPZEOF
done

# -- Prepare Suricata --

mkdir -p /var/run/suricata
mkdir -p /var/lib/suricata/rules
mkdir -p /var/lib/suricata/update
echo "Downloading Suricata rules..."
suricata-update --no-test --no-reload 2>/dev/null || echo "WARNING: suricata-update failed"

# -- nftables config --

mkdir -p /etc/conf.d
echo 'NFTABLES_CONFIG="/etc/nftables.conf"' > /etc/conf.d/nftables

# -- Start everything --

echo "Starting scoutd..."
rc-service scoutd start
sleep 3

if [ -f /etc/nftables.conf ]; then
  echo "OK: nftables.conf rendered"
  nft -f /etc/nftables.conf 2>/dev/null || echo "WARNING: nft load failed"
  rc-service nftables start 2>/dev/null || true
else
  echo "WARNING: /etc/nftables.conf not rendered"
  cat /var/log/scout/scoutd.log 2>/dev/null | tail -20
fi

rc-service chronyd start 2>/dev/null || true
rc-service dnsmasq start 2>/dev/null || true
rc-service unbound start || echo "WARNING: unbound failed to start"
rc-service suricata start 2>/dev/null || echo "WARNING: suricata failed to start"

# Reload nftables with scoutd-rendered config
nft -f /etc/nftables.conf 2>/dev/null || true

# -- Done --

echo ""
echo "=== Installation complete ==="
echo ""
echo "Services:"
echo "  scoutd:   $(rc-service scoutd status 2>/dev/null || echo 'not running')"
echo "  unbound:  $(rc-service unbound status 2>/dev/null || echo 'not running')"
echo "  suricata: $(rc-service suricata status 2>/dev/null || echo 'not running')"
echo "  chronyd:  $(rc-service chronyd status 2>/dev/null || echo 'not running')"
echo ""
echo "Ports:"
ss -tlnp 2>/dev/null | grep -E ':(80|443)\s' || echo "  (waiting for scoutd to bind)"
echo ""
echo "Web UI: https://<this-machine-ip>"
echo "Run the setup wizard to set your admin password."
echo ""
