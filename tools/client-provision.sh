#!/bin/sh
set -e

echo "=== ScoutUTM LAN Client Provisioning ==="

# Fix DNS — VirtualBox NAT DNS proxy can be unreliable.
echo "nameserver 8.8.8.8" > /etc/resolv.conf

# Switch to official Alpine CDN.
ALPINE_VER=$(cat /etc/alpine-release | cut -d. -f1,2)
cat > /etc/apk/repositories << EOF
https://dl-cdn.alpinelinux.org/alpine/v${ALPINE_VER}/main
https://dl-cdn.alpinelinux.org/alpine/v${ALPINE_VER}/community
EOF

# Install minimal tools.
apk update
apk add curl bind-tools

# ── Network Setup ──
# Bring up eth1 via DHCP from ScoutUTM's dnsmasq.
echo "Requesting DHCP lease on eth1..."
ip link set eth1 up
udhcpc -i eth1 -n -q 2>/dev/null || {
  echo "WARNING: DHCP on eth1 failed — ScoutUTM may not be ready yet"
  echo "Retrying in 5 seconds..."
  sleep 5
  udhcpc -i eth1 -n -q 2>/dev/null || {
    echo "ERROR: DHCP failed after retry. Is the UTM VM provisioned?"
    exit 1
  }
}

CLIENT_IP=$(ip -4 addr show eth1 | grep -oP 'inet \K[0-9.]+')
echo "  eth1 IP: $CLIENT_IP"

# Replace default route to go through ScoutUTM instead of VirtualBox NAT.
# Keep eth0 up for Vagrant SSH but remove its default route.
echo "Setting default route via ScoutUTM (192.168.1.1)..."
ip route del default via 10.0.2.2 dev eth0 2>/dev/null || true
ip route add default via 192.168.1.1 dev eth1

# Re-add a specific route so Vagrant SSH (via NAT on 10.0.2.x) still works.
ip route add 10.0.2.0/24 via 10.0.2.2 dev eth0 2>/dev/null || true

echo ""
echo "Routes:"
ip route

# ── Connectivity Verification ──
echo ""
echo "=== Connectivity Checks ==="

PASS=0
FAIL=0

check() {
  desc="$1"; shift
  if "$@" >/dev/null 2>&1; then
    echo "  PASS: $desc"
    PASS=$((PASS + 1))
  else
    echo "  FAIL: $desc"
    FAIL=$((FAIL + 1))
  fi
}

# 1. Ping gateway.
check "Ping ScoutUTM gateway (192.168.1.1)" ping -c 1 -W 3 192.168.1.1

# 2. DNS resolution via ScoutUTM.
check "DNS resolution via ScoutUTM" nslookup example.com 192.168.1.1

# 3. Internet access through ScoutUTM (curl through NAT chain).
check "Internet access through ScoutUTM" curl -sf --max-time 10 http://example.com

# 4. ScoutUTM web UI reachable.
check "ScoutUTM web UI (HTTPS)" curl -sfk --max-time 5 https://192.168.1.1/api/v1/health

echo ""

# ── IDS Test Trigger ──
# Fetch the testmynids.org test page so Suricata generates an alert on first boot.
# The response contains "uid=0(root)" which triggers ET Open SID 2100498.
# Traffic flows through ScoutUTM's eth1 where Suricata is monitoring.
if [ "$FAIL" -eq 0 ]; then
  echo "Triggering IDS test alert (testmynids.org)..."
  curl -sf --max-time 10 http://testmynids.org/uid/index.html >/dev/null 2>&1 \
    && echo "  OK: IDS test traffic sent" \
    || echo "  WARNING: testmynids.org fetch failed (IDS alert may not appear)"
fi

echo ""
echo "=== Client Provisioning Complete ==="
echo "  Checks passed: $PASS"
echo "  Checks failed: $FAIL"
echo "  Client IP:     $CLIENT_IP"
echo "  Gateway:       192.168.1.1"

if [ "$FAIL" -gt 0 ]; then
  echo ""
  echo "WARNING: $FAIL connectivity check(s) failed."
  echo "  The client may not have full connectivity through ScoutUTM."
  echo "  This can happen if VirtualBox NAT forwarding isn't working yet."
fi
