.PHONY: test test-verbose test-coverage test-race test-integration bench clean

# Run all tests
test:
	go test -v ./...

# Run tests with verbose output
test-verbose:
	go test -v -race ./...

# Run tests with coverage
test-coverage:
	go test -v -race -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

# Run tests with race detection
test-race:
	go test -v -race ./...

# Run only integration tests
test-integration:
	go test -v -run Integration ./...

# Run only unit tests (exclude integration tests)
test-unit:
	go test -v -run "^((?!Integration).)*$$" ./...

# Run benchmarks
bench:
	go test -bench=. -benchmem ./...

# Run performance tests
perf:
	go test -v -run "Performance" ./...

# Run benchmarks with CPU profiling
bench-profile:
	go test -bench=. -benchmem -cpuprofile=cpu.prof ./...
	@echo "CPU profile saved to cpu.prof"

# Run specific test
test-specific:
	go test -v -run $(TEST) ./...

# Clean test artifacts
clean:
	rm -f coverage.out coverage.html

# Format code
fmt:
	go fmt ./...

# Run linter
lint:
	golangci-lint run

# Run tests and generate coverage report
coverage: test-coverage
	@echo "Coverage report generated: coverage.html"

# Build the application
build:
	go build -o bin/k8s-resource-visualizer .

# Run the application
run:
	go run .

# Install dependencies
deps:
	go mod download
	go mod tidy

# Update dependencies
update-deps:
	go get -u ./...
	go mod tidy

help:
	@echo "Available targets:"
	@echo "  test            - Run all tests"
	@echo "  test-verbose    - Run tests with verbose output and race detection"
	@echo "  test-coverage   - Run tests with coverage report"
	@echo "  test-race       - Run tests with race detection"
	@echo "  test-integration- Run only integration tests"
	@echo "  test-unit       - Run only unit tests"
	@echo "  bench           - Run benchmarks"
	@echo "  perf            - Run performance tests"
	@echo "  bench-profile   - Run benchmarks with CPU profiling"
	@echo "  test-specific   - Run specific test (use TEST=TestName)"
	@echo "  clean           - Clean test artifacts"
	@echo "  fmt             - Format code"
	@echo "  lint            - Run linter"
	@echo "  coverage        - Generate coverage report"
	@echo "  build           - Build the application"
	@echo "  run             - Run the application"
	@echo "  deps            - Install dependencies"
	@echo "  update-deps     - Update dependencies"
	@echo "  help            - Show this help message"
