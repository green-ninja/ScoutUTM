#!/bin/sh
# Provision the second client VM on VLAN 20 (IoT zone) and run VLAN
# integration tests. This script creates the IoT zone and VLAN via the
# UTM API, sets up 802.1Q tagging on the client, and verifies VLAN
# connectivity + zone isolation against the LAN client.
set -e

echo "=== ScoutUTM VLAN Client (client2) Provisioning ==="

UTM_LAN_IP="192.168.1.1"
UTM_API="https://${UTM_LAN_IP}/api/v1"
CURL="curl -sk"

# ── Phase 1: System Setup ──

echo "nameserver 8.8.8.8" > /etc/resolv.conf

ALPINE_VER=$(cat /etc/alpine-release | cut -d. -f1,2)
cat > /etc/apk/repositories << EOF
https://dl-cdn.alpinelinux.org/alpine/v${ALPINE_VER}/main
https://dl-cdn.alpinelinux.org/alpine/v${ALPINE_VER}/community
EOF

apk update
apk add curl bind-tools iproute2

# ── Phase 2: Bootstrap LAN access (untagged) to reach UTM API ──

echo "Bringing up eth1 (untagged) to reach UTM API..."
ip link set eth1 up
udhcpc -i eth1 -n -q 2>/dev/null || {
  echo "Retrying DHCP on eth1 in 5 seconds..."
  sleep 5
  udhcpc -i eth1 -n -q 2>/dev/null || {
    echo "ERROR: DHCP on eth1 failed. Is the UTM VM provisioned?"
    exit 1
  }
}

BOOTSTRAP_IP=$(ip -4 addr show eth1 | grep -oP 'inet \K[0-9.]+')
echo "  Bootstrap IP (LAN): $BOOTSTRAP_IP"

# ── Phase 3: Create IoT zone + VLAN 20 via UTM API ──

echo ""
echo "=== Creating IoT VLAN via UTM API ==="

# Login.
LOGIN_RESP=$($CURL -X POST "$UTM_API/auth/login" \
  -H "Content-Type: application/json" \
  -d '{"password":"admin"}')
TOKEN=$(echo "$LOGIN_RESP" | jq -r '.token // empty')
if [ -z "$TOKEN" ]; then
  echo "ERROR: Login failed: $LOGIN_RESP"
  exit 1
fi
echo "  Logged in to UTM API"

api_post() {
  $CURL -X POST -H "Authorization: Bearer $TOKEN" \
    -H "Content-Type: application/json" -d "$2" "$UTM_API$1"
}
api_get() {
  $CURL -H "Authorization: Bearer $TOKEN" "$UTM_API$1"
}

# Create IoT zone (ignore conflict if already exists).
RESP=$(api_post "/network/zones" \
  '{"name":"iot","input_policy":"drop","forward_policy":"drop","output_policy":"accept","masquerade":false}')
echo "  IoT zone: $(echo "$RESP" | jq -r '.name // .error')"

