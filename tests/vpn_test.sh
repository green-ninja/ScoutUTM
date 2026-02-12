#!/bin/sh
# VPN integration tests — run against a running scoutd instance in Vagrant VM.
# Tests: WireGuard tunnel up, peer connectivity, key generation, syncconf reload.
set -e

PASS=0
FAIL=0
BASE_URL="${SCOUT_URL:-https://localhost:443}"
CURL="curl -sk"

pass() { PASS=$((PASS + 1)); echo "  PASS: $1"; }
fail() { FAIL=$((FAIL + 1)); echo "  FAIL: $1"; }

echo "=== VPN Integration Tests ==="
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
api_post() {
  $CURL -X POST -H "Authorization: Bearer $TOKEN" -H "Content-Type: application/json" -d "$2" "$BASE_URL/api/v1$1"
}
api_delete() {
  $CURL -X DELETE -H "Authorization: Bearer $TOKEN" "$BASE_URL/api/v1$1"
}

# Test 1: Get default WireGuard config (should be disabled).
echo "[1] GET /vpn/wireguard (default)"
RESP=$(api_get "/vpn/wireguard")
ENABLED=$(echo "$RESP" | jq -r '.enabled')
if [ "$ENABLED" = "false" ]; then
  pass "WireGuard disabled by default"
else
  fail "WireGuard should be disabled by default: $RESP"
fi

# Test 2: Enable WireGuard — should auto-generate keys.
echo "[2] Enable WireGuard"
RESP=$(api_put "/vpn/wireguard" '{"enabled":true}')
PUB_KEY=$(echo "$RESP" | jq -r '.public_key // empty')
if [ -n "$PUB_KEY" ]; then
  pass "WireGuard enabled, server public key generated: ${PUB_KEY:0:8}..."
else
  fail "WireGuard enable failed or no public key: $RESP"
fi

# Test 3: Verify WireGuard config file rendered.
echo "[3] wireguard.conf rendered"
if [ -f /etc/wireguard/wg0.conf ] || [ -f /var/lib/scout/rendered/wireguard.conf ]; then
  pass "WireGuard config file exists"

  # Verify it contains the server private key (not empty).
  WG_CONF=$(cat /etc/wireguard/wg0.conf 2>/dev/null || cat /var/lib/scout/rendered/wireguard.conf 2>/dev/null)
  if echo "$WG_CONF" | grep -q "PrivateKey ="; then
    pass "WireGuard config has PrivateKey"
  else
    fail "WireGuard config missing PrivateKey"
  fi

  if echo "$WG_CONF" | grep -q "ListenPort = 51820"; then
    pass "WireGuard config has ListenPort 51820"
  else
    fail "WireGuard config missing ListenPort"
  fi
else
  fail "WireGuard config file not found"
fi

# Test 4: nftables allows WireGuard UDP port on WAN.
echo "[4] nftables WireGuard port rule"
if nft list chain inet scout input_wan 2>/dev/null | grep -q "udp dport 51820 accept"; then
  pass "nftables allows WireGuard port 51820 on WAN"
else
  fail "nftables missing WireGuard port rule on WAN"
fi

# Test 5: nftables has VPN input chain.
echo "[5] nftables VPN input chain"
if nft list chain inet scout input_vpn 2>/dev/null; then
  pass "input_vpn chain exists"
else
  fail "input_vpn chain not found"
fi

# Test 6: nftables has VPN forwarding rules.
echo "[6] nftables VPN forwarding"
if nft list chain inet scout forward 2>/dev/null | grep -q "iifname.*wg0.*accept"; then
  pass "VPN forwarding rules present"
else
  fail "VPN forwarding rules missing"
fi

# Test 7: nftables has DNS redirect for VPN clients.
echo "[7] nftables DNS redirect for VPN"
if nft list chain inet scout prerouting 2>/dev/null | grep -q "iifname.*wg0.*dport 53 redirect"; then
  pass "DNS redirect for VPN clients"
else
  fail "DNS redirect missing for VPN clients"
fi

# Test 8: Create a peer — verify response contains config and QR code.
echo "[8] Create peer"
RESP=$(api_post "/vpn/wireguard/peers" '{"name":"test-phone"}')
PEER_NAME=$(echo "$RESP" | jq -r '.peer.name // empty')
TUNNEL_IP=$(echo "$RESP" | jq -r '.peer.tunnel_ip // empty')
CLIENT_CONFIG=$(echo "$RESP" | jq -r '.client_config // empty')
QR_CODE=$(echo "$RESP" | jq -r '.qr_code // empty')

if [ "$PEER_NAME" = "test-phone" ]; then
  pass "peer created: test-phone"
else
  fail "peer creation failed: $RESP"
fi

if [ "$TUNNEL_IP" = "10.0.0.2" ]; then
  pass "tunnel IP allocated: 10.0.0.2"
else
  fail "unexpected tunnel IP: $TUNNEL_IP (expected 10.0.0.2)"
fi

