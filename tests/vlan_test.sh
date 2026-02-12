#!/bin/sh
# VLAN integration tests — run against a running scoutd instance in Vagrant VM.
# Tests: VLAN creation, zone isolation, inter-zone forwarding rules.
set -e

PASS=0
FAIL=0
BASE_URL="${SCOUT_URL:-https://localhost:443}"
CURL="curl -sk"

pass() { PASS=$((PASS + 1)); echo "  PASS: $1"; }
fail() { FAIL=$((FAIL + 1)); echo "  FAIL: $1"; }

echo "=== VLAN Integration Tests ==="
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
api_post() {
  $CURL -X POST -H "Authorization: Bearer $TOKEN" -H "Content-Type: application/json" -d "$2" "$BASE_URL/api/v1$1"
}
api_delete() {
  $CURL -X DELETE -H "Authorization: Bearer $TOKEN" "$BASE_URL/api/v1$1"
}

# Test 1: Create IoT zone.
echo "[1] Create IoT zone"
RESP=$(api_post "/network/zones" '{"name":"iot","input_policy":"drop","forward_policy":"drop","output_policy":"accept","masquerade":false}')
if echo "$RESP" | jq -e '.name == "iot"' >/dev/null 2>&1; then
  pass "IoT zone created"
else
  fail "IoT zone creation failed: $RESP"
fi

# Test 2: Create VLAN 20 for IoT zone.
echo "[2] Create VLAN 20 (IoT)"
RESP=$(api_post "/network/vlans" '{"id":20,"parent":"eth1","zone":"iot","subnet":"192.168.20.1/24","dhcp_range":{"start":"192.168.20.100","end":"192.168.20.253"}}')
if echo "$RESP" | jq -e '.id == 20' >/dev/null 2>&1; then
  pass "VLAN 20 created"
else
  fail "VLAN 20 creation failed: $RESP"
fi

# Test 3: Verify VLAN appears in list.
echo "[3] GET /network/vlans includes VLAN 20"
RESP=$(api_get "/network/vlans")
VLAN_COUNT=$(echo "$RESP" | jq '[.[] | select(.id == 20)] | length')
if [ "$VLAN_COUNT" -eq 1 ]; then
  pass "VLAN 20 in list"
else
  fail "VLAN 20 not found in list: $RESP"
fi

# Test 4: Duplicate VLAN ID rejected.
echo "[4] Duplicate VLAN ID rejected"
RESP=$(api_post "/network/vlans" '{"id":20,"parent":"eth1","zone":"iot","subnet":"192.168.21.1/24","dhcp_range":{"start":"192.168.21.100","end":"192.168.21.253"}}')
if echo "$RESP" | jq -e '.error' >/dev/null 2>&1; then
  pass "duplicate VLAN ID rejected"
else
  fail "duplicate VLAN ID should be rejected: $RESP"
fi

# Test 5: Invalid VLAN ID rejected.
echo "[5] Invalid VLAN ID rejected"
RESP=$(api_post "/network/vlans" '{"id":5000,"parent":"eth1","zone":"iot","subnet":"192.168.22.1/24"}')
if echo "$RESP" | jq -e '.error' >/dev/null 2>&1; then
  pass "invalid VLAN ID (5000) rejected"
else
  fail "VLAN ID 5000 should be rejected: $RESP"
fi

# Test 6: Create Guest zone + VLAN 30.
echo "[6] Create Guest zone + VLAN 30"
api_post "/network/zones" '{"name":"guest","input_policy":"drop","forward_policy":"drop","output_policy":"accept","masquerade":false}' >/dev/null
RESP=$(api_post "/network/vlans" '{"id":30,"parent":"eth1","zone":"guest","subnet":"192.168.30.1/24","dhcp_range":{"start":"192.168.30.100","end":"192.168.30.253"}}')
if echo "$RESP" | jq -e '.id == 30' >/dev/null 2>&1; then
  pass "VLAN 30 (Guest) created"
else
  fail "VLAN 30 creation failed: $RESP"
fi

# Reload nftables — dev-mode stubs service reloads, so apply rendered config manually.
nft -f /etc/nftables.conf 2>/dev/null || true

# Test 7: Verify nftables has VLAN sub-interface in zone set.
echo "[7] nftables zone sets include VLAN interfaces"
if nft list table inet scout 2>/dev/null | grep -q "eth1.20"; then
  pass "eth1.20 (IoT VLAN) in nftables"
else
  fail "eth1.20 not found in nftables rules"
fi

if nft list table inet scout 2>/dev/null | grep -q "eth1.30"; then
  pass "eth1.30 (Guest VLAN) in nftables"
else
  fail "eth1.30 not found in nftables rules"
fi

# Test 8: Verify IoT zone has its own input chain (isolated from LAN).
echo "[8] Zone isolation — IoT has separate input chain"
if nft list chain inet scout input_iot 2>/dev/null | grep -q "drop"; then
  pass "input_iot chain exists with drop policy"
else
  fail "input_iot chain not found or missing drop"
fi

# Test 9: Verify dnsmasq has DHCP for VLAN interface.
echo "[9] dnsmasq DHCP for VLAN interfaces"
if grep -q "interface=eth1.20" /etc/dnsmasq.conf 2>/dev/null; then
  pass "dnsmasq serves DHCP on eth1.20"
else
  fail "dnsmasq missing eth1.20 interface"
fi

if grep -q "dhcp-range=eth1.20" /etc/dnsmasq.conf 2>/dev/null; then
  pass "dnsmasq has DHCP range for eth1.20"
else
  fail "dnsmasq missing DHCP range for eth1.20"
fi

# Test 10: Verify DNS redirect for VLAN zone.
echo "[10] DNS redirect for VLAN zones"
if nft list chain inet scout prerouting 2>/dev/null | grep -q "iifname @iot_devices.*dport 53 redirect"; then
  pass "DNS redirect for IoT zone"
else
  fail "DNS redirect missing for IoT zone"
fi

# Test 11: chrony allows VLAN subnet.
echo "[11] chrony allows VLAN subnets"
if grep -q "allow 192.168.20.0/24" /etc/chrony/chrony.conf 2>/dev/null; then
  pass "chrony allows IoT VLAN subnet"
else
  fail "chrony missing allow for IoT VLAN subnet"
fi

# Cleanup: delete VLANs, then zones.
echo "[cleanup] Removing test VLANs and zones"
api_delete "/network/vlans/20" >/dev/null
api_delete "/network/vlans/30" >/dev/null
api_delete "/network/zones/iot" >/dev/null
api_delete "/network/zones/guest" >/dev/null

# Summary.
echo ""
echo "=== Results ==="
echo "  Passed: $PASS"
echo "  Failed: $FAIL"

if [ "$FAIL" -gt 0 ]; then
  exit 1
fi
exit 0
