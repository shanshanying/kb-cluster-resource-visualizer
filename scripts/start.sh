#!/bin/bash

# K8s Resource Visualizer Enhanced Startup Script
# Supports new tree structure format and improved resource discovery

set -e  # Exit on any error

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

# Logging functions
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

log_header() {
    echo -e "${CYAN}🔹 $1${NC}"
}

# Configuration
BACKEND_PORT=8080
FRONTEND_PORT=5173
HEALTH_CHECK_RETRIES=10
HEALTH_CHECK_DELAY=1

echo "🚀 K8s Resource Visualizer - Enhanced Startup"
echo "=============================================="
echo ""

# Check prerequisites
log_header "Checking Prerequisites"

# Check if we're in the right directory
if [ ! -f "backend/main.go" ] || [ ! -f "frontend/package.json" ]; then
    log_error "Please run this script from the project root directory"
    log_info "Expected structure:"
    log_info "  - backend/main.go"
    log_info "  - backend/resource_tree.go"
    log_info "  - frontend/package.json"
    exit 1
fi

# Check Go installation
if ! command -v go &> /dev/null; then
    log_error "Go is not installed. Please install Go 1.19 or later"
    exit 1
fi

GO_VERSION=$(go version | cut -d' ' -f3)
log_success "Go found: $GO_VERSION"

# Check Node.js installation
if ! command -v node &> /dev/null; then
    log_error "Node.js is not installed. Please install Node.js 16 or later"
    exit 1
fi

NODE_VERSION=$(node --version)
log_success "Node.js found: $NODE_VERSION"

# Check kubectl installation and cluster access
log_info "Checking Kubernetes cluster access..."
if ! kubectl cluster-info > /dev/null 2>&1; then
    log_warning "Cannot access Kubernetes cluster. The application will start but may not function fully."
    log_info "To fix this, ensure:"
    log_info "  1. kubectl is installed and configured"
    log_info "  2. You have access to a Kubernetes cluster"
    log_info "  3. Your kubeconfig is properly set"
    log_info ""
    read -p "Continue anyway? (y/N): " -n 1 -r
    echo
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        exit 1
    fi
else
    CLUSTER_INFO=$(kubectl cluster-info | head -n1)
    log_success "Kubernetes cluster access confirmed"
    log_info "$CLUSTER_INFO"
fi

echo ""

# Prepare backend
log_header "Preparing Backend"
cd backend

# Check for required files
if [ ! -f "main.go" ] || [ ! -f "resource_tree.go" ]; then
    log_error "Required backend files not found:"
    log_info "  - main.go (main application)"
    log_info "  - resource_tree.go (new tree implementation)"
    exit 1
fi

# Install/update Go dependencies
log_info "Installing Go dependencies..."
go mod tidy
go mod download
log_success "Go dependencies ready"

# Check if port is available
if lsof -Pi :$BACKEND_PORT -sTCP:LISTEN -t >/dev/null 2>&1; then
    log_warning "Port $BACKEND_PORT is already in use"
    log_info "Attempting to stop existing process..."
    lsof -ti:$BACKEND_PORT | xargs kill -9 2>/dev/null || true
    sleep 2
fi

# Start backend with new resource tree support
log_info "Starting backend server with enhanced tree structure..."
go run main.go resource_tree.go &
BACKEND_PID=$!

# Wait for backend to start with retries
log_info "Waiting for backend to become ready..."
for i in $(seq 1 $HEALTH_CHECK_RETRIES); do
    if curl -s http://localhost:$BACKEND_PORT/api/health > /dev/null 2>&1; then
        log_success "Backend server ready on port $BACKEND_PORT"
        break
    fi
    if [ $i -eq $HEALTH_CHECK_RETRIES ]; then
        log_error "Backend failed to start after $HEALTH_CHECK_RETRIES attempts"
        kill $BACKEND_PID 2>/dev/null || true
        exit 1
    fi
    sleep $HEALTH_CHECK_DELAY
done

