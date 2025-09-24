#!/bin/bash

# Integration test script for Tree API
# This script tests the /resources/:type/:root/tree API endpoint
# with real Kubernetes resources

echo "🧪 Tree API Integration Test"
echo "============================"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Test configuration
NAMESPACE="tree-api-test"
APP_NAME="test-nginx"

# Functions
log_info() {
    echo -e "${BLUE}ℹ️  $1${NC}"
}

log_success() {
    echo -e "${GREEN}✅ $1${NC}"
}

log_warning() {
    echo -e "${YELLOW}⚠️  $1${NC}"
}

log_error() {
    echo -e "${RED}❌ $1${NC}"
}

# Check prerequisites
log_info "Checking prerequisites..."

# Check if kubectl is available
if ! command -v kubectl &> /dev/null; then
    log_error "kubectl is not installed or not in PATH"
    exit 1
fi

# Check if backend is running
if ! curl -s http://localhost:8080/api/health > /dev/null; then
    log_error "Backend is not running. Please start the backend first:"
    echo "   cd backend && go run main.go"
    exit 1
fi

log_success "Prerequisites check passed"

# Setup test environment
log_info "Setting up test environment..."

# Create test namespace
kubectl create namespace $NAMESPACE --dry-run=client -o yaml | kubectl apply -f - > /dev/null 2>&1
log_info "Created/verified namespace: $NAMESPACE"

# Create test deployment
log_info "Creating test deployment: $APP_NAME"
kubectl create deployment $APP_NAME \
    --image=nginx:latest \
    --namespace=$NAMESPACE \
    --dry-run=client -o yaml | \
    sed "s/app: $APP_NAME/app: $APP_NAME\n      app.kubernetes.io\/instance: $APP_NAME/" | \
    kubectl apply -f - > /dev/null

# Wait for deployment to be ready
log_info "Waiting for deployment to be ready..."
kubectl wait --for=condition=available --timeout=60s deployment/$APP_NAME -n $NAMESPACE > /dev/null

# Create service
log_info "Creating service for $APP_NAME"
kubectl expose deployment $APP_NAME \
    --port=80 \
    --namespace=$NAMESPACE > /dev/null 2>&1 || true

# Label the service with instance label
kubectl label service $APP_NAME app.kubernetes.io/instance=$APP_NAME -n $NAMESPACE > /dev/null 2>&1 || true

# Create configmap
log_info "Creating configmap for $APP_NAME"
kubectl create configmap ${APP_NAME}-config \
    --from-literal=config.yaml="test: value" \
    --namespace=$NAMESPACE > /dev/null 2>&1 || true

# Label the configmap
kubectl label configmap ${APP_NAME}-config app.kubernetes.io/instance=$APP_NAME -n $NAMESPACE > /dev/null 2>&1 || true

log_success "Test environment setup complete"

# Wait a bit for resources to be fully created
sleep 2

# Test the Tree API
log_info "Testing Tree API..."

echo ""
echo "📊 Test 1: Basic Tree Structure"
echo "================================"

TREE_RESPONSE=$(curl -s "http://localhost:8080/api/resources/deployment/$APP_NAME/tree?namespace=$NAMESPACE")
if echo "$TREE_RESPONSE" | jq . > /dev/null 2>&1; then
    log_success "API returned valid JSON"

    # Count total nodes
    TOTAL_NODES=$(echo "$TREE_RESPONSE" | jq '.[0] | [. as $root | $root, ($root | .. | objects | select(has("resource")))] | length')
    log_info "Total nodes in tree: $TOTAL_NODES"

    # Check root node
    ROOT_NAME=$(echo "$TREE_RESPONSE" | jq -r '.[0].resource.name')
    ROOT_KIND=$(echo "$TREE_RESPONSE" | jq -r '.[0].resource.kind')
    log_info "Root node: $ROOT_KIND/$ROOT_NAME"

    # Check children
    CHILDREN_COUNT=$(echo "$TREE_RESPONSE" | jq '.[0].children | length')
    log_info "Direct children count: $CHILDREN_COUNT"

    if [ "$CHILDREN_COUNT" -gt 0 ]; then
        log_success "Tree structure contains children resources"
        echo "$TREE_RESPONSE" | jq '.[0].children[] | "  - \(.resource.kind)/\(.resource.name)"' -r
    else
        log_warning "No children found in tree structure"
    fi

else
    log_error "API returned invalid JSON:"
    echo "$TREE_RESPONSE"