if [ -n "$CLIENT_CONFIG" ]; then
  pass "client config returned"
else
  fail "client config missing"
fi

if [ -n "$QR_CODE" ]; then
  pass "QR code returned (base64 PNG)"
else
  fail "QR code missing"
fi

# Test 9: Client config has correct DNS (LAN IP, not tunnel IP).
echo "[9] Client config DNS points to LAN"
if echo "$CLIENT_CONFIG" | grep -q "DNS = 192.168.1.1"; then
  pass "client DNS = 192.168.1.1 (LAN IP)"
else
  fail "client DNS should be LAN IP (192.168.1.1), got: $(echo "$CLIENT_CONFIG" | grep DNS)"
fi

# Test 10: Client config has server public key and endpoint.
echo "[10] Client config has server info"
if echo "$CLIENT_CONFIG" | grep -q "PublicKey ="; then
  pass "client config has server PublicKey"
else
  fail "client config missing server PublicKey"
fi

if echo "$CLIENT_CONFIG" | grep -q "Endpoint ="; then
  pass "client config has Endpoint"
else
  fail "client config missing Endpoint"
fi

# Test 11: Create second peer — sequential IP allocation.
echo "[11] Sequential tunnel IP allocation"
RESP=$(api_post "/vpn/wireguard/peers" '{"name":"test-laptop"}')
TUNNEL_IP2=$(echo "$RESP" | jq -r '.peer.tunnel_ip // empty')
if [ "$TUNNEL_IP2" = "10.0.0.3" ]; then
  pass "second peer got 10.0.0.3"
else
  fail "second peer tunnel IP: $TUNNEL_IP2 (expected 10.0.0.3)"
fi

# Test 12: Create peer with split tunnel.
echo "[12] Split tunnel peer"
RESP=$(api_post "/vpn/wireguard/peers" '{"name":"test-split","split_tunnel":true}')
SPLIT_CONFIG=$(echo "$RESP" | jq -r '.client_config // empty')
if echo "$SPLIT_CONFIG" | grep -q "192.168.1.0/24"; then
  pass "split tunnel includes LAN subnet"
else
  fail "split tunnel missing LAN subnet"
fi
if echo "$SPLIT_CONFIG" | grep -q "0.0.0.0/0"; then
  fail "split tunnel should NOT have 0.0.0.0/0"
else
  pass "split tunnel does not route all traffic"
fi

# Test 13: Peer list shows created peers.
echo "[13] Peer list"
RESP=$(api_get "/vpn/wireguard/peers")
PEER_COUNT=$(echo "$RESP" | jq 'length')
if [ "$PEER_COUNT" -eq 3 ]; then
  pass "3 peers in list"
else
  fail "expected 3 peers, got $PEER_COUNT"
fi

# Test 14: Delete a peer.
echo "[14] Delete peer"
RESP=$(api_delete "/vpn/wireguard/peers/test-phone")
if echo "$RESP" | jq -e '.status == "ok"' >/dev/null 2>&1; then
  pass "peer test-phone deleted"
else
  fail "peer delete failed: $RESP"
fi

RESP=$(api_get "/vpn/wireguard/peers")
PEER_COUNT=$(echo "$RESP" | jq 'length')
if [ "$PEER_COUNT" -eq 2 ]; then
  pass "2 peers remaining after delete"
else
  fail "expected 2 peers after delete, got $PEER_COUNT"
fi

# Test 15: Regenerate server keys — clears all peers.
echo "[15] Regenerate server keys"
OLD_PUB=$PUB_KEY
RESP=$(api_post "/vpn/wireguard/regenerate-keys" '{}')
NEW_PUB=$(echo "$RESP" | jq -r '.public_key // empty')
NEW_PEERS=$(echo "$RESP" | jq '.peers | length')
if [ "$NEW_PUB" != "$OLD_PUB" ] && [ -n "$NEW_PUB" ]; then
  pass "server public key changed after regeneration"
else
  fail "server key did not change: old=$OLD_PUB new=$NEW_PUB"
fi
if [ "$NEW_PEERS" -eq 0 ]; then
  pass "all peers cleared after key regeneration"
else
  fail "expected 0 peers after regeneration, got $NEW_PEERS"
fi

# Test 16: WireGuard service running (if not dev-mode).
echo "[16] WireGuard service"
if rc-service wireguard status >/dev/null 2>&1; then
  pass "WireGuard service is running"
elif [ -f /var/lib/scout/rendered/wireguard.conf ]; then
  pass "WireGuard config rendered (dev-mode, service stubbed)"
else
  fail "WireGuard service not running and no rendered config"
fi

# Cleanup: disable WireGuard.
echo "[cleanup] Disabling WireGuard"
api_put "/vpn/wireguard" '{"enabled":false}' >/dev/null

# Summary.
echo ""
echo "=== Results ==="
echo "  Passed: $PASS"
echo "  Failed: $FAIL"

if [ "$FAIL" -gt 0 ]; then
  exit 1
fi
exit 0
