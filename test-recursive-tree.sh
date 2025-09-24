#!/bin/bash

# Test script for recursive tree structure building
echo "🌳 Testing Recursive Tree Structure Building"
echo "============================================="

# Check if backend is running
echo "📡 Checking if backend is running..."
if ! curl -s http://localhost:8080/api/health > /dev/null; then
    echo "❌ Backend is not running. Please start the backend first:"
    echo "   cd backend && go run main.go"
    exit 1
fi

echo "✅ Backend is running"

# Test namespaces endpoint
echo ""
echo "📦 Getting available namespaces..."
NAMESPACES=$(curl -s http://localhost:8080/api/namespaces)
FIRST_NAMESPACE=$(echo $NAMESPACES | jq -r '.[0]' 2>/dev/null)

if [ "$FIRST_NAMESPACE" = "null" ] || [ -z "$FIRST_NAMESPACE" ]; then
    echo "⚠️  No namespaces found. Using default namespace for testing."
    FIRST_NAMESPACE="default"
fi

echo "Using namespace: $FIRST_NAMESPACE"

# Test deployments (common root resource)
echo ""
echo "🚀 Testing with Deployments as root nodes..."
DEPLOYMENTS=$(curl -s "http://localhost:8080/api/resources/deployments?namespace=$FIRST_NAMESPACE")
FIRST_DEPLOYMENT=$(echo $DEPLOYMENTS | jq -r '.[0].name' 2>/dev/null)

if [ "$FIRST_DEPLOYMENT" != "null" ] && [ -n "$FIRST_DEPLOYMENT" ]; then
    echo "Found deployment: $FIRST_DEPLOYMENT"
    echo ""
    echo "🌳 Building recursive tree with $FIRST_DEPLOYMENT as root:"
    echo "========================================================="
    curl -s "http://localhost:8080/api/resources/deployment/$FIRST_DEPLOYMENT/tree?namespace=$FIRST_NAMESPACE" | jq '.' 2>/dev/null || echo "Error building tree"
else
    echo "No deployments found. Creating a test deployment..."
    echo ""
    echo "📝 To test recursive tree building, create a deployment:"
    echo "kubectl create deployment nginx-test --image=nginx --namespace=$FIRST_NAMESPACE"
    echo "kubectl expose deployment nginx-test --port=80 --namespace=$FIRST_NAMESPACE"
    echo ""
    echo "This will create a hierarchy: Deployment -> ReplicaSet -> Pod(s) + Service"
fi

# Test KubeBlocks resources if available
echo ""
echo "🎯 Testing KubeBlocks resources..."
CLUSTERS=$(curl -s "http://localhost:8080/api/resources/clusters?namespace=$FIRST_NAMESPACE")
FIRST_CLUSTER=$(echo $CLUSTERS | jq -r '.[0].name' 2>/dev/null)

if [ "$FIRST_CLUSTER" != "null" ] && [ -n "$FIRST_CLUSTER" ]; then
    echo "Found KubeBlocks cluster: $FIRST_CLUSTER"
    echo ""
    echo "🌳 Building recursive tree with KubeBlocks cluster $FIRST_CLUSTER as root:"
    echo "========================================================================"
    curl -s "http://localhost:8080/api/resources/cluster/$FIRST_CLUSTER/tree?namespace=$FIRST_NAMESPACE" | jq '.' 2>/dev/null || echo "Error building cluster tree"
else
    echo "No KubeBlocks clusters found. If you have KubeBlocks installed, create a cluster to test recursive tree building."
fi

echo ""
echo "✨ Recursive tree test completed!"
echo ""
echo "📖 What to expect in recursive tree building:"
echo "1. 🏷️  All resources are found using the root resource's name as label selector"
echo "2. 🔄 Each found child resource is recursively searched for its own children"
echo "3. 🌳 This creates a complete multi-level tree structure"
echo "4. 📊 The tree shows the full hierarchy: Root -> Children -> Grandchildren -> ..."
echo ""
echo "🎯 Example hierarchy for a Deployment:"
echo "Deployment (root)"
echo "├── ReplicaSet (child)"
echo "│   └── Pod (grandchild)"
echo "├── Service (child)"
echo "└── ConfigMap (child)"
