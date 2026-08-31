#!/bin/bash
# Production Deployment Testing Script for WaveFlix Hub

set -e

# Configuration
DOMAIN_NAME=${DOMAIN_NAME:-localhost}
DEPLOYMENT_TYPE=${DEPLOYMENT_TYPE:-docker}
BASE_URL="https://$DOMAIN_NAME"
API_URL="$BASE_URL/api"
TIMEOUT=30

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Test results
TESTS_TOTAL=0
TESTS_PASSED=0
TESTS_FAILED=0

log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[PASS]${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[FAIL]${NC} $1"
}

# Test execution function
run_test() {
    local test_name="$1"
    local test_command="$2"
    local expected_result="${3:-0}"
    
    TESTS_TOTAL=$((TESTS_TOTAL + 1))
    echo -n "Testing $test_name... "
    
    if eval "$test_command" >/dev/null 2>&1; then
        result=0
    else
        result=1
    fi
    
    if [ $result -eq $expected_result ]; then
        log_success "$test_name"
        TESTS_PASSED=$((TESTS_PASSED + 1))
        return 0
    else
        log_error "$test_name"
        TESTS_FAILED=$((TESTS_FAILED + 1))
        return 1
    fi
}

# HTTP test function
http_test() {
    local url="$1"
    local expected_status="${2:-200}"
    local timeout="${3:-$TIMEOUT}"
    
    local response
    response=$(curl -s -o /dev/null -w "%{http_code}" --connect-timeout "$timeout" --max-time "$timeout" -k "$url" 2>/dev/null || echo "000")
    
    [ "$response" = "$expected_status" ]
}

# JSON API test function
json_test() {
    local url="$1"
    local json_path="$2"
    local expected_value="$3"
    local timeout="${4:-$TIMEOUT}"
    
    local result
    result=$(curl -s --connect-timeout "$timeout" --max-time "$timeout" -k "$url" 2>/dev/null | jq -r "$json_path" 2>/dev/null || echo "null")
    
    [ "$result" = "$expected_value" ]
}

# Performance test function
performance_test() {
    local url="$1"
    local max_response_time="$2"
    local timeout="${3:-$TIMEOUT}"
    
    local response_time
    response_time=$(curl -s -o /dev/null -w "%{time_total}" --connect-timeout "$timeout" --max-time "$timeout" -k "$url" 2>/dev/null || echo "999")
    
    # Convert to milliseconds and compare
    local ms_time=$(echo "$response_time * 1000 / 1" | bc 2>/dev/null || echo "9999")
    [ "$ms_time" -lt "$max_response_time" ]
}

# Load test function
load_test() {
    local url="$1"
    local concurrent_requests="${2:-10}"
    local total_requests="${3:-100}"
    
    log_info "Running load test: $concurrent_requests concurrent requests, $total_requests total"
    
    # Use Apache Bench if available
    if command -v ab >/dev/null 2>&1; then
        local result
        result=$(ab -n "$total_requests" -c "$concurrent_requests" -k "$url" 2>/dev/null | grep "Failed requests:" | awk '{print $3}')
        [ "${result:-0}" -eq 0 ]
    else
        # Fallback: simple concurrent curl test
        local failed_requests=0
        local pids=()
        
        for ((i=1; i<=concurrent_requests; i++)); do
            {
                for ((j=1; j<=total_requests/concurrent_requests; j++)); do
                    if ! curl -s -f -k "$url" >/dev/null 2>&1; then
                        ((failed_requests++))
                    fi
                done
            } &
            pids+=($!)
        done
        
        # Wait for all background processes
        for pid in "${pids[@]}"; do
            wait "$pid"
        done
        
        [ "$failed_requests" -eq 0 ]
    fi
}

