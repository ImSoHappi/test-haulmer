#!/usr/bin/env bash
set -euo pipefail

# Colors for clear terminal reporting
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

API_BASE_URL="${API_BASE_URL:-http://localhost:8080}"
DASHBOARD_BASE_URL="${DASHBOARD_BASE_URL:-http://localhost:3000}"

echo -e "${CYAN}=====================================================${NC}"
echo -e "${CYAN}   HAULMER PAYMENT GATEWAY - END-TO-END SUITE       ${NC}"
echo -e "${CYAN}=====================================================${NC}"
echo -e "Testing target API: ${YELLOW}${API_BASE_URL}${NC}"
echo ""

# Helper for testing assertions
assert_equal() {
    local expected="$1"
    local actual="$2"
    local msg="$3"
    if [ "$expected" != "$actual" ]; then
        echo -e "${RED}✗ FAIL: ${msg}${NC} (Expected: '$expected', Got: '$actual')"
        exit 1
    else
        echo -e "${GREEN}✓ PASS: ${msg}${NC}"
    fi
}

assert_contains() {
    local haystack="$1"
    local needle="$2"
    local msg="$3"
    if [[ "$haystack" != *"$needle"* ]]; then
        echo -e "${RED}✗ FAIL: ${msg}${NC} (Expected to contain: '$needle')"
        echo "Response was: $haystack"
        exit 1
    else
        echo -e "${GREEN}✓ PASS: ${msg}${NC}"
    fi
}

# 1. Healthcheck Assertion
echo -e "${YELLOW}Step 1: Checking API Health Readiness...${NC}"
HEALTH_HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" "${API_BASE_URL}/health" || echo "000")
if [ "${HEALTH_HTTP_CODE}" != "200" ]; then
    echo -e "${RED}✗ API is not healthy or unreachable at ${API_BASE_URL}/health (HTTP ${HEALTH_HTTP_CODE})${NC}"
    exit 1
fi
echo -e "${GREEN}✓ API is Healthy (HTTP 200)${NC}"
echo ""

# Generate unique run identifiers
RUN_ID=$(date +%s%N | tail -c 8)
ORDER_APPROVED="ORD${RUN_ID:0:3}"
IDEM_APPROVED="idem-app-${RUN_ID}"

# 2. Scenario 1: Approved Payment (PAN 4242424242424242)
echo -e "${YELLOW}Step 2: Scenario 1 - Approved Payment (Deterministic PAN 4242424242424242)...${NC}"
RESP=$(curl -s -w "\n%{http_code}" -X POST "${API_BASE_URL}/v1/payments" \
    -H "Content-Type: application/json" \
    -H "Idempotency-Key: ${IDEM_APPROVED}" \
    -d "{
        \"merchant_id\": \"1001\",
        \"order_reference\": \"${ORDER_APPROVED}\",
        \"amount\": 35000,
        \"currency\": \"CLP\",
        \"card_number\": \"4242424242424242\",
        \"card_expiry_month\": 12,
        \"card_expiry_year\": 2028,
        \"card_cvv\": \"123\",
        \"card_holder_name\": \"Juan Perez\"
    }")

HTTP_CODE=$(echo "$RESP" | tail -n1)
BODY=$(echo "$RESP" | sed '$d')

assert_equal "201" "$HTTP_CODE" "Scenario 1 returns HTTP 201 Created"
assert_contains "$BODY" '"status":"APPROVED"' "Scenario 1 status is APPROVED"
assert_contains "$BODY" '"authorization_code":"AUT-' "Scenario 1 includes authorization_code"

TX_ID_APPROVED=$(echo "$BODY" | grep -o '"transaction_id":"[^"]*' | cut -d'"' -f4)
echo -e "Transaction created: ${CYAN}${TX_ID_APPROVED}${NC}"
echo ""

# 3. Scenario 2: Insufficient Funds (PAN 4000000000009995 -> Declined 51)
echo -e "${YELLOW}Step 3: Scenario 2 - Insufficient Funds (PAN 4000000000009995)...${NC}"
IDEM_DECLINED="idem-dec-${RUN_ID}"
ORDER_DECLINED="ORD${RUN_ID:3:3}"
RESP=$(curl -s -w "\n%{http_code}" -X POST "${API_BASE_URL}/v1/payments" \
    -H "Content-Type: application/json" \
    -H "Idempotency-Key: ${IDEM_DECLINED}" \
    -d "{
        \"merchant_id\": \"1001\",
        \"order_reference\": \"${ORDER_DECLINED}\",
        \"amount\": 99000,
        \"currency\": \"CLP\",
        \"card_number\": \"4000000000009995\",
        \"card_expiry_month\": 10,
        \"card_expiry_year\": 2028,
        \"card_cvv\": \"456\",
        \"card_holder_name\": \"Maria Silva\"
    }")

HTTP_CODE=$(echo "$RESP" | tail -n1)
BODY=$(echo "$RESP" | sed '$d')

assert_equal "402" "$HTTP_CODE" "Scenario 2 returns HTTP 402 Payment Required (declined)"
assert_contains "$BODY" '"status":"DECLINED"' "Scenario 2 status is DECLINED"
assert_contains "$BODY" '"result_code":"51"' "Scenario 2 result_code is 51 (insufficient funds)"
echo ""

