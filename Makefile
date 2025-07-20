.PHONY: build install test lint clean tidy test-coverage test-coverage-html bench test-integration test-nix test-shell

# Variables
APP_NAME := proj
BUILD_DIR := ./build
CMD_DIR := ./cmd/proj
GO_FILES := $(shell find . -type f -name '*.go')
TEMPLATE_FILES := $(shell find pkg/template -type f -name '*.init' 2>/dev/null || true)
COVERAGE_OUT := coverage.out
COVERAGE_HTML := coverage.html

# Default target
all: build

# Build the application
build: $(BUILD_DIR)/$(APP_NAME)

$(BUILD_DIR)/$(APP_NAME): $(GO_FILES) $(TEMPLATE_FILES) go.mod go.sum
	@mkdir -p $(BUILD_DIR)
	go build -o $(BUILD_DIR)/$(APP_NAME) $(CMD_DIR)

# Install the application to GOBIN
install:
	go install $(CMD_DIR)

# Run tests
test:
	go test -v ./...

# Run shell integration tests (Go-based)
test-shell:
	go test -v ./internal/shell/ -run TestShellIntegration

# Run tests with coverage
test-coverage:
	go test -v -coverprofile=$(COVERAGE_OUT) ./...
	go tool cover -func=$(COVERAGE_OUT)

# Generate HTML coverage report
test-coverage-html: test-coverage
	go tool cover -html=$(COVERAGE_OUT) -o $(COVERAGE_HTML)
	@echo "Coverage report generated: $(COVERAGE_HTML)"

# Run benchmarks
bench:
	go test -bench=. -benchmem ./...

# Run linting
lint:
	go vet ./...
	go fmt ./...

# Clean build artifacts
clean:
	rm -rf $(BUILD_DIR) $(COVERAGE_OUT) $(COVERAGE_HTML)

# Tidy dependencies
tidy:
	go mod tidy

# Development target - build and run
dev: build
	$(BUILD_DIR)/$(APP_NAME)

# Run integration tests (requires bats and expect)
test-integration: build
	@echo "Running integration tests..."
	@./tests/run_tests.sh

# Run all tests in Nix environment
test-nix:
	@if command -v nix-shell >/dev/null 2>&1; then \
		echo "Running tests in Nix environment..."; \
		nix-shell --run "make test-integration"; \
	else \
		echo "Error: Nix is not installed. Please install Nix first."; \
		echo "Visit: https://nixos.org/download.html"; \
		exit 1; \
	fi

# Enter Nix shell for testing
shell-nix:
	@if command -v nix-shell >/dev/null 2>&1; then \
		nix-shell; \
	else \
		echo "Error: Nix is not installed. Please install Nix first."; \
		echo "Visit: https://nixos.org/download.html"; \
		exit 1; \
	fi