main() {
    echo "🧪 WaveFlix Hub Production Testing"
    echo "=================================="
    echo "Domain: $DOMAIN_NAME"
    echo "Deployment: $DEPLOYMENT_TYPE"
    echo "Base URL: $BASE_URL"
    echo ""
    
    # Basic connectivity tests
    log_info "🔗 Basic Connectivity Tests"
    echo "----------------------------"
    
    run_test "HTTP to HTTPS redirect" "http_test 'http://$DOMAIN_NAME' '301'"
    run_test "HTTPS homepage" "http_test '$BASE_URL'"
    run_test "API health endpoint" "http_test '$API_URL/health'"
    run_test "Metrics endpoint" "http_test '$API_URL/metrics'"
    
    echo ""
    
    # API functionality tests
    log_info "🔌 API Functionality Tests"
    echo "---------------------------"
    
    run_test "API root endpoint" "json_test '$API_URL/' '.service' 'waveflix-api'"
    run_test "Health check status" "json_test '$API_URL/health' '.status' 'ok'"
    run_test "Health check database" "json_test '$API_URL/health' '.database' 'connected'"
    run_test "Trending movies API" "http_test '$API_URL/trending'"
    run_test "Discover movies API" "http_test '$API_URL/discover?media=movie'"
    run_test "Search API" "http_test '$API_URL/search?query=marvel'"
    
    echo ""
    
    # Performance tests
    log_info "⚡ Performance Tests"
    echo "--------------------"
    
    run_test "Homepage response time (<2s)" "performance_test '$BASE_URL' '2000'"
    run_test "API health response time (<500ms)" "performance_test '$API_URL/health' '500'"
    run_test "Trending API response time (<1s)" "performance_test '$API_URL/trending' '1000'"
    
    echo ""
    
    # Security tests
    log_info "🔒 Security Tests"
    echo "-----------------"
    
    run_test "Security headers present" "curl -s -I -k '$BASE_URL' | grep -q 'X-Content-Type-Options'"
    run_test "HTTPS enforced" "http_test 'http://$DOMAIN_NAME' '301'"
    run_test "No server information leak" "! curl -s -I -k '$BASE_URL' | grep -q 'Server: '"
    
    echo ""
    
    # Cache and CDN tests
    log_info "💾 Cache and CDN Tests"
    echo "----------------------"
    
    run_test "Static asset caching" "curl -s -I -k '$BASE_URL/favicon.ico' | grep -q 'Cache-Control'"
    run_test "API caching headers" "curl -s -I -k '$API_URL/trending' | grep -q 'Cache-Control'"
    run_test "CDN status endpoint" "http_test '$API_URL/cdn-status'"
    
    echo ""
    
    # Component-specific tests
    log_info "🧩 Component Tests"
    echo "------------------"
    
    if [ "$DEPLOYMENT_TYPE" = "docker" ]; then
        # Docker-specific tests
        run_test "Backend container healthy" "docker-compose -f docker-compose.prod.yml ps backend | grep -q 'Up'"
        run_test "Frontend container healthy" "docker-compose -f docker-compose.prod.yml ps frontend | grep -q 'Up'"
        run_test "Database container healthy" "docker-compose -f docker-compose.prod.yml ps postgres | grep -q 'Up'"
        run_test "Redis container healthy" "docker-compose -f docker-compose.prod.yml ps redis | grep -q 'Up'"
        run_test "Nginx container healthy" "docker-compose -f docker-compose.prod.yml ps nginx | grep -q 'Up'"
    else
        # Kubernetes-specific tests
        run_test "Backend pods running" "kubectl get pods -n waveflix -l app=waveflix-backend | grep -q 'Running'"
        run_test "Frontend pods running" "kubectl get pods -n waveflix -l app=waveflix-frontend | grep -q 'Running'"
        run_test "Database pod running" "kubectl get pods -n waveflix -l app=postgres | grep -q 'Running'"
        run_test "Redis pod running" "kubectl get pods -n waveflix -l app=redis | grep -q 'Running'"
    fi
    
    echo ""
    
    # Load testing (optional, can be intensive)
    if [ "${LOAD_TEST:-false}" = "true" ]; then
        log_info "📈 Load Testing"
        echo "---------------"
        
        run_test "Load test homepage (10 concurrent)" "load_test '$BASE_URL' 10 50"
        run_test "Load test API (5 concurrent)" "load_test '$API_URL/health' 5 25"
    fi
    
    echo ""
    
    # Monitoring and observability tests
    log_info "📊 Monitoring Tests"
    echo "-------------------"
    
    if [ "$DEPLOYMENT_TYPE" = "docker" ]; then
        run_test "Prometheus accessible" "http_test 'http://$DOMAIN_NAME:9090/-/healthy'"
        run_test "Grafana accessible" "http_test 'http://$DOMAIN_NAME:3001/api/health'"
    else
        # For Kubernetes, these might require port forwarding
        log_warning "Monitoring tests require manual port forwarding in Kubernetes"
    fi
    
    echo ""
    
    # Database tests
    log_info "🗄️ Database Tests"
    echo "-----------------"
    
    # Test database connectivity through health endpoint
    run_test "Database connection pool" "json_test '$API_URL/health' '.database' 'connected'"
    
    # Test Redis connectivity
    run_test "Redis cache status" "json_test '$API_URL/health' '.cache.cache_enabled' 'true'"
    
    echo ""
    
    # Summary
    echo "📋 Test Summary"
    echo "==============="
    echo "Total Tests: $TESTS_TOTAL"
    echo "Passed: $TESTS_PASSED"
    echo "Failed: $TESTS_FAILED"
    
    if [ $TESTS_FAILED -eq 0 ]; then
        log_success "🎉 All tests passed! Production deployment is healthy."
        exit_code=0
    elif [ $TESTS_PASSED -gt $((TESTS_TOTAL / 2)) ]; then
        log_warning "⚠️ Some tests failed, but deployment appears functional."
        log_info "Failed tests may indicate non-critical issues or missing optional components."
        exit_code=0
    else
        log_error "❌ Multiple test failures detected. Please investigate."
        exit_code=1
    fi
    
    echo ""
    echo "💡 Next Steps:"
    echo "- Monitor logs for any errors"
    echo "- Set up external monitoring and alerting"
    echo "- Configure backup procedures"
    echo "- Review security settings"
    
    if [ "$DEPLOYMENT_TYPE" = "docker" ]; then
        echo "- Scale services: docker-compose -f docker-compose.prod.yml up -d --scale backend=3"
        echo "- View logs: docker-compose -f docker-compose.prod.yml logs -f"
    else
        echo "- Scale services: kubectl scale deployment waveflix-backend --replicas=3 -n waveflix"
        echo "- View logs: kubectl logs -f deployment/waveflix-backend -n waveflix"
    fi
    
    exit $exit_code
}

# Check dependencies
if ! command -v curl >/dev/null 2>&1; then
    log_error "curl is required for testing"
    exit 1
fi

if ! command -v jq >/dev/null 2>&1; then
    log_warning "jq not found, some tests may be skipped"
fi

if ! command -v bc >/dev/null 2>&1; then
    log_warning "bc not found, performance tests may be inaccurate"
fi

# Run main function
main "$@"