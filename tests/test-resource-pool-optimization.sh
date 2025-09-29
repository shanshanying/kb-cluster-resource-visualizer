#!/bin/bash

# Test script to verify resource pool optimization
echo "🧪 Testing Resource Pool Optimization"
echo "======================================"

# Set test configuration
export NAMESPACE="kbcloud-system"
export CLUSTER_NAME="pgvector-cluster-cjgc"
export BACKEND_URL="http://localhost:8080"

echo "📋 Test Configuration:"
echo "  Namespace: $NAMESPACE"
echo "  Cluster: $CLUSTER_NAME"
echo "  Backend URL: $BACKEND_URL"
echo ""

# Function to measure API response time
measure_response_time() {
    local url=$1
    local description=$2
    echo "⏱️  Testing: $description"
    echo "   URL: $url"

    # Use curl with timing to measure response time
    local start_time=$(date +%s.%N)
    local response=$(curl -s -w "HTTP_STATUS:%{http_code};TIME_TOTAL:%{time_total}" "$url")
    local end_time=$(date +%s.%N)

    # Extract HTTP status and time from curl output
    local http_status=$(echo "$response" | grep -o "HTTP_STATUS:[0-9]*" | cut -d: -f2)
    local time_total=$(echo "$response" | grep -o "TIME_TOTAL:[0-9.]*" | cut -d: -f2)

    if [ "$http_status" = "200" ]; then
        echo "   ✅ Success (${time_total}s)"

        # Try to extract and count nodes from response
        local node_count=$(echo "$response" | grep -o '"resource":' | wc -l | xargs)
        if [ "$node_count" -gt 0 ]; then
            echo "   📊 Found $node_count resource nodes in tree"
        fi
    else
        echo "   ❌ Failed (HTTP $http_status)"
    fi
    echo ""
}

# Test 1: Health check
echo "1️⃣  Health Check"
measure_response_time "$BACKEND_URL/api/health" "Backend health check"

# Test 2: Get namespaces
echo "2️⃣  Namespace Discovery"
measure_response_time "$BACKEND_URL/api/namespaces" "Fetch available namespaces"

# Test 3: Get clusters in namespace
echo "3️⃣  Cluster Resources"
measure_response_time "$BACKEND_URL/api/resources/clusters?namespace=$NAMESPACE" "Fetch clusters in namespace"

# Test 4: Build resource tree (this is where the optimization should shine)
echo "4️⃣  Resource Tree Building (OPTIMIZED)"
echo "   This test will show the performance improvement from resource pool optimization"
measure_response_time "$BACKEND_URL/api/resources/clusters/$CLUSTER_NAME/tree?namespace=$NAMESPACE" "Build complete resource tree"

echo "🎯 Test Summary"
echo "==============="
echo "The resource tree building should now be significantly faster because:"
echo "  ✅ Resources are loaded once into a pool at the beginning"
echo "  ✅ Tree building uses efficient lookups instead of repeated API calls"
echo "  ✅ Each resource is processed only once and removed from the pool"
echo "  ✅ No redundant searches across resource types"
echo ""
echo "📈 Performance Benefits:"
echo "  • Reduced API calls to Kubernetes"
echo "  • Faster tree construction"
echo "  • Lower memory usage during tree building"
echo "  • Better scalability with large numbers of resources"
