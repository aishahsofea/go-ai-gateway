#!/bin/bash

echo "Timeout Testing with Command Line Configuration"
echo "-----------------------------------------------"

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
PURPLE='\033[0;35m'
NC='\033[0m'

BASE_URL="http://localhost:8080"
DATABASE_URL="postgres://gateway_user:gateway_password@localhost:5433/gateway?sslmode=disable"


start_gateway_with_timeouts() {
    local request_timeout=$1
    local backend_timeout=$2
    local connect_timeout=${3:-2s}
    local test_name=$4

    echo -e "${YELLOW} 🚀 Starting gateway for $test_name${NC}"
    echo -e "${YELLOW} Timeouts: Request=${request_timeout}, Backend=${backend_timeout}, Connect=${connect_timeout}${NC}"

    # Stop existing gateway
    docker compose stop gateway >/dev/null 2>&1

    # Build gateway binary
    echo -e "${BLUE} Building gateway...${NC}"
    go build -o ./gateway ./cmd/gateway/ || {
        echo -e "${RED} ❌ Failed to build gateway${NC}"
        return 1
    }

    # Start gateway with custom timeouts in background
    echo -e "${BLUE} Starting gateway with custom timeouts...${NC}"
    DATABASE_URL="$DATABASE_URL" ./gateway \
        --use-localhost \
        --request-timeout="$request_timeout" \
        --backend-timeout="$backend_timeout" \
        --connect-timeout="$connect_timeout" > gateway_test.log 2>&1 &

    GATEWAY_PID=$!
    echo $GATEWAY_PID > gateway_test.pid

    # Wait for gateway to start
    echo -e "${YELLOW} Waiting for gateway to start...${NC}"
    for i in {1..10}; do 
        if curl -s "$BASE_URL/health" >/dev/null 2>&1; then
            echo -e "${GREEN} ✅ Gateway started successfully (attempt $i)${NC}"
            return 0
        fi
        sleep 1
    done

    echo -e "${RED} ❌ Gateway failed to start after 10 seconds${NC}"
    echo "Last few lines of gateway log:"
    tail -5 gateway_test.log
    return 1
}

stop_custom_gateway() {
    if [ -f gateway_test.pid ]; then
        echo -e "${YELLOW} 🛑 Stopping custom gateway...${NC}"
        kill $(cat gateway_test.pid) 2>/dev/null
        rm gateway_test.pid
        sleep 2
    fi
}

run_timed_test() {
    local description=$1
    local url=$2
    local expected_max_time_ms=$3
    local expected_status=${4:-200}

    echo -e "${BLUE} 🧪 Test: $description${NC}"

    start_time=$(date +%s%N)
    response=$(curl -s -w "\nHTTP_STATUS:%{http_code}\nTIME_TOTAL:%{time_total}" "$url")
    end_time=$(date +%s%N)

    duration_ms=$((($end_time - $start_time) / 1000000))
    status=$(echo "$response" | sed -n 's/.*HTTP_STATUS:\([0-9]*\).*/\1/p')
    time_total=$(echo "$response" | sed -n 's/.*TIME_TOTAL:\([0-9.]*\).*/\1/p')

    echo " Duration: ${duration_ms}ms"
    echo " Status: $status"
    echo " Curl time: ${time_total} seconds"

    # Check timing
    if [ $duration_ms -le $expected_max_time_ms ]; then 
        echo -e "  ${GREEN} ✅ Timing: Completed within expected limit (<=${expected_max_time_ms}ms)${NC}" 
    else
        echo -e "  ${RED} ❌ Timing:Exceeded expected time limit (${expected_max_time_ms}ms)${NC}" 
    fi

    # Check status
    if [ "$status" = "$expected_status" ]; then
        echo -e "  ${GREEN} ✅ Status: Received expected status code: $expected_status${NC}" 
    else
        echo -e "  ${RED} ❌ Status: Unexpected status code: $status (expected $expected_status)${NC}" 
    fi

    echo ""
}