# Add IoT → WAN forwarding so VLAN clients can reach the internet.
RESP=$($CURL -X PUT -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" "$UTM_API/firewall" \
  -d "$(api_get /firewall | jq '.forwardings += [{"src":"iot","dest":"wan"}] | .forwardings |= unique_by(.src + .dest)')")
echo "  IoT→WAN forwarding: $(echo "$RESP" | jq -r '.forwardings | length') forwarding rule(s)"

# Create VLAN 20 (ignore conflict if already exists).
RESP=$(api_post "/network/vlans" \
  '{"id":20,"parent":"eth1","zone":"iot","subnet":"192.168.20.0/24","dhcp_range":{"start":"192.168.20.100","end":"192.168.20.200"}}')
echo "  VLAN 20: $(echo "$RESP" | jq -r '.id // .error')"

# Give UTM a moment to create the interface and reload daemons.
sleep 2

# ── Phase 4: Switch to VLAN 20 tagged interface ──

echo ""
echo "=== Setting up 802.1Q VLAN interface ==="

# Release the untagged DHCP lease and tear down bootstrap.
ip route del default via "$UTM_LAN_IP" dev eth1 2>/dev/null || true
ip addr flush dev eth1

# Create VLAN sub-interface with 802.1Q tagging.
ip link add link eth1 name eth1.20 type vlan id 20

# Request DHCP on the tagged interface from UTM's dnsmasq.
ip link set eth1.20 up
echo "Requesting DHCP on eth1.20 (VLAN 20)..."
udhcpc -i eth1.20 -n -q 2>/dev/null || {
  echo "Retrying DHCP on eth1.20 in 5 seconds..."
  sleep 5
  udhcpc -i eth1.20 -n -q 2>/dev/null || {
    echo "ERROR: DHCP on eth1.20 (VLAN 20) failed."
    echo "  This means UTM's dnsmasq is not serving DHCP on the VLAN interface."
    exit 1
  }
}

CLIENT_IP=$(ip -4 addr show eth1.20 | grep -oP 'inet \K[0-9.]+')
echo "  VLAN IP: $CLIENT_IP"

# Set default route via UTM's VLAN gateway.
ip route del default 2>/dev/null || true
ip route add default via 192.168.20.1 dev eth1.20

# Re-add NAT route so Vagrant SSH still works.
ip route add 10.0.2.0/24 via 10.0.2.2 dev eth0 2>/dev/null || true

echo ""
echo "Routes:"
ip route

# ── Phase 5: VLAN Integration Test Suite ──

echo ""
echo "========================================="
echo "  VLAN Integration Test Suite"
echo "========================================="
echo ""

PASS=0
FAIL=0

pass() { PASS=$((PASS + 1)); echo "  PASS: $1"; }
fail() { FAIL=$((FAIL + 1)); echo "  FAIL: $1"; }

# ── Test 1: DHCP assigned IP in VLAN range ──
echo "[1] DHCP assigned IP in VLAN 20 range (192.168.20.100-200)"
case "$CLIENT_IP" in
  192.168.20.*)
    pass "IP $CLIENT_IP is in VLAN 20 subnet"
    ;;
  *)
    fail "IP $CLIENT_IP is NOT in VLAN 20 subnet"
    ;;
esac

# ── Test 2: Ping VLAN gateway ──
echo "[2] Ping VLAN gateway (192.168.20.1)"
if ping -c 1 -W 3 192.168.20.1 >/dev/null 2>&1; then
  pass "VLAN gateway reachable"
else
  fail "VLAN gateway unreachable"
fi

# ── Test 3: DNS resolution via UTM ──
echo "[3] DNS resolution via UTM (from VLAN)"
if nslookup example.com 192.168.20.1 >/dev/null 2>&1; then
  pass "DNS resolution works via VLAN gateway"
else
  fail "DNS resolution failed from VLAN"
fi

# ── Test 4: Internet access through UTM ──
echo "[4] Internet access through UTM (IoT→WAN forwarding)"
if curl -sf --max-time 10 http://example.com >/dev/null 2>&1; then
  pass "internet access works from VLAN"
else
  fail "internet access failed from VLAN"
fi

# ── Test 5: UTM health API reachable via VLAN gateway ──
echo "[5] UTM health API via VLAN gateway"
HEALTH=$($CURL --max-time 5 "https://192.168.20.1/api/v1/health" 2>/dev/null)
if echo "$HEALTH" | jq -e '.healthy == true' >/dev/null 2>&1; then
  pass "UTM health API reachable from VLAN"
else
  fail "UTM health API not reachable from VLAN: $HEALTH"
fi

# ── Test 6: Zone isolation — cannot reach LAN gateway ──
echo "[6] Zone isolation — cannot reach LAN gateway (192.168.1.1)"
# IoT zone has input_policy=drop, so ICMP to UTM's LAN IP should be dropped.
if ping -c 1 -W 2 192.168.1.1 >/dev/null 2>&1; then
  fail "VLAN client can ping LAN gateway — isolation broken"
else
  pass "LAN gateway unreachable from VLAN (input_policy=drop)"
fi

