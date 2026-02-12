#!/bin/sh
# Network integration tests — run inside Vagrant VM with real daemons.
set -e

PASS=0
FAIL=0
BASE_URL="${SCOUT_URL:-https://localhost:443}"

pass() { PASS=$((PASS + 1)); echo "  PASS: $1"; }
fail() { FAIL=$((FAIL + 1)); echo "  FAIL: $1"; }

echo "=== Network Integration Tests ==="
echo ""

# Ensure scout nftables rules are loaded (dev-mode stubs service reloads).
nft -f /etc/nftables.conf 2>/dev/null || true

# Test 1: nftables rules loaded.
echo "[1] nftables rules loaded"
if nft list table inet scout >/dev/null 2>&1; then
  pass "inet scout table exists"
else
  fail "inet scout table not found"
fi

# Test 2: Zone device sets exist.
echo "[2] Zone device sets"
if nft list set inet scout wan_devices >/dev/null 2>&1; then
  pass "wan_devices set exists"
else
  fail "wan_devices set not found"
fi

if nft list set inet scout lan_devices >/dev/null 2>&1; then
  pass "lan_devices set exists"
else
  fail "lan_devices set not found"
fi

# Test 3: IPv6 drop rule present.
echo "[3] IPv6 drop rule"
if nft list chain inet scout prerouting_raw 2>/dev/null | grep -q "meta nfproto ipv6 drop"; then
  pass "IPv6 drop rule present"
else
  fail "IPv6 drop rule not found"
fi

# Test 4: DNS redirect rule present.
echo "[4] DNS redirect rule"
if nft list chain inet scout prerouting 2>/dev/null | grep -q "dport 53 redirect"; then
  pass "DNS redirect rule present"
else
  fail "DNS redirect rule not found"
fi

# Test 5: Masquerade on WAN.
echo "[5] NAT masquerade"
if nft list chain inet scout postrouting 2>/dev/null | grep -q "masquerade"; then
  pass "masquerade rule present"
else
  fail "masquerade rule not found"
fi

# Test 6: DHCP serving on LAN.
echo "[6] DHCP server (dnsmasq)"
if rc-service dnsmasq status >/dev/null 2>&1; then
  pass "dnsmasq is running"
else
  fail "dnsmasq is not running"
fi

# Test 7: dnsmasq config has DNS disabled.
echo "[7] dnsmasq DNS disabled"
if grep -q "port=0" /etc/dnsmasq.conf 2>/dev/null; then
  pass "dnsmasq port=0 (DNS disabled)"
else
  fail "dnsmasq DNS not disabled"
fi

# Test 8: chrony running.
echo "[8] NTP (chronyd)"
if rc-service chronyd status >/dev/null 2>&1; then
  pass "chronyd is running"
else
  fail "chronyd is not running"
fi

# Summary.
echo ""
echo "=== Results ==="
echo "  Passed: $PASS"
echo "  Failed: $FAIL"

if [ "$FAIL" -gt 0 ]; then
  exit 1
fi
exit 0
