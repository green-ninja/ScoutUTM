#!/bin/sh
# Parental controls integration tests — run against a running scoutd instance in Vagrant VM.
# Tests: profile CRUD, device assignment with static IP, unbound config rendering, dnsmasq static hosts.
set -e

PASS=0
FAIL=0
BASE_URL="${SCOUT_URL:-https://localhost:443}"
CURL="curl -sk"

pass() { PASS=$((PASS + 1)); echo "  PASS: $1"; }
fail() { FAIL=$((FAIL + 1)); echo "  FAIL: $1"; }

echo "=== Parental Controls Integration Tests ==="
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

# Test 1: Create a parental profile.
echo "[1] Create parental profile"
RESP=$(api_post "/parental/profiles" '{
  "name": "Kids",
  "block_categories": ["adult", "gambling"],
  "safe_search": true,
  "youtube_restricted": true,
  "schedule": {"enabled": true, "start_time": "21:00", "end_time": "07:00", "days": []}
}')
PROFILE_ID=$(echo "$RESP" | jq -r '.id // empty')
if [ -n "$PROFILE_ID" ]; then
  pass "created profile Kids (id=$PROFILE_ID)"
else
  fail "failed to create profile: $RESP"
fi

# Test 2: List profiles — should have our new profile.
echo "[2] List profiles"
RESP=$(api_get "/parental/profiles")
COUNT=$(echo "$RESP" | jq 'length')
if [ "$COUNT" -ge 1 ]; then
  pass "profile list has $COUNT profile(s)"
else
  fail "expected at least 1 profile, got $COUNT"
fi

# Test 3: Get devices — check they're available.
echo "[3] List devices"
RESP=$(api_get "/devices")
DEVICE_COUNT=$(echo "$RESP" | jq 'length')
if [ "$DEVICE_COUNT" -ge 1 ]; then
  pass "found $DEVICE_COUNT device(s)"
else
  fail "no devices found (need at least 1 for assignment test)"
fi

# Get first device MAC and IP.
FIRST_MAC=$(echo "$RESP" | jq -r '.[0].mac // empty')
FIRST_IP=$(echo "$RESP" | jq -r '.[0].ip // empty')

# Test 4: Assign device to profile.
echo "[4] Assign device to profile"
if [ -n "$FIRST_MAC" ] && [ -n "$PROFILE_ID" ]; then
  RESP=$(api_put "/devices/$FIRST_MAC" "{\"profile_id\":\"$PROFILE_ID\"}")
  ASSIGNED_PROFILE=$(echo "$RESP" | jq -r '.profile_id // empty')
  if [ "$ASSIGNED_PROFILE" = "$PROFILE_ID" ]; then
    pass "device $FIRST_MAC assigned to profile $PROFILE_ID"
  else
    fail "device profile_id = $ASSIGNED_PROFILE, expected $PROFILE_ID"
  fi
else
  fail "no device or profile available for assignment"
fi

# Test 5: Check config has device with static IP.
echo "[5] Config has device with IP"
RESP=$(api_get "/parental")
DEVICE_IP=$(echo "$RESP" | jq -r '.devices[0].ip // empty')
if [ -n "$DEVICE_IP" ]; then
  pass "parental device has IP: $DEVICE_IP"
else
  fail "parental device has no IP set"
fi

# Test 6: Check dnsmasq.conf has dhcp-host for static reservation.
echo "[6] dnsmasq.conf has static DHCP host"
if [ -f /etc/dnsmasq.conf ]; then
  if grep -q "dhcp-host=" /etc/dnsmasq.conf; then
    pass "dnsmasq.conf has dhcp-host line"
  else
    fail "dnsmasq.conf missing dhcp-host line"
  fi
else
  fail "dnsmasq.conf not found"
fi

# Test 7: Check unbound.conf has access-control-tag.
echo "[7] unbound.conf has parental tags"
if [ -f /etc/unbound/unbound.conf ]; then
  if grep -q "access-control-tag:" /etc/unbound/unbound.conf; then
    pass "unbound.conf has access-control-tag"
  else
    fail "unbound.conf missing access-control-tag"
  fi
  if grep -q "define-tag:" /etc/unbound/unbound.conf; then
    pass "unbound.conf has define-tag"
  else
    fail "unbound.conf missing define-tag"
  fi
else
  fail "unbound.conf not found"
fi

# Test 8: Check unbound.conf has safe search overrides.
echo "[8] unbound.conf has safe search"
if [ -f /etc/unbound/unbound.conf ]; then
  if grep -q "forcesafesearch.google.com" /etc/unbound/unbound.conf; then
    pass "unbound.conf has Google safe search CNAME"
  else
    fail "unbound.conf missing Google safe search CNAME"
  fi
else
  fail "unbound.conf not found"
fi

# Test 9: Check unbound.conf has YouTube restricted.
echo "[9] unbound.conf has YouTube restricted"
if [ -f /etc/unbound/unbound.conf ]; then
  if grep -q "restrict.youtube.com" /etc/unbound/unbound.conf; then
    pass "unbound.conf has YouTube restricted mode CNAME"
  else
    fail "unbound.conf missing YouTube restricted mode CNAME"
  fi
else
  fail "unbound.conf not found"
fi

# Test 10: Check unbound.conf has parental RPZ zones.
echo "[10] unbound.conf has parental RPZ zones"
if [ -f /etc/unbound/unbound.conf ]; then
  if grep -q "rpz-parental-" /etc/unbound/unbound.conf; then
    pass "unbound.conf has parental RPZ zones"
  else
    fail "unbound.conf missing parental RPZ zones"
  fi
else
  fail "unbound.conf not found"
fi

# Test 11: Device has manufacturer field.
echo "[11] Device has manufacturer field"
RESP=$(api_get "/devices")
# Just check the field exists (may be empty for synthetic dev-mode MACs).
HAS_FIELD=$(echo "$RESP" | jq '.[0] | has("manufacturer")')
if [ "$HAS_FIELD" = "true" ]; then
  pass "device response includes manufacturer field"
else
  fail "device response missing manufacturer field"
fi

# Test 12: DNS query log endpoint works.
echo "[12] DNS query log endpoint"
RESP=$(api_get "/dns/query-log")
if echo "$RESP" | jq -e 'type == "array"' >/dev/null 2>&1; then
  pass "dns/query-log returns array"
else
  fail "dns/query-log did not return array: $RESP"
fi

# Cleanup: delete the test profile.
echo "[cleanup] Delete test profile"
if [ -n "$PROFILE_ID" ]; then
  api_delete "/parental/profiles/$PROFILE_ID" >/dev/null 2>&1
  pass "cleaned up test profile"
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
