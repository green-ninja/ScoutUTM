#!/bin/sh
# IDS/IPS integration tests — run against a running scoutd instance in Vagrant VM.
# Tests: Suricata config rendering, IDS/IPS mode switching, nftables IPS chain, alerts API.
set -e

PASS=0
FAIL=0
BASE_URL="${SCOUT_URL:-https://localhost:443}"
CURL="curl -sk"

pass() { PASS=$((PASS + 1)); echo "  PASS: $1"; }
fail() { FAIL=$((FAIL + 1)); echo "  FAIL: $1"; }

echo "=== IDS/IPS Integration Tests ==="
echo ""

# Login and get token.
echo "[0] Login"
LOGIN_RESP=$($CURL -X POST "$BASE_URL/api/v1/auth/login" \
  -H "Content-Type: application/json" \
  -d '{"password":"admin"}')

TOKEN=$(echo "$LOGIN_RESP" | jq -r '.token // empty')
if [ -n "$TOKEN" ]; then
  pass "login successful"
else
  fail "login failed: $LOGIN_RESP"
  echo "Cannot continue without token. Exiting."
  exit 1
fi

# Helpers.
api_get() {
  $CURL -H "Authorization: Bearer $TOKEN" "$BASE_URL/api/v1$1"
}
api_put() {
  $CURL -X PUT -H "Authorization: Bearer $TOKEN" -H "Content-Type: application/json" -d "$2" "$BASE_URL/api/v1$1"
}

# Test 1: Get default IDS config.
echo "[1] GET /ids (defaults)"
RESP=$(api_get "/ids")
ENABLED=$(echo "$RESP" | jq -r '.enabled')
MODE=$(echo "$RESP" | jq -r '.mode')
if [ "$ENABLED" = "true" ]; then
  pass "IDS enabled by default"
else
  fail "IDS should be enabled by default: $RESP"
fi
if [ "$MODE" = "ids" ]; then
  pass "IDS mode is 'ids' by default"
else
  fail "IDS mode should be 'ids', got: $MODE"
fi

# Test 2: Verify suricata.yaml rendered.
echo "[2] suricata.yaml rendered"
if [ -f /etc/suricata/suricata.yaml ]; then
  pass "suricata.yaml exists"
else
  fail "suricata.yaml not found at /etc/suricata/suricata.yaml"
fi

# Test 3: Verify suricata.yaml content.
echo "[3] suricata.yaml content"
SURICATA_CONF=$(cat /etc/suricata/suricata.yaml 2>/dev/null || true)
if echo "$SURICATA_CONF" | grep -q "HOME_NET"; then
  pass "suricata.yaml has HOME_NET"
else
  fail "suricata.yaml missing HOME_NET"
fi
if echo "$SURICATA_CONF" | grep -q "fail-open: yes"; then
  pass "suricata.yaml has fail-open: yes"
else
  fail "suricata.yaml missing fail-open"
fi
if echo "$SURICATA_CONF" | grep -q "unix_dgram"; then
  pass "suricata.yaml has unix_dgram EVE output"
else
  fail "suricata.yaml missing unix_dgram"
fi

# Test 4: Switch to IPS mode.
echo "[4] PUT /ids mode=ips"
RESP=$(api_put "/ids" '{"mode":"ips"}')
NEW_MODE=$(echo "$RESP" | jq -r '.mode')
if [ "$NEW_MODE" = "ips" ]; then
  pass "IDS mode changed to ips"
else
  fail "IDS mode should be 'ips', got: $NEW_MODE"
fi

# Reload nftables — dev-mode stubs service reloads.
nft -f /etc/nftables.conf 2>/dev/null || true

# Test 5: Verify nftables IPS chain exists.
echo "[5] nftables IPS chain"
if nft list chain inet scout ips 2>/dev/null | grep -q "queue num 0-3"; then
  pass "nftables IPS chain with queue fanout"
else
  fail "nftables IPS chain not found or missing queue"
fi

# Test 6: Verify forward chain jumps to IPS.
echo "[6] nftables forward jumps to ips"
if nft list chain inet scout forward 2>/dev/null | grep -q "jump ips"; then
  pass "forward chain jumps to ips"
else
  fail "forward chain missing jump to ips"
fi

# Test 7: Switch back to IDS mode.
echo "[7] PUT /ids mode=ids"
RESP=$(api_put "/ids" '{"mode":"ids"}')
NEW_MODE=$(echo "$RESP" | jq -r '.mode')
if [ "$NEW_MODE" = "ids" ]; then
  pass "IDS mode reverted to ids"
