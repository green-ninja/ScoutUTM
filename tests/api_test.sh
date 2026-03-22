#!/bin/sh
# API integration tests — run against a running scoutd instance.
set -e

PASS=0
FAIL=0
BASE_URL="${SCOUT_URL:-https://localhost:443}"
CURL="curl -sk"

pass() { PASS=$((PASS + 1)); echo "  PASS: $1"; }
fail() { FAIL=$((FAIL + 1)); echo "  FAIL: $1"; }

echo "=== API Integration Tests ==="
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

AUTH="-H \"Authorization: Bearer $TOKEN\""

# Helper to make authenticated requests.
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

# Test 1: GET /firewall returns config.
echo "[1] GET /firewall"
RESP=$(api_get "/firewall")
if echo "$RESP" | jq -e '.forwardings' >/dev/null 2>&1; then
  pass "GET /firewall returns firewall config"
else
  fail "GET /firewall unexpected response: $RESP"
fi

# Test 2: GET /network returns config.
echo "[2] GET /network"
RESP=$(api_get "/network")
if echo "$RESP" | jq -e '.interfaces' >/dev/null 2>&1; then
  pass "GET /network returns network config"
else
  fail "GET /network unexpected response: $RESP"
fi

# Test 3: GET /system returns config without admin_hash.
echo "[3] GET /system"
RESP=$(api_get "/system")
if echo "$RESP" | jq -e '.hostname' >/dev/null 2>&1; then
  pass "GET /system returns system config"
else
  fail "GET /system unexpected response: $RESP"
fi
if echo "$RESP" | jq -e '.admin_hash' >/dev/null 2>&1; then
  fail "GET /system leaks admin_hash"
else
  pass "GET /system hides admin_hash"
fi

# Test 4: CRUD firewall rules.
echo "[4] CRUD firewall rules"
RESP=$(api_post "/firewall/rules" '{"name":"test-rule","src":"lan","dest":"wan","proto":"tcp","port":8080,"action":"drop"}')
if echo "$RESP" | jq -e '.name == "test-rule"' >/dev/null 2>&1; then
  pass "POST /firewall/rules creates rule"
else
  fail "POST /firewall/rules failed: $RESP"
fi

RESP=$(api_get "/firewall/rules")
RULE_COUNT=$(echo "$RESP" | jq 'length')
if [ "$RULE_COUNT" -ge 1 ]; then
  pass "GET /firewall/rules returns rules"
else
  fail "GET /firewall/rules empty"
fi

RESP=$(api_delete "/firewall/rules/0")
if echo "$RESP" | jq -e '.status == "ok"' >/dev/null 2>&1; then
  pass "DELETE /firewall/rules/0 succeeds"
else
  fail "DELETE /firewall/rules/0 failed: $RESP"
fi

# Test 5: PATCH /system partial update.
echo "[5] PATCH /system partial update"
api_patch() {
  $CURL -X PATCH -H "Authorization: Bearer $TOKEN" -H "Content-Type: application/json" -d "$2" "$BASE_URL/api/v1$1"
}
RESP=$(api_patch "/system" '{"hostname":"test-router"}')
if echo "$RESP" | jq -e '.hostname == "test-router"' >/dev/null 2>&1; then
  pass "PATCH /system updates hostname"
else
  fail "PATCH /system failed: $RESP"
fi

# Restore hostname.
api_patch "/system" '{"hostname":"scoututm"}' >/dev/null

# Test 6: Protected zones.
echo "[6] Protected zones"
RESP=$(api_delete "/network/zones/wan")
if echo "$RESP" | jq -e '.error' >/dev/null 2>&1; then
  pass "DELETE /network/zones/wan rejected"
else
  fail "DELETE /network/zones/wan should be rejected"
fi

# Test 7: Logout.
echo "[7] Logout"
RESP=$($CURL -X POST -H "Authorization: Bearer $TOKEN" "$BASE_URL/api/v1/auth/logout")
if echo "$RESP" | jq -e '.status == "ok"' >/dev/null 2>&1; then
  pass "logout successful"
else
  fail "logout failed: $RESP"
fi

# Verify token is invalid after logout.
RESP=$(api_get "/firewall")
if echo "$RESP" | jq -e '.error == "unauthorized"' >/dev/null 2>&1; then
  pass "token invalid after logout"
else
  fail "token still valid after logout"
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
