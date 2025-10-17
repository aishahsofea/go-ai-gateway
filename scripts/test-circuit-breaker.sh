#!/bin/bash

echo "Testing Circuit Breaker Recovery Cycle"
echo "--------------------------------"

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

BASE_URL="http://localhost:8080"
USER_SERVICE="$BASE_URL/api/users/123"

echo ""
echo "Test Plan:"
echo "1. Make service fail (trigger OPEN state)"
echo "2. Wait for recovery timeout (OPEN -> HALF-OPEN)"
echo "3. Recover service and test successful requests (HALF-OPEN -> CLOSED)"
echo "4. Test failure in HALF-OPEN state (HALF-OPEN -> OPEN)"
echo ""

make_request() {
    local url=$1
    local description=$2

    echo -e "${BLUE}Making request: $description${NC}"
    response=$(curl -s -w "\nHTTP_STATUS:%{http_code}" "$url" 2>/dev/null)
    body=$(echo "$response" | sed '$d')
    status=$(echo "$response" | tail -1 | cut -d: -f2)

    if [ "$status" -ge 200 ] && [ "$status" -lt 300 ]; then
        echo -e "${GREEN}✅ Success ($status)${NC}"
        echo "$body" | jq -r '.headers["X-Backend-URL"]'
    elif [ "$status" -ge 500 ]; then
        echo -e "${RED}❌ Server Error ($status)${NC}"
        echo "$body" | jq -r '.error // "Unknown error"' | sed 's/^/   Error: /'
    else 
        echo -e "${YELLOW}⚠️ Other Status ($status)${NC}"
    fi
    echo ""
}

trigger_failure() {
    echo -e "${RED}🔥 Step 1: Triggering service failure${NC}"
    curl -s "http://localhost:8001/admin/fail" | jq '.'
    echo ""
}

recover_service() {
    echo -e "${GREEN} Recovering service...${NC}"
    curl -s "http://localhost:8001/admin/recover" | jq '.'
    echo ""
}

check_service_status() {
    echo -e "${BLUE} Checking service status${NC}"
    curl -s "http://localhost:8001/admin/status" | jq '.'
    echo ""
}

echo "🚀 Starting Circuit Breaker Test"
echo ""

# Step 1: Trigger service failure to open the circuit breaker
trigger_failure
check_service_status

echo "Making 18 requests to trigger circuit breaker"
for i in {1..18}; do
    echo "Request #$i:"
    make_request "$USER_SERVICE" "Trigger attempt #$i"
done

echo -e "${YELLOW} Circuit breaker should now be OPEN. Waiting 35 seconds for recovery timeout...${NC}"
echo "(Default timeout is 30s, waiting extra 5 seconds to be sure)"

for i in {35..1}; do
    printf "\rTime remaining: %02d seconds" $i
    sleep 1
done

echo ""

# Step 2: Test OPEN -> HALF-OPEN transition
echo -e "${YELLOW} Step 2: Testing OPEN -> HALF-OPEN transition${NC}"
echo "First request after timeout should trigger HALF-OPEN state"
make_request "$USER_SERVICE" "First request after timeout (should still fail but circuit is HALF-OPEN)"

# Step 3: Recover service and test HALF-OPEN -> CLOSED transition
echo -e "${GREEN} Step 3: Recovering service and testing HALF-OPEN -> CLOSED transition${NC}"
recover_service
check_service_status

echo "Making successful requests to trigger HALF-OPEN -> CLOSED:"
for i in {1..3}; do 
    echo "Success request #$i:"
    make_request "$USER_SERVICE" "Recovery test #$i"
    sleep 1
done

# Step 4: Test the system is fully recovered
echo -e "${GREEN} Step 4: Testing fully recovered system${NC}"
echo "Making additional requests to confirm CLOSED state:"
for i in {1..3}; do 
    echo "Confirmation request #$i:"
    make_request "$USER_SERVICE" "Confirmation test #$i"
    sleep 0.5
done

echo ""
echo -e "${GREEN}✅ Circuit Breaker tests completed successfully.${NC}"
echo "🔍 Check the gateway logs to see detailed circuit breaker state transitions!"