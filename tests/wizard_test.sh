#!/bin/sh
# Integration test: Setup Wizard flow
# Run against dev-mode scoutd with a fresh config (setup_done=false).
set -e

PASS=0
FAIL=0
BASE_URL="${SCOUT_URL:-https://localhost:8443}"
CURL="curl -sk"

pass() { PASS=$((PASS + 1)); echo "  PASS: $1"; }
fail() { FAIL=$((FAIL + 1)); echo "  FAIL: $1 (got: $2)"; }

check_code() {
  local desc="$1" expected="$2" actual="$3"
  if [ "$actual" = "$expected" ]; then pass "$desc"; else fail "$desc" "HTTP $actual"; fi
}

echo "=== Wizard Integration Test ==="
echo "Target: $BASE_URL"
echo ""

# Authenticate first (test config should have password "testpassword1").
TOKEN=$($CURL -X POST "$BASE_URL/api/v1/auth/login" \
  -H "Content-Type: application/json" \
  -d '{"password":"testpassword1"}' 2>/dev/null | grep -o '"token":"[^"]*"' | cut -d'"' -f4)

if [ -z "$TOKEN" ]; then
  echo "SKIP: Cannot authenticate. Ensure scoutd is running with a known password."
  exit 0
fi
AUTH="Authorization: Bearer $TOKEN"

# Verify setup_done is false before starting.
echo "[0] Pre-check"
SETUP=$($CURL -H "$AUTH" "$BASE_URL/api/v1/system" 2>/dev/null)
if echo "$SETUP" | grep -q '"setup_done":false'; then
  pass "setup_done is false (fresh config)"
else
  echo "SKIP: setup_done is already true. Use a fresh config to test wizard."
  exit 0
fi

# Step 1: Welcome
echo "[1] Welcome"
CODE=$($CURL -o /dev/null -w '%{http_code}' -X POST "$BASE_URL/api/v1/wizard/step/1" -H "$AUTH")
check_code "Step 1 returns 200" "200" "$CODE"

# Step 2: Set password
echo "[2] Set password"
CODE=$($CURL -o /dev/null -w '%{http_code}' -X POST "$BASE_URL/api/v1/wizard/step/2" \
  -H "$AUTH" -H "Content-Type: application/json" \
  -d '{"password":"securepassword1"}')
check_code "Step 2 returns 200" "200" "$CODE"

# Step 2: Short password should fail (validation test)
CODE=$($CURL -o /dev/null -w '%{http_code}' -X POST "$BASE_URL/api/v1/wizard/step/2" \
  -H "$AUTH" -H "Content-Type: application/json" \
  -d '{"password":"short"}')
check_code "Step 2 rejects short password" "400" "$CODE"

# Step 3: Network detection
echo "[3] Network detection"
RESP=$($CURL -X POST "$BASE_URL/api/v1/wizard/step/3" -H "$AUTH")
if echo "$RESP" | grep -q "interfaces"; then
  pass "Step 3 returns interfaces"
else
  fail "Step 3 missing interfaces" "$RESP"
fi

# Step 4: LAN config
echo "[4] LAN config"
CODE=$($CURL -o /dev/null -w '%{http_code}' -X POST "$BASE_URL/api/v1/wizard/step/4" \
  -H "$AUTH" -H "Content-Type: application/json" \
  -d '{"lan_address":"192.168.1.1/24"}')
check_code "Step 4 returns 200" "200" "$CODE"

# Step 5: Security defaults
echo "[5] Security defaults"
CODE=$($CURL -o /dev/null -w '%{http_code}' -X POST "$BASE_URL/api/v1/wizard/step/5" \
  -H "$AUTH" -H "Content-Type: application/json" \
  -d '{"ids_enabled":true,"ids_mode":"ids","block_malware":true,"block_phishing":true}')
check_code "Step 5 returns 200" "200" "$CODE"

# Step 6: Skip VLANs
echo "[6] VLAN setup (skip)"
CODE=$($CURL -o /dev/null -w '%{http_code}' -X POST "$BASE_URL/api/v1/wizard/step/6" \
  -H "$AUTH" -H "Content-Type: application/json" \
  -d '{"vlans":[],"zones":[]}')
check_code "Step 6 returns 200" "200" "$CODE"

# Step 7: Apply
echo "[7] Apply"
CODE=$($CURL -o /dev/null -w '%{http_code}' -X POST "$BASE_URL/api/v1/wizard/step/7" \
  -H "$AUTH" -H "Content-Type: application/json" -d '{}')
check_code "Step 7 returns 200" "200" "$CODE"

# Verify setup_done flipped
echo "[8] Verify setup_done"
RESP=$($CURL -H "$AUTH" "$BASE_URL/api/v1/system")
if echo "$RESP" | grep -q '"setup_done":true'; then
  pass "setup_done is now true"
else
  fail "setup_done not set" "$RESP"
fi

# Wizard should return 403 after setup
echo "[9] Wizard blocked after setup"
CODE=$($CURL -o /dev/null -w '%{http_code}' -X POST "$BASE_URL/api/v1/wizard/step/1" -H "$AUTH")
check_code "Wizard returns 403 after setup" "403" "$CODE"

echo ""
echo "=== Results: $PASS passed, $FAIL failed ==="
[ "$FAIL" -eq 0 ] && exit 0 || exit 1