# 4. Scenario 3: Acquirer 503 Flaky Simulation (PAN 4000000000000341 -> Recovers after retries)
echo -e "${YELLOW}Step 4: Scenario 3 - Acquirer 503 Retries (PAN 4000000000000341)...${NC}"
IDEM_RETRY="idem-ret-${RUN_ID}"
ORDER_RETRY="RET${RUN_ID:0:3}"
RESP=$(curl -s -w "\n%{http_code}" -X POST "${API_BASE_URL}/v1/payments" \
    -H "Content-Type: application/json" \
    -H "Idempotency-Key: ${IDEM_RETRY}" \
    -d "{
        \"merchant_id\": \"1001\",
        \"order_reference\": \"${ORDER_RETRY}\",
        \"amount\": 15000,
        \"currency\": \"CLP\",
        \"card_number\": \"4000000000000341\",
        \"card_expiry_month\": 11,
        \"card_expiry_year\": 2027,
        \"card_cvv\": \"789\",
        \"card_holder_name\": \"Carlos Rojas\"
    }")

HTTP_CODE=$(echo "$RESP" | tail -n1)
BODY=$(echo "$RESP" | sed '$d')

assert_equal "201" "$HTTP_CODE" "Scenario 3 recovers and returns HTTP 201"
assert_contains "$BODY" '"status":"APPROVED"' "Scenario 3 status is APPROVED after retries"
echo ""

# 5. Scenario 4: Exact Idempotent Replay
echo -e "${YELLOW}Step 5: Scenario 4 - Exact Idempotent Replay...${NC}"
RESP=$(curl -s -w "\n%{http_code}" -X POST "${API_BASE_URL}/v1/payments" \
    -H "Content-Type: application/json" \
    -H "Idempotency-Key: ${IDEM_APPROVED}" \
    -d "{
        \"merchant_id\": \"1001\",
        \"order_reference\": \"${ORDER_APPROVED}\",
        \"amount\": 35000,
        \"currency\": \"CLP\",
        \"card_number\": \"4242424242424242\",
        \"card_expiry_month\": 12,
        \"card_expiry_year\": 2028,
        \"card_cvv\": \"123\",
        \"card_holder_name\": \"Juan Perez\"
    }")

HTTP_CODE=$(echo "$RESP" | tail -n1)
BODY=$(echo "$RESP" | sed '$d')

assert_equal "200" "$HTTP_CODE" "Scenario 4 replay returns HTTP 200 OK"
assert_contains "$BODY" '"is_replayed":true' "Scenario 4 marks is_replayed as true"
assert_contains "$BODY" "\"transaction_id\":\"${TX_ID_APPROVED}\"" "Scenario 4 returns identical transaction_id"
echo ""

# 6. Scenario 5: Idempotency Key Collision with Altered Amount
echo -e "${YELLOW}Step 6: Scenario 5 - Idempotency Key Collision with Altered Payload...${NC}"
RESP=$(curl -s -w "\n%{http_code}" -X POST "${API_BASE_URL}/v1/payments" \
    -H "Content-Type: application/json" \
    -H "Idempotency-Key: ${IDEM_APPROVED}" \
    -d "{
        \"merchant_id\": \"1001\",
        \"order_reference\": \"${ORDER_APPROVED}\",
        \"amount\": 50000,
        \"currency\": \"CLP\",
        \"card_number\": \"4242424242424242\",
        \"card_expiry_month\": 12,
        \"card_expiry_year\": 2028,
        \"card_cvv\": \"123\",
        \"card_holder_name\": \"Juan Perez\"
    }")

HTTP_CODE=$(echo "$RESP" | tail -n1)
BODY=$(echo "$RESP" | sed '$d')

assert_equal "409" "$HTTP_CODE" "Scenario 5 returns HTTP 409 Conflict"
assert_contains "$BODY" "idempotency_conflict" "Scenario 5 reports idempotency_conflict error code"
echo ""

# 7. Scenario 6: Order Integrity - Physical No Double-Charge Guarantee
echo -e "${YELLOW}Step 7: Scenario 6 - Double Charge Prevention on Already Paid Order...${NC}"
NEW_IDEM="idem-diff-${RUN_ID}"
RESP=$(curl -s -w "\n%{http_code}" -X POST "${API_BASE_URL}/v1/payments" \
    -H "Content-Type: application/json" \
    -H "Idempotency-Key: ${NEW_IDEM}" \
    -d "{
        \"merchant_id\": \"1001\",
        \"order_reference\": \"${ORDER_APPROVED}\",
        \"amount\": 35000,
        \"currency\": \"CLP\",
        \"card_number\": \"4242424242424242\",
        \"card_expiry_month\": 12,
        \"card_expiry_year\": 2028,
        \"card_cvv\": \"123\",
        \"card_holder_name\": \"Juan Perez\"
    }")

HTTP_CODE=$(echo "$RESP" | tail -n1)
BODY=$(echo "$RESP" | sed '$d')

assert_equal "409" "$HTTP_CODE" "Scenario 6 returns HTTP 409 Conflict"
assert_contains "$BODY" "intent_already_approved" "Scenario 6 reports intent_already_approved"
echo ""

# 8. Scenario 7: Multi-Tenant Isolation
echo -e "${YELLOW}Step 8: Scenario 7 - Multi-Tenant Isolation Check...${NC}"
RESP=$(curl -s -w "\n%{http_code}" -X GET "${API_BASE_URL}/v1/payments/${TX_ID_APPROVED}" \
    -H "X-Merchant-ID: 1002")

HTTP_CODE=$(echo "$RESP" | tail -n1)
assert_equal "404" "$HTTP_CODE" "Merchant 1002 cannot read Merchant 1001 transaction (HTTP 404)"
echo ""

echo -e "${GREEN}=====================================================${NC}"
echo -e "${GREEN}   ALL 7 END-TO-END SCENARIOS PASSED SUCCESSFULLY!   ${NC}"
echo -e "${GREEN}=====================================================${NC}"
