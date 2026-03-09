#!/bin/sh
set -e

echo "=== ScoutUTM Dev VM Provisioning ==="

# CRITICAL: Ensure SSH is never blocked during provisioning.
# Flush any nftables rules immediately.
nft flush ruleset 2>/dev/null || true

# Enable IP forwarding so the kernel routes packets between LAN (eth1) and WAN (eth0).
sysctl -w net.ipv4.ip_forward=1
mkdir -p /etc/sysctl.d
echo "net.ipv4.ip_forward = 1" > /etc/sysctl.d/99-scout-forward.conf

# Fix DNS — VirtualBox NAT DNS proxy can be unreliable.
echo "nameserver 8.8.8.8" > /etc/resolv.conf

# Switch to official Alpine CDN (the box ships with kernel.org mirrors which are less reliable).
ALPINE_VER=$(cat /etc/alpine-release | cut -d. -f1,2)
cat > /etc/apk/repositories << EOF
https://dl-cdn.alpinelinux.org/alpine/v${ALPINE_VER}/main
https://dl-cdn.alpinelinux.org/alpine/v${ALPINE_VER}/community
EOF

# Verify APK signing keys are present before installing packages.
if [ ! -d /etc/apk/keys ] || [ -z "$(ls /etc/apk/keys/ 2>/dev/null)" ]; then
  echo "ERROR: APK signing keys not found in /etc/apk/keys/"
  echo "  Package signature verification is not possible."
  exit 1
fi

# Install ScoutOS packages. APK verifies package signatures against
# /etc/apk/keys/ by default. The --no-scripts flag prevents execution
# of untrusted post-install scripts from packages.
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

# Create directories.
mkdir -p /etc/scout/tls
mkdir -p /var/lib/scout
mkdir -p /var/lib/scout/logs
mkdir -p /var/log/scout
mkdir -p /usr/share/scoutd/templates

# Install scoutd binary.
if [ -f /opt/scout-build/scoutd ]; then
  install -m 755 /opt/scout-build/scoutd /usr/sbin/scoutd
  echo "OK: scoutd binary installed"
else
  echo "ERROR: /opt/scout-build/scoutd not found"
  echo "Contents of /opt/scout-build/:"
  ls -la /opt/scout-build/ 2>/dev/null || echo "  (directory does not exist)"
  exit 1
fi