# ── Test 7: Zone isolation — cannot reach LAN subnet ──
echo "[7] Zone isolation — cannot reach LAN subnet (192.168.1.100)"
# IoT→LAN forwarding is not configured, so forwarded packets should be dropped.
if ping -c 1 -W 2 192.168.1.100 >/dev/null 2>&1; then
  fail "VLAN client can reach LAN host — forwarding isolation broken"
else
  pass "LAN subnet unreachable from VLAN (forward_policy=drop)"
fi

# ── Test 8: Verify VLAN interface exists on UTM ──
echo "[8] UTM has VLAN sub-interface (via API)"
# Re-login via VLAN gateway (the old token was from LAN bootstrap).
LOGIN_RESP=$($CURL -X POST "https://192.168.20.1/api/v1/auth/login" \
  -H "Content-Type: application/json" \
  -d '{"password":"admin"}')
TOKEN=$(echo "$LOGIN_RESP" | jq -r '.token // empty')
if [ -z "$TOKEN" ]; then
  fail "cannot re-login via VLAN gateway"
else
  VLANS=$($CURL -H "Authorization: Bearer $TOKEN" "https://192.168.20.1/api/v1/network/vlans")
  if echo "$VLANS" | jq -e '.[] | select(.id == 20)' >/dev/null 2>&1; then
    pass "VLAN 20 present in UTM config"
  else
    fail "VLAN 20 not found in UTM config: $VLANS"
  fi
fi

# ── Test 9: Device visible in UTM device list ──
echo "[9] Client2 visible in UTM device list"
# Wait for device scanner to pick up the new client (runs every 30s).
FOUND=false
for i in 1 2 3; do
  DEVICES=$($CURL -H "Authorization: Bearer $TOKEN" "https://192.168.20.1/api/v1/devices" 2>/dev/null)
  if echo "$DEVICES" | jq -e ".[] | select(.ip == \"$CLIENT_IP\")" >/dev/null 2>&1; then
    FOUND=true
    break
  fi
  sleep 10
done
if [ "$FOUND" = "true" ]; then
  pass "client2 ($CLIENT_IP) visible in device list"
else
  fail "client2 ($CLIENT_IP) not found in device list after 30s"
fi

# ── Test 10: nftables has IoT zone chain ──
echo "[10] nftables IoT zone isolation (via API config)"
FIREWALL=$($CURL -H "Authorization: Bearer $TOKEN" "https://192.168.20.1/api/v1/firewall")
IOT_FWD=$(echo "$FIREWALL" | jq '[.forwardings[] | select(.src == "iot" and .dest == "lan")] | length')
if [ "$IOT_FWD" = "0" ]; then
  pass "no IoT→LAN forwarding rule (isolated)"
else
  fail "IoT→LAN forwarding rule exists — isolation compromised"
fi

# ── Test 11: dnsmasq serves DHCP on VLAN (verify via lease) ──
echo "[11] DHCP lease obtained on VLAN interface"
LEASE_IP=$(ip -4 addr show eth1.20 | grep -oP 'inet \K[0-9.]+')
if [ -n "$LEASE_IP" ] && echo "$LEASE_IP" | grep -q "^192\.168\.20\."; then
  pass "valid DHCP lease on eth1.20: $LEASE_IP"
else
  fail "no valid DHCP lease on eth1.20"
fi

# ── Test 12: VLAN tagged frames work (verify VLAN ID) ──
echo "[12] VLAN 20 tagging active on eth1.20"
VLAN_ID=$(ip -d link show eth1.20 2>/dev/null | grep -oP 'vlan.*id \K[0-9]+')
if [ "$VLAN_ID" = "20" ]; then
  pass "eth1.20 is VLAN ID 20"
else
  fail "eth1.20 VLAN ID incorrect: $VLAN_ID"
fi

# ── Summary ──

echo ""
echo "========================================="
echo "  VLAN Test Results"
echo "========================================="
echo "  Passed: $PASS"
echo "  Failed: $FAIL"
echo "  Client IP:  $CLIENT_IP (VLAN 20)"
echo "  Gateway:    192.168.20.1"
echo "========================================="

if [ "$FAIL" -gt 0 ]; then
  echo ""
  echo "WARNING: $FAIL VLAN test(s) failed."
fi