cleanup() {
    echo -e "${YELLOW} Cleaning up...${NC}"
    stop_custom_gateway

    echo -e "${BLUE} Restarting normal gateway...${NC}"
    docker compose up -d gateway >/dev/null 2>&1

    # rm -f gateway_test.log gateway ./gateway

    echo -e "${GREEN}✅ Cleanup complete.${NC}"
}

test_with_slow_backend() {
    echo -e "${PURPLE}🐌 Testing with artificially slow backend...${NC}"

    curl -s "http://localhost:8001/admin/fail" >/dev/null 2>&1

    run_timed_test "Slow backend with short timeout" "${BASE_URL}/api/users/123" 2000 503

    curl -s "http://localhost:8001/admin/recover" >/dev/null 2>&1
}

main() {
    echo -e "${PURPLE}Starting comprehensive timeout testing...${NC}"
    echo -e "${YELLOW}This will test different timeout configurations using command-line args${NC}"
    echo ""

    echo -e "${BLUE}🔧 Ensuring mock services are running...${NC}"
    if ! curl -s "http://localhost:8001/admin/status" >/dev/null 2>&1; then
        echo -e "${RED}❌ Mock backend service is not running. Please start with: go run cmd/test-runner/main.go${NC}"
        exit 1
    fi

    trap cleanup EXIT

    # Test 1: Normal timeouts
    echo -e "${YELLOW}=== TEST 1: Normal Operation ===${NC}"
    start_gateway_with_timeouts "30s" "5s" "2s" "Normal Operation" || exit 1
    run_timed_test "Normal request with standard timeouts" "$BASE_URL/api/users/123" 6000 200
    stop_custom_gateway
    echo ""

    # Test 2: Short backend timeout
    echo -e "${YELLOW}=== TEST 2: Short Backend Timeout ===${NC}"
    start_gateway_with_timeouts "30s" "1s" "2s" "Short Backend Timeout" || exit 1
    test_with_slow_backend
    stop_custom_gateway
    echo ""

    # Test 3: Very short request timeout
    echo -e "${YELLOW}=== TEST 3: Very Short Request Timeout ===${NC}"
    start_gateway_with_timeouts "3s" "5s" "2s" "Very Short Request Timeout" || exit 1
    run_timed_test "Request with 3s total timeout" "$BASE_URL/api/users/123" 4000 200
    stop_custom_gateway
    echo ""

    # Test 4: Retry behavior with timeouts
    echo -e "${YELLOW}=== TEST 4: Retry + Timeout Interaction ===${NC}"
    start_gateway_with_timeouts "10s" "1s" "2s" "Retry with timeouts" || exit 1

    echo -e "${PURPLE}Testing retry behavior with short backend timeouts...${NC}"
    run_timed_test "Retry with 1s backend timeout (should try multiple backends)" "$BASE_URL/api/users/123" 5000 200
    stop_custom_gateway
    echo ""

    # Test 5: Extreme timeout test
    echo -e "${YELLOW}=== TEST 5: Extreme Short Timeout ===${NC}"
    start_gateway_with_timeouts "1s" "500ms" "1s" "Extreme Short Timeouts" || exit 1
    run_timed_test "Extreme short timeout test" "$BASE_URL/api/users/123" 2000
    stop_custom_gateway
    echo ""


    echo -e "${GREEN}🎉 All timeout tests completed!${NC}"
    echo -e "${YELLOW}📊 Summary:${NC}"
    echo -e "${YELLOW}  - Tested normal operation with standard timeouts${NC}"
    echo -e "${YELLOW}  - Verified backend timeout behavior${NC}"
    echo -e "${YELLOW}  - Tested request timeout limits${NC}"
    echo -e "${YELLOW}  - Verified retry + timeout interaction${NC}"
    echo -e "${YELLOW}  - Tested extreme timeout scenarios${NC}"
    echo ""
    echo -e "${BLUE}💡 Check gateway_test.log for detailed timeout behavior${NC}"


}

chmod +x "$0"
main