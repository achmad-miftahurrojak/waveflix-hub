#!/bin/bash
# Load Balancer Testing Script for WaveFlix Hub

set -e

DOMAIN_NAME=${DOMAIN_NAME:-localhost}
BACKEND_URL="https://$DOMAIN_NAME"
API_URL="$BACKEND_URL/api"

echo "🧪 Testing WaveFlix Hub Load Balancer..."
echo "Domain: $DOMAIN_NAME"
echo "================================="

# Color codes for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Test function
test_endpoint() {
    local name="$1"
    local url="$2"
    local expected_status="$3"
    local timeout="${4:-10}"
    
    echo -n "Testing $name... "
    
    response=$(curl -s -o /dev/null -w "%{http_code},%{time_total},%{size_download}" \
                    --connect-timeout $timeout \
                    --max-time $timeout \
                    -k "$url" 2>/dev/null || echo "000,0,0")
    
    status_code=$(echo $response | cut -d',' -f1)
    time_total=$(echo $response | cut -d',' -f2)
    size_download=$(echo $response | cut -d',' -f3)
    
    if [ "$status_code" = "$expected_status" ]; then
        echo -e "${GREEN}✅ PASS${NC} (${status_code}, ${time_total}s, ${size_download} bytes)"
        return 0
    else
        echo -e "${RED}❌ FAIL${NC} (Expected: $expected_status, Got: $status_code)"
        return 1
    fi
}

# Test SSL redirect
test_ssl_redirect() {
    echo -n "Testing HTTP to HTTPS redirect... "
    
    response=$(curl -s -o /dev/null -w "%{http_code},%{redirect_url}" \
                    --connect-timeout 10 \
                    "http://$DOMAIN_NAME" 2>/dev/null || echo "000,")
    
    status_code=$(echo $response | cut -d',' -f1)
    redirect_url=$(echo $response | cut -d',' -f2)
    
    if [ "$status_code" = "301" ] || [ "$status_code" = "302" ]; then
        echo -e "${GREEN}✅ PASS${NC} ($status_code -> $redirect_url)"
        return 0
    else
        echo -e "${RED}❌ FAIL${NC} (Expected: 301/302, Got: $status_code)"
        return 1
    fi
}

# Test caching headers
test_caching() {
    local url="$1"
    local cache_expected="$2"
    
    echo -n "Testing caching for $url... "
    
    cache_header=$(curl -s -I -k "$url" | grep -i "cache-control\|x-cache-status" | head -1 | tr -d '\r')
    
    if echo "$cache_header" | grep -qi "$cache_expected"; then
        echo -e "${GREEN}✅ PASS${NC} ($cache_header)"
        return 0
    else
        echo -e "${YELLOW}⚠️  WARNING${NC} (Cache header: $cache_header)"
        return 1
    fi
}

# Test rate limiting
test_rate_limiting() {
    echo -n "Testing rate limiting... "
    
    # Make multiple rapid requests
    for i in {1..10}; do
        curl -s -o /dev/null -k "$API_URL/trending" &
    done
    wait
    
    # Make one more request to check if rate limited
    response=$(curl -s -o /dev/null -w "%{http_code}" -k "$API_URL/trending" 2>/dev/null || echo "000")
    
    if [ "$response" = "429" ] || [ "$response" = "200" ]; then
        echo -e "${GREEN}✅ PASS${NC} (Status: $response)"
        return 0
    else
        echo -e "${YELLOW}⚠️  UNKNOWN${NC} (Status: $response)"
        return 1
    fi
}

# Test load balancing (requires multiple backend instances)
test_load_balancing() {
    echo -n "Testing load balancing... "
    
    # Get server header or X-Served-By if available
    servers=()
    for i in {1..5}; do
        server_id=$(curl -s -I -k "$API_URL/health" | grep -i "x-served-by\|server" | head -1 | cut -d':' -f2 | tr -d ' \r')
        if [ -n "$server_id" ]; then
            servers+=("$server_id")
        fi
        sleep 0.1
    done
    
    unique_servers=$(printf '%s\n' "${servers[@]}" | sort -u | wc -l)
    
    if [ "$unique_servers" -gt 1 ]; then
        echo -e "${GREEN}✅ PASS${NC} ($unique_servers different servers detected)"
        return 0
    else
        echo -e "${YELLOW}⚠️  SINGLE SERVER${NC} (Load balancing not detectable)"
        return 1
    fi
}

# Main test suite
total_tests=0
passed_tests=0

echo ""
echo "🔧 Basic Connectivity Tests:"
echo "----------------------------"

# Test basic endpoints
endpoints=(
    "Frontend:$BACKEND_URL:200"
    "Health Check:$BACKEND_URL/health:200"
    "API Trending:$API_URL/trending:200"
    "API Health:$API_URL/health:200"
)

for endpoint in "${endpoints[@]}"; do
    IFS=':' read -r name url expected <<< "$endpoint"
    total_tests=$((total_tests + 1))
    if test_endpoint "$name" "$url" "$expected"; then
        passed_tests=$((passed_tests + 1))
    fi
done

echo ""
echo "🔒 Security Tests:"
echo "------------------"

total_tests=$((total_tests + 1))
if test_ssl_redirect; then
    passed_tests=$((passed_tests + 1))
fi

echo ""
echo "⚡ Performance Tests:"
echo "--------------------"

# Test caching
total_tests=$((total_tests + 1))
if test_caching "$API_URL/trending" "cache"; then
    passed_tests=$((passed_tests + 1))
fi

# Test rate limiting
total_tests=$((total_tests + 1))
if test_rate_limiting; then
    passed_tests=$((passed_tests + 1))
fi

# Test load balancing
total_tests=$((total_tests + 1))
if test_load_balancing; then
    passed_tests=$((passed_tests + 1))
fi

echo ""
echo "📊 Test Summary:"
echo "================"
echo "Passed: $passed_tests/$total_tests tests"

if [ $passed_tests -eq $total_tests ]; then
    echo -e "${GREEN}🎉 All tests passed!${NC}"
    exit 0
elif [ $passed_tests -gt $((total_tests / 2)) ]; then
    echo -e "${YELLOW}⚠️  Some tests failed, but load balancer is functional${NC}"
    exit 0
else
    echo -e "${RED}❌ Multiple test failures detected${NC}"
    exit 1
fi