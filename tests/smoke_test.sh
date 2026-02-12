#!/bin/sh
# Smoke test for ScoutUTM — run inside the Vagrant VM or against a running scoutd.
set -e

PASS=0
FAIL=0
WARN=0
BASE_URL="${SCOUT_URL:-https://localhost:443}"

pass() { PASS=$((PASS + 1)); echo "  PASS: $1"; }
fail() { FAIL=$((FAIL + 1)); echo "  FAIL: $1"; }
warn() { WARN=$((WARN + 1)); echo "  WARN: $1"; }

echo "=== ScoutUTM Smoke Tests ==="
echo "Target: $BASE_URL"
echo ""

# Test 1: Health endpoint responds.
echo "[1] Health endpoint"
HEALTH=$(curl -sk "$BASE_URL/api/v1/health" 2>/dev/null)
if echo "$HEALTH" | jq -e '.healthy == true' >/dev/null 2>&1; then
  pass "health endpoint returns healthy=true"
else
  fail "health endpoint did not return healthy=true"
fi

# Test 2: Version field present.
echo "[2] Version field"
if echo "$HEALTH" | jq -e '.version' >/dev/null 2>&1; then
  VER=$(echo "$HEALTH" | jq -r '.version')
  pass "version field present: $VER"
else
  fail "version field missing from health response"
fi

# Test 3: Config file exists and is valid JSON.
echo "[3] Config file"
if [ -f /etc/scout/config.json ]; then
  if jq empty /etc/scout/config.json 2>/dev/null; then
    pass "config.json exists and is valid JSON"
  else
    fail "config.json exists but is not valid JSON"
  fi
else
  fail "config.json does not exist"
fi

# Test 4: Config has all top-level keys.
echo "[4] Config keys"
if [ -f /etc/scout/config.json ]; then
  MISSING=""
  for key in version system network firewall dns vpn ids parental update; do
    if ! jq -e ".$key" /etc/scout/config.json >/dev/null 2>&1; then
      MISSING="$MISSING $key"
    fi
  done
  if [ -z "$MISSING" ]; then
    pass "config has all top-level keys"
  else
    fail "config missing keys:$MISSING"
  fi
fi

# Test 5: Services running.
echo "[5] Services"
for svc in nftables dnsmasq chronyd scoutd; do
  if rc-service "$svc" status >/dev/null 2>&1; then
    pass "$svc is running"
  else
    fail "$svc is not running"
  fi
done
# Suricata may not be running without rules — treat as warning.
if rc-service suricata status >/dev/null 2>&1; then
  pass "suricata is running"
else
  warn "suricata is not running (may need rules installed first)"
fi
# Unbound check.
if rc-service unbound status >/dev/null 2>&1; then
  pass "unbound is running"
else
  fail "unbound is not running"
fi

# Test 6: TLS cert exists.
echo "[6] TLS certificate"
if [ -f /etc/scout/tls/server.crt ] && [ -f /etc/scout/tls/server.key ]; then
  pass "TLS cert and key exist"
else
  fail "TLS cert or key missing"
fi

# Summary.
echo ""
echo "=== Results ==="
echo "  Passed: $PASS"
echo "  Failed: $FAIL"
echo "  Warnings: $WARN"

if [ "$FAIL" -gt 0 ]; then
  exit 1
fi
exit 0
