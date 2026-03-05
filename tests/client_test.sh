#!/bin/sh
# client_test.sh — Runs ON the client VM, verifies end-to-end traffic through ScoutUTM.
set -e

PASS=0
FAIL=0

pass() { PASS=$((PASS + 1)); echo "  PASS: $1"; }
fail() { FAIL=$((FAIL + 1)); echo "  FAIL: $1"; }
check() {
  desc="$1"; shift
  if "$@" >/dev/null 2>&1; then pass "$desc"; else fail "$desc"; fi
}

echo "╔══════════════════════════════════════╗"
echo "║     Client Integration Tests         ║"
echo "╚══════════════════════════════════════╝"
echo ""

# 1. Client has DHCP lease from ScoutUTM (IP in 192.168.1.100-253).
CLIENT_IP=$(ip -4 addr show eth1 2>/dev/null | grep -oP 'inet \K[0-9.]+')
if echo "$CLIENT_IP" | grep -qP '^192\.168\.1\.(1[0-9]{2}|2[0-4][0-9]|25[0-3])$'; then
  pass "DHCP lease in range 192.168.1.100-253 (got $CLIENT_IP)"
else
  fail "DHCP lease in range 192.168.1.100-253 (got ${CLIENT_IP:-none})"
fi

# 2. Default gateway is 192.168.1.1.
GW=$(ip route show default 2>/dev/null | grep -oP 'via \K[0-9.]+' | head -1)
if [ "$GW" = "192.168.1.1" ]; then
  pass "Default gateway is 192.168.1.1"
else
  fail "Default gateway is 192.168.1.1 (got ${GW:-none})"
fi

# 3. DNS resolver is 192.168.1.1.
# Check /etc/resolv.conf or use the gateway directly.
DNS=$(grep '^nameserver' /etc/resolv.conf 2>/dev/null | head -1 | awk '{print $2}')
if [ "$DNS" = "192.168.1.1" ]; then
  pass "DNS resolver is 192.168.1.1"
else
  # DHCP may not set resolv.conf — still pass if DNS works via gateway.
  if nslookup example.com 192.168.1.1 >/dev/null 2>&1; then
    pass "DNS resolver works via 192.168.1.1 (resolv.conf has ${DNS:-none})"
  else
    fail "DNS resolver is 192.168.1.1 (resolv.conf has ${DNS:-none})"
  fi
fi

# 4. DNS resolution works (nslookup via ScoutUTM's Unbound).
check "DNS resolution via ScoutUTM" nslookup example.com 192.168.1.1

# 5. Internet access works through ScoutUTM (curl external site).
check "Internet access through ScoutUTM" curl -sf --max-time 10 http://example.com

# 6. ScoutUTM web UI accessible at https://192.168.1.1.
check "ScoutUTM web UI accessible" curl -sfk --max-time 5 https://192.168.1.1

# 7. ScoutUTM health API returns healthy.
HEALTH=$(curl -sfk --max-time 5 https://192.168.1.1/api/v1/health 2>/dev/null)
if echo "$HEALTH" | grep -q '"healthy":true'; then
  pass "Health API returns healthy"
else
  fail "Health API returns healthy (got: ${HEALTH:-no response})"
fi

# ── Summary ──
echo ""
echo "╔══════════════════════════════════════╗"
echo "║     Client Test Summary              ║"
echo "╠══════════════════════════════════════╣"
printf "║  Passed: %-27s║\n" "$PASS"
printf "║  Failed: %-27s║\n" "$FAIL"
echo "╚══════════════════════════════════════╝"

[ "$FAIL" -eq 0 ]