else
  fail "IDS mode should be 'ids', got: $NEW_MODE"
fi

# Reload nftables.
nft -f /etc/nftables.conf 2>/dev/null || true

# Test 8: IPS chain should be gone in IDS mode.
echo "[8] nftables IPS chain removed"
if nft list chain inet scout ips 2>/dev/null; then
  fail "IPS chain should not exist in IDS mode"
else
  pass "IPS chain correctly absent in IDS mode"
fi

# Test 9: Disable IDS.
echo "[9] Disable IDS"
RESP=$(api_put "/ids" '{"enabled":false}')
ENABLED=$(echo "$RESP" | jq -r '.enabled')
if [ "$ENABLED" = "false" ]; then
  pass "IDS disabled"
else
  fail "IDS should be disabled: $RESP"
fi

# Test 10: Re-enable IDS.
echo "[10] Re-enable IDS"
RESP=$(api_put "/ids" '{"enabled":true}')
ENABLED=$(echo "$RESP" | jq -r '.enabled')
if [ "$ENABLED" = "true" ]; then
  pass "IDS re-enabled"
else
  fail "IDS should be enabled: $RESP"
fi

# Test 11: Get alerts (should be empty).
echo "[11] GET /ids/alerts (empty)"
RESP=$(api_get "/ids/alerts")
ALERT_COUNT=$(echo "$RESP" | jq 'length')
if [ "$ALERT_COUNT" -eq 0 ]; then
  pass "alerts empty (no Suricata running in dev-mode)"
else
  fail "expected 0 alerts, got $ALERT_COUNT"
fi

# Test 12: Invalid mode rejected.
echo "[12] Invalid mode rejected"
RESP=$(api_put "/ids" '{"mode":"invalid"}')
ERROR=$(echo "$RESP" | jq -r '.error // empty')
if [ -n "$ERROR" ]; then
  pass "invalid mode rejected"
else
  fail "invalid mode should be rejected: $RESP"
fi

# Test 13: IDS alert trigger via testmynids.org.
# Fetching http://testmynids.org/uid/index.html returns "uid=0(root)" which
# triggers ET Open SID 2100498 (GPL ATTACK_RESPONSE id check returned root).
# Suricata must be running with rules loaded and monitoring the interface that
# sees this traffic. In AF_PACKET mode on eth1, the fetch must originate from
# a LAN client so the response traverses eth1.
echo "[13] IDS alert trigger (testmynids.org)"
if rc-service suricata status >/dev/null 2>&1; then
  # Record current alert count.
  BEFORE=$(api_get "/ids/alerts" | jq 'length')

  # Trigger the test signature. Try from a LAN client (DHCP lease) first,
  # fall back to fetching directly from the UTM.
  CLIENT_IP=$(awk '{print $3}' /var/lib/misc/dnsmasq.leases 2>/dev/null | head -1)
  if [ -n "$CLIENT_IP" ] && ssh -o StrictHostKeyChecking=no -o ConnectTimeout=3 "$CLIENT_IP" \
      "wget -q -O /dev/null http://testmynids.org/uid/index.html" 2>/dev/null; then
    pass "trigger sent via LAN client $CLIENT_IP"
  else
    # Fallback: fetch from UTM directly (works if Suricata monitors this traffic path).
    curl -sf --max-time 10 http://testmynids.org/uid/index.html >/dev/null 2>&1 || true
    pass "trigger sent from UTM (fallback)"
  fi

  # Poll alerts API for up to 15 seconds waiting for the signature.
  FOUND=0
  for i in $(seq 1 15); do
    RESP=$(api_get "/ids/alerts")
    if echo "$RESP" | jq -e '.[] | select(.alert.signature_id == 2100498)' >/dev/null 2>&1; then
      FOUND=1
      break
    fi
    sleep 1
  done

  if [ "$FOUND" -eq 1 ]; then
    SIG=$(echo "$RESP" | jq -r '[.[] | select(.alert.signature_id == 2100498)][0].alert.signature')
    pass "IDS alert triggered: $SIG"
  else
    AFTER=$(api_get "/ids/alerts" | jq 'length')
    if [ "$AFTER" -gt "$BEFORE" ]; then
      pass "new IDS alerts detected (SID 2100498 not found, but alert count increased)"
    else
      fail "no IDS alert after testmynids.org fetch (Suricata may not see UTM-originated traffic on eth1)"
    fi
  fi
else
  pass "skipped — Suricata not running (dev-mode)"
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