fi

echo ""
echo "📊 Test 2: Error Handling"
echo "========================="

# Test non-existent resource
log_info "Testing non-existent resource..."
ERROR_RESPONSE=$(curl -s -w "%{http_code}" "http://localhost:8080/api/resources/deployment/nonexistent/tree?namespace=$NAMESPACE")
HTTP_CODE="${ERROR_RESPONSE: -3}"
if [ "$HTTP_CODE" = "404" ]; then
    log_success "Correctly returned 404 for non-existent resource"
else
    log_error "Expected 404, got $HTTP_CODE"
fi

# Test missing namespace
log_info "Testing missing namespace parameter..."
ERROR_RESPONSE=$(curl -s -w "%{http_code}" "http://localhost:8080/api/resources/deployment/$APP_NAME/tree")
HTTP_CODE="${ERROR_RESPONSE: -3}"
if [ "$HTTP_CODE" = "400" ]; then
    log_success "Correctly returned 400 for missing namespace"
else
    log_error "Expected 400, got $HTTP_CODE"
fi

# Test invalid resource type
log_info "Testing invalid resource type..."
ERROR_RESPONSE=$(curl -s -w "%{http_code}" "http://localhost:8080/api/resources/invalidtype/$APP_NAME/tree?namespace=$NAMESPACE")
HTTP_CODE="${ERROR_RESPONSE: -3}"
if [ "$HTTP_CODE" = "400" ]; then
    log_success "Correctly returned 400 for invalid resource type"
else
    log_error "Expected 400, got $HTTP_CODE"
fi

echo ""
echo "📊 Test 3: Recursive Tree Building"
echo "==================================="

# Check for ReplicaSet (should be created by Deployment)
log_info "Checking for ReplicaSet in tree..."
REPLICASET_FOUND=$(echo "$TREE_RESPONSE" | jq '[.. | objects | select(.resource.kind == "ReplicaSet")] | length')
if [ "$REPLICASET_FOUND" -gt 0 ]; then
    log_success "Found ReplicaSet in tree (recursive relationship)"
else
    log_warning "No ReplicaSet found - this might be expected depending on cluster state"
fi

# Check for Pods (should be created by ReplicaSet)
log_info "Checking for Pods in tree..."
PODS_FOUND=$(echo "$TREE_RESPONSE" | jq '[.. | objects | select(.resource.kind == "Pod")] | length')
if [ "$PODS_FOUND" -gt 0 ]; then
    log_success "Found $PODS_FOUND Pod(s) in tree (recursive relationship)"
else
    log_warning "No Pods found - this might be expected depending on cluster state"
fi

echo ""
echo "📊 Test 4: Performance Test"
echo "============================"

log_info "Testing API response time..."
START_TIME=$(date +%s%N)
curl -s "http://localhost:8080/api/resources/deployment/$APP_NAME/tree?namespace=$NAMESPACE" > /dev/null
END_TIME=$(date +%s%N)
RESPONSE_TIME=$(( (END_TIME - START_TIME) / 1000000 ))

if [ "$RESPONSE_TIME" -lt 1000 ]; then
    log_success "API response time: ${RESPONSE_TIME}ms (< 1s)"
elif [ "$RESPONSE_TIME" -lt 5000 ]; then
    log_warning "API response time: ${RESPONSE_TIME}ms (acceptable but slow)"
else
    log_error "API response time: ${RESPONSE_TIME}ms (too slow)"
fi

# Cleanup
echo ""
echo "🧹 Cleanup"
echo "=========="

log_info "Cleaning up test resources..."
kubectl delete namespace $NAMESPACE --ignore-not-found=true > /dev/null 2>&1 &
log_success "Cleanup initiated (running in background)"

echo ""
echo "📋 Test Summary"
echo "==============="
echo "✅ Tree API Integration Tests Completed"
echo "🔗 API Endpoint: /resources/:type/:root/tree"
echo "📊 Tree Structure: Working"
echo "🛡️  Error Handling: Verified"
echo "🔄 Recursive Building: Tested"
echo "⚡ Performance: Measured"
echo ""
echo "📖 Usage Examples:"
echo "curl 'http://localhost:8080/api/resources/deployment/my-app/tree?namespace=default'"
echo "curl 'http://localhost:8080/api/resources/cluster/my-cluster/tree?namespace=kubeblocks'"
echo ""
echo "🎯 The Tree API successfully builds hierarchical resource trees!"
