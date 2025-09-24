#!/bin/bash

# K8s Resource Visualizer Startup Script

echo "🚀 Starting K8s Resource Visualizer..."

# Check if kubectl is available and cluster is accessible
echo "📋 Checking Kubernetes cluster access..."
if ! kubectl cluster-info > /dev/null 2>&1; then
    echo "❌ Cannot access Kubernetes cluster. Please ensure:"
    echo "   1. kubectl is installed and configured"
    echo "   2. You have access to a Kubernetes cluster"
    echo "   3. Your kubeconfig is properly set"
    exit 1
fi

echo "✅ Kubernetes cluster access confirmed"

# Start backend
echo "🔧 Starting backend server..."
cd backend
if [ ! -f "go.mod" ]; then
    echo "❌ go.mod not found. Please run from the project root directory."
    exit 1
fi

# Install dependencies if needed
if [ ! -d "vendor" ]; then
    echo "📦 Installing Go dependencies..."
    go mod tidy
fi

# Start backend in background
echo "🌟 Starting Go backend on port 8080..."
go run main.go &
BACKEND_PID=$!

# Wait a moment for backend to start
sleep 3

# Check if backend is running
if ! curl -s http://localhost:8080/api/health > /dev/null; then
    echo "❌ Backend failed to start"
    kill $BACKEND_PID 2>/dev/null
    exit 1
fi

echo "✅ Backend started successfully"

# Start frontend
echo "🎨 Starting frontend..."
cd ../frontend

# Install dependencies if needed
if [ ! -d "node_modules" ]; then
    echo "📦 Installing npm dependencies..."
    npm install
fi

echo "🌟 Starting React frontend on port 5173..."
npm run dev &
FRONTEND_PID=$!

# Wait for frontend to start
sleep 5

echo ""
echo "🎉 K8s Resource Visualizer is running!"
echo "📱 Frontend: http://localhost:5173"
echo "🔌 Backend API: http://localhost:8080"
echo ""
echo "Press Ctrl+C to stop all services"

# Function to cleanup on exit
cleanup() {
    echo ""
    echo "🛑 Stopping services..."
    kill $BACKEND_PID 2>/dev/null
    kill $FRONTEND_PID 2>/dev/null
    echo "👋 Goodbye!"
    exit 0
}

# Set trap to cleanup on exit
trap cleanup SIGINT SIGTERM

# Wait for processes
wait
