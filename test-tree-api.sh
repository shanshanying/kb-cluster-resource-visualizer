#!/bin/bash

# Test script for tree structure API
# This script tests the tree structure API endpoint

echo "🚀 Testing K8s Resource Visualizer Tree API"
echo "=============================================="

# Check if backend is running
echo "📡 Checking if backend is running..."
if ! curl -s http://localhost:8080/api/health > /dev/null; then
    echo "❌ Backend is not running. Please start the backend first:"
    echo "   cd backend && go run main.go"
    exit 1
fi

echo "✅ Backend is running"

# Test health endpoint
echo ""
echo "🏥 Testing health endpoint..."
curl -s http://localhost:8080/api/health | jq '.'

# Test namespaces endpoint
echo ""
echo "📦 Testing namespaces endpoint..."
NAMESPACES=$(curl -s http://localhost:8080/api/namespaces)
echo "Available namespaces: $NAMESPACES"

# Extract first namespace for testing (if any)
FIRST_NAMESPACE=$(echo $NAMESPACES | jq -r '.[0]' 2>/dev/null)

if [ "$FIRST_NAMESPACE" = "null" ] || [ -z "$FIRST_NAMESPACE" ]; then
    echo "⚠️  No namespaces found. Using default namespace for testing."
    FIRST_NAMESPACE="default"
fi

echo "Using namespace: $FIRST_NAMESPACE"

# Test getting pods in the namespace
echo ""
echo "🐳 Testing pods endpoint..."
PODS_RESPONSE=$(curl -s "http://localhost:8080/api/resources/pods?namespace=$FIRST_NAMESPACE")
echo "Pods response: $PODS_RESPONSE" | jq '.' 2>/dev/null || echo "$PODS_RESPONSE"

# Extract first pod for tree testing (if any)
FIRST_POD=$(echo $PODS_RESPONSE | jq -r '.[0].name' 2>/dev/null)

if [ "$FIRST_POD" = "null" ] || [ -z "$FIRST_POD" ]; then
    echo "⚠️  No pods found in namespace $FIRST_NAMESPACE for tree testing."
    echo ""
    echo "📋 To test the tree API, you need some Kubernetes resources."
    echo "   Try creating a deployment:"
    echo "   kubectl create deployment test-app --image=nginx --namespace=$FIRST_NAMESPACE"
    echo ""
    echo "🌳 Testing tree API with a hypothetical deployment as root node..."
    curl -s "http://localhost:8080/api/resources/deployment/test-app/tree?namespace=$FIRST_NAMESPACE" | jq '.' 2>/dev/null || echo "No deployment found"

    echo ""
    echo "🎯 Testing KubeBlocks resources..."
    echo "Testing clusters endpoint:"
    curl -s "http://localhost:8080/api/resources/clusters?namespace=$FIRST_NAMESPACE" | jq '.' 2>/dev/null || echo "No clusters found or KubeBlocks not installed"

    echo "Testing components endpoint:"
    curl -s "http://localhost:8080/api/resources/components?namespace=$FIRST_NAMESPACE" | jq '.' 2>/dev/null || echo "No components found or KubeBlocks not installed"
else
    echo ""
    echo "🌳 Testing tree API with pod as root node: $FIRST_POD"
    TREE_RESPONSE=$(curl -s "http://localhost:8080/api/resources/pod/$FIRST_POD/tree?namespace=$FIRST_NAMESPACE")
    echo "Tree structure response (with $FIRST_POD as root):"
    echo "$TREE_RESPONSE" | jq '.' 2>/dev/null || echo "$TREE_RESPONSE"

    echo ""
    echo "🎯 Testing KubeBlocks resources..."
    echo "Testing clusters endpoint:"
    curl -s "http://localhost:8080/api/resources/clusters?namespace=$FIRST_NAMESPACE" | jq '.' 2>/dev/null || echo "No clusters found or KubeBlocks not installed"

    echo "Testing components endpoint:"
    curl -s "http://localhost:8080/api/resources/components?namespace=$FIRST_NAMESPACE" | jq '.' 2>/dev/null || echo "No components found or KubeBlocks not installed"
fi

echo ""
echo "✨ Test completed!"
echo ""
echo "📖 How to use:"
echo "1. Start the backend: cd backend && go run main.go"
echo "2. Start the frontend: cd frontend && npm run dev"
echo "3. Open http://localhost:5173 in your browser"
echo "4. Toggle 'Tree Layout' to ON in the header"
echo "5. Select a resource to use as the root node of the tree"
echo "6. The system will build a tree structure with your selected resource as the root"
echo ""
echo "🎯 KubeBlocks Support:"
echo "- The tree now includes KubeBlocks custom resources (clusters, components, etc.)"
echo "- If you have KubeBlocks installed, you can visualize complete KubeBlocks resource hierarchies"
echo "- Standard Kubernetes resources and KubeBlocks resources can be mixed in the same tree"