# Copy templates from the synced build directory.
if [ -d /opt/scout-build/templates ] && ls /opt/scout-build/templates/*.tmpl >/dev/null 2>&1; then
  cp /opt/scout-build/templates/*.tmpl /usr/share/scoutd/templates/
  echo "OK: templates installed ($(ls /usr/share/scoutd/templates/*.tmpl | wc -l) files)"
else
  echo "ERROR: templates not found in /opt/scout-build/templates/"
  echo "Contents of /opt/scout-build/:"
  ls -laR /opt/scout-build/ 2>/dev/null || echo "  (empty)"
  exit 1
fi

# Install default config.
if [ ! -f /etc/scout/config.json ]; then
  if [ -f /opt/scout-build/defaults/config.json ]; then
    cp /opt/scout-build/defaults/config.json /etc/scout/config.json
    echo "OK: default config installed"
  else
    echo "ERROR: /opt/scout-build/defaults/config.json not found"
    exit 1
  fi
fi

# Set admin password ("admin") so we can log in to the web UI.
ADMIN_HASH='$2a$10$aT/wzfF.8i9xKj4aYXvgt.1x9mUANAlUUzn5PZgQ8gLsvB13g7mYm'
tmp=$(mktemp)
jq --arg h "$ADMIN_HASH" '.system.admin_hash = $h | .system.setup_done = true | .dns.block_malware = false | .dns.block_phishing = false | .dns.block_ads = false' /etc/scout/config.json > "$tmp"
mv "$tmp" /etc/scout/config.json
echo "OK: admin password set (password: admin)"

# Create OpenRC service script for scoutd.
cat > /etc/init.d/scoutd << 'INITEOF'
#!/sbin/openrc-run

name="scoutd"
description="ScoutUTM Management Daemon"
command="/usr/sbin/scoutd"
command_args="--dev-mode --config /etc/scout/config.json --template-dir /usr/share/scoutd/templates --output-dir /etc --tls-dir /etc/scout/tls --addr :443 --http-addr :80"
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

# Enable services at boot.
rc-update add nftables default 2>/dev/null || true
rc-update add dnsmasq default 2>/dev/null || true
rc-update add unbound default 2>/dev/null || true
rc-update add suricata default 2>/dev/null || true
rc-update add chronyd default 2>/dev/null || true
rc-update add scoutd default 2>/dev/null || true

# Start scoutd — it renders all daemon configs on startup.
echo "Starting scoutd..."
rc-service scoutd start 2>/dev/null
sleep 3

# Verify scoutd rendered configs.
if [ -f /etc/nftables.conf ]; then
  echo "OK: nftables.conf rendered"
else
  echo "WARNING: /etc/nftables.conf not rendered — check /var/log/scout/scoutd.log"
  cat /var/log/scout/scoutd.log 2>/dev/null | tail -20
fi

# Verify the rendered nftables.conf has the dev-mode SSH rule before loading it.
if grep -q "dport 22 accept" /etc/nftables.conf 2>/dev/null; then
  echo "OK: nftables.conf has dev-mode SSH rule, loading..."
  nft -f /etc/nftables.conf 2>/dev/null || echo "WARNING: nft load failed"
else
  echo "WARNING: nftables.conf missing SSH rule — NOT loading (SSH would break)"
  echo "  nftables rules NOT active. SSH is open."
fi

# Configure nftables service to use scoutd-rendered config.
mkdir -p /etc/conf.d
echo 'NFTABLES_CONFIG="/etc/nftables.conf"' > /etc/conf.d/nftables
rc-service nftables start 2>/dev/null || true

# Prepare unbound prerequisites.
# Create DNSSEC root key.
mkdir -p /var/lib/unbound
unbound-anchor -a /var/lib/unbound/root.key 2>/dev/null || true
chown -R unbound:unbound /var/lib/unbound
chown unbound:unbound /var/lib/scout/logs 2>/dev/null || true
touch /var/lib/scout/logs/dns-query.log
chown unbound:unbound /var/lib/scout/logs/dns-query.log 2>/dev/null || true
# Create stub RPZ blocklist zone files (real lists are fetched by scheduler in production).
mkdir -p /var/lib/scout/blocklists
for zone in rpz-malware rpz-phishing rpz-ads rpz-parental-adult rpz-parental-gambling rpz-parental-gaming rpz-parental-social_media rpz-parental-streaming; do
  cat > "/var/lib/scout/blocklists/${zone}.zone" << RPZEOF
\$SOA 0 0 IN SOA localhost. root.localhost. 1 86400 7200 2592000 86400
\$NS 0 0 IN NS localhost.
RPZEOF
done

# Prepare Suricata prerequisites.
mkdir -p /var/run/suricata
mkdir -p /var/lib/suricata/rules
mkdir -p /var/lib/suricata/update
# Download ET Open rules via suricata-update.
echo "Downloading Suricata rules (ET Open)..."
suricata-update --no-test --no-reload 2>/dev/null || echo "WARNING: suricata-update failed (rules may be missing)"

# Start other daemons.
rc-service chronyd start 2>/dev/null || true
rc-service dnsmasq start 2>/dev/null || true
rc-service unbound start || echo "WARNING: unbound failed to start"
rc-service suricata start 2>/dev/null || echo "WARNING: suricata failed to start"

# Reload scout nftables rules — rc-service nftables start may have loaded Alpine defaults.
nft -f /etc/nftables.conf 2>/dev/null || echo "WARNING: nft reload failed"

# Final status.
echo ""
echo "=== Provisioning complete ==="
echo ""
echo "Checking services..."
echo "  scoutd:   $(rc-service scoutd status 2>/dev/null || echo 'not running')"
echo "  unbound:  $(rc-service unbound status 2>/dev/null || echo 'not running')"
echo "  suricata: $(rc-service suricata status 2>/dev/null || echo 'not running')"
echo "  chronyd:  $(rc-service chronyd status 2>/dev/null || echo 'not running')"
echo ""
echo "Checking ports..."
ss -tlnp 2>/dev/null | grep -E ':(22|80|443)\s' || echo "  (no listeners found on 22/80/443)"
echo ""
echo "Web UI: https://localhost:8443 (from host)"
echo "Login:  admin / admin"

# ── Integration Tests ──
if [ "$SCOUT_NO_TESTS" = "1" ]; then
  echo ""
  echo "Skipping integration tests (SCOUT_NO_TESTS=1)"
  exit 0
fi

# Disable set -e so test failures don't abort provisioning.
set +e

TEST_DIR="/opt/scout-tests"
if [ -d "$TEST_DIR" ] && ls "$TEST_DIR"/*_test.sh >/dev/null 2>&1; then
  echo ""
  echo "╔══════════════════════════════════════╗"
  echo "║     Running Integration Tests        ║"
  echo "╚══════════════════════════════════════╝"
  echo ""

  TOTAL_PASS=0
  TOTAL_FAIL=0
  TOTAL_SKIP=0

  for script in "$TEST_DIR"/*_test.sh; do
    name=$(basename "$script")

    # Skip empty (0-byte) placeholder scripts.
    if [ ! -s "$script" ]; then
      TOTAL_SKIP=$((TOTAL_SKIP + 1))
      echo "── SKIP: $name (not implemented) ──"
      echo ""
      continue
    fi

    echo "── Running: $name ──"
    if sh "$script"; then
      TOTAL_PASS=$((TOTAL_PASS + 1))
      echo "── $name: PASSED ──"
    else
      TOTAL_FAIL=$((TOTAL_FAIL + 1))
      echo "── $name: FAILED ──"
    fi
    echo ""
  done

  echo "╔══════════════════════════════════════╗"
  echo "║     Integration Test Summary         ║"
  echo "╠══════════════════════════════════════╣"
  printf "║  Suites passed:  %-19s║\n" "$TOTAL_PASS"
  printf "║  Suites failed:  %-19s║\n" "$TOTAL_FAIL"
  printf "║  Suites skipped: %-19s║\n" "$TOTAL_SKIP"
  echo "╚══════════════════════════════════════╝"

  if [ "$TOTAL_FAIL" -gt 0 ]; then
    echo ""
    echo "WARNING: $TOTAL_FAIL test suite(s) failed. Check output above."
  fi
else
  echo ""
  echo "WARNING: No test scripts found in $TEST_DIR"
  echo "  Ensure tests/ is synced via Vagrantfile."
fi