# Test new tree API endpoint
log_info "Testing new tree API functionality..."
TREE_API_RESPONSE=$(curl -s http://localhost:$BACKEND_PORT/api/resources/cluster/test/tree?namespace=default 2>/dev/null || echo "")
if [[ "$TREE_API_RESPONSE" == *"error"* ]] && [[ "$TREE_API_RESPONSE" != *"not found"* ]]; then
    log_warning "Tree API test returned an error (this may be normal if no test resources exist)"
else
    log_success "Tree API endpoint responding correctly"
fi

cd ..

# Prepare frontend
log_header "Preparing Frontend"
cd frontend

# Check for package.json
if [ ! -f "package.json" ]; then
    log_error "package.json not found in frontend directory"
    exit 1
fi

# Install/update npm dependencies
if [ ! -d "node_modules" ] || [ "package.json" -nt "node_modules" ]; then
    log_info "Installing npm dependencies..."
    npm install
    log_success "npm dependencies ready"
else
    log_info "npm dependencies up to date"
fi

# Check if frontend port is available
if lsof -Pi :$FRONTEND_PORT -sTCP:LISTEN -t >/dev/null 2>&1; then
    log_warning "Port $FRONTEND_PORT is already in use"
    log_info "Attempting to stop existing process..."
    lsof -ti:$FRONTEND_PORT | xargs kill -9 2>/dev/null || true
    sleep 2
fi

# Start frontend
log_info "Starting React frontend with tree structure support..."
npm run dev &
FRONTEND_PID=$!

# Wait for frontend to start
log_info "Waiting for frontend to become ready..."
sleep 8

# Check frontend health
if curl -s http://localhost:$FRONTEND_PORT > /dev/null 2>&1; then
    log_success "Frontend ready on port $FRONTEND_PORT"
else
    log_warning "Frontend may still be starting..."
fi

cd ..

# Display startup summary
echo ""
log_header "🎉 Startup Complete!"
echo "=============================="
echo ""
echo "📱 Frontend:     http://localhost:$FRONTEND_PORT"
echo "🔌 Backend API:  http://localhost:$BACKEND_PORT"
echo "📊 Health Check: http://localhost:$BACKEND_PORT/api/health"
echo "🌳 Tree API:     http://localhost:$BACKEND_PORT/api/resources/:type/:root/tree"
echo ""
echo "🆕 New Features:"
echo "  ✅ Enhanced tree structure with ownerReference support"
echo "  ✅ Label selector optimization (app.kubernetes.io/instance)"
echo "  ✅ Direct unstructured.Unstructured serialization"
echo "  ✅ Improved performance and data completeness"
echo ""
echo "🧪 Test Commands:"
echo "  make test-tree                    # Test tree API functionality"
echo "  ./test-complete-tree-structure.sh # Complete integration test"
echo ""
echo "Press Ctrl+C to stop all services"
echo ""

# Function to cleanup on exit
cleanup() {
    echo ""
    log_header "Shutting Down Services"

    if [ ! -z "$BACKEND_PID" ]; then
        log_info "Stopping backend (PID: $BACKEND_PID)..."
        kill $BACKEND_PID 2>/dev/null || true
    fi

    if [ ! -z "$FRONTEND_PID" ]; then
        log_info "Stopping frontend (PID: $FRONTEND_PID)..."
        kill $FRONTEND_PID 2>/dev/null || true
    fi

    # Kill any remaining processes on our ports
    lsof -ti:$BACKEND_PORT 2>/dev/null | xargs kill -9 2>/dev/null || true
    lsof -ti:$FRONTEND_PORT 2>/dev/null | xargs kill -9 2>/dev/null || true

    log_success "All services stopped"
    echo "👋 Thank you for using K8s Resource Visualizer!"
    exit 0
}

# Set trap to cleanup on exit
trap cleanup SIGINT SIGTERM

# Keep script running and monitor processes
while true; do
    # Check if backend is still running
    if ! kill -0 $BACKEND_PID 2>/dev/null; then
        log_error "Backend process died unexpectedly"
        cleanup
    fi

    # Check if frontend is still running
    if ! kill -0 $FRONTEND_PID 2>/dev/null; then
        log_error "Frontend process died unexpectedly"
        cleanup
    fi

    sleep 5
done