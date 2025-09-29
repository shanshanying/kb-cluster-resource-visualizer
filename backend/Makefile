# K8s Resource Visualizer Backend Makefile

.PHONY: fmt lint test build run deps clean dev test-tree help

# Default target
all: deps fmt lint build

# Format code
fmt:
	@echo "🎨 Formatting Go code..."
	go fmt ./...

# Run linter
lint:
	@echo "🔍 Running linter..."
	golangci-lint run || echo "⚠️  Install golangci-lint for better linting"

# Run tests
test:
	@echo "🧪 Running tests..."
	go test -v ./...

# Run tests with coverage
test-coverage:
	@echo "📊 Running tests with coverage..."
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

# Build the application with new resource tree support
build:
	@echo "🔨 Building application..."
	go build -o bin/k8s-resource-visualizer main.go resource_tree.go

# Run the application in development mode
run:
	@echo "🚀 Starting development server..."
	go run main.go resource_tree.go

# Development mode with auto-reload (requires air)
dev:
	@echo "🔄 Starting development mode with auto-reload..."
	@if command -v air >/dev/null 2>&1; then \
		air; \
	else \
		echo "📦 Installing air for auto-reload..."; \
		go install github.com/cosmtrek/air@latest; \
		air; \
	fi

# Install dependencies
deps:
	@echo "📦 Installing dependencies..."
	go mod tidy
	go mod download

# Clean build artifacts
clean:
	@echo "🧹 Cleaning build artifacts..."
	rm -rf bin/
	rm -f coverage.out coverage.html
	go clean

# Test the new tree API functionality
test-tree:
	@echo "🌳 Testing tree API functionality..."
	@if [ -f "../test-complete-tree-structure.sh" ]; then \
		cd .. && ./test-complete-tree-structure.sh; \
	else \
		echo "❌ Tree test script not found. Please run from project root."; \
	fi

# Docker build
docker-build:
	@echo "🐳 Building Docker image..."
	docker build -t k8s-resource-visualizer:latest .

# Docker run
docker-run:
	@echo "🐳 Running Docker container..."
	docker run -p 8080:8080 k8s-resource-visualizer:latest

# Performance benchmark
benchmark:
	@echo "⚡ Running performance benchmarks..."
	go test -bench=. -benchmem ./...

# Check Go version and dependencies
check-env:
	@echo "🔍 Checking environment..."
	@echo "Go version: $$(go version)"
	@echo "GOPATH: $$(go env GOPATH)"
	@echo "GOROOT: $$(go env GOROOT)"
	@kubectl version --client --short 2>/dev/null || echo "⚠️  kubectl not found"

# Help
help:
	@echo "🛠️  K8s Resource Visualizer Backend Makefile"
	@echo ""
	@echo "Available targets:"
	@echo "  all          - Run deps, fmt, lint, and build"
	@echo "  fmt          - Format Go code"
	@echo "  lint         - Run linter"
	@echo "  test         - Run tests"
	@echo "  test-coverage - Run tests with coverage report"
	@echo "  build        - Build the application"
	@echo "  run          - Run the application"
	@echo "  dev          - Run in development mode with auto-reload"
	@echo "  deps         - Install dependencies"
	@echo "  clean        - Clean build artifacts"
	@echo "  test-tree    - Test the new tree API functionality"
	@echo "  docker-build - Build Docker image"
	@echo "  docker-run   - Run Docker container"
	@echo "  benchmark    - Run performance benchmarks"
	@echo "  check-env    - Check environment setup"
	@echo "  help         - Show this help message"