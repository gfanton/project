.PHONY: build build-tmux install test test-coverage test-shell test-integration test-tmux test-nix lint clean tidy dev update-vendor-hash release

# Variables
APP_NAME := proj
TMUX_APP_NAME := proj-tmux
BUILD_DIR := ./build
CMD_DIR := ./cmd/proj
TMUX_CMD_DIR := ./plugins/proj-tmux

# Default target
all: build

# Build the main application
build:
	@mkdir -p $(BUILD_DIR)
	go build -o $(BUILD_DIR)/$(APP_NAME) $(CMD_DIR)

# Build the tmux integration binary
build-tmux:
	@mkdir -p $(BUILD_DIR)
	go build -o $(BUILD_DIR)/$(TMUX_APP_NAME) $(TMUX_CMD_DIR)

# Build both binaries
build-all: build build-tmux

# Install the main application to GOBIN
install:
	go install $(CMD_DIR)

# Install the tmux integration binary to GOBIN
install-tmux:
	go install $(TMUX_CMD_DIR)

# Install both binaries
install-all: install install-tmux

# Run tests
test:
	go test -v ./...

# Run tests with coverage
test-coverage:
	go test -cover ./...

# Run Go-based shell tests (similar to zoxide)
test-shell:
	go test -v ./internal/shell/

# Run integration tests (BATS + Expect)
test-integration:
	@echo "Running integration tests..."
	./tests/run_tests.sh

# Run tmux unit tests with BATS
test-tmux: build build-tmux
	@echo "Running tmux unit tests..."
	@if command -v bats >/dev/null 2>&1; then \
		if [[ -d tests/unit && $$(find tests/unit -name "*.bats" | wc -l) -gt 0 ]]; then \
			bats tests/unit/; \
		else \
			echo "No BATS unit tests found in tests/unit/"; \
		fi \
	else \
		echo "BATS not found. Install with: nix develop .#testing"; \
		exit 1; \
	fi

# Run all tests in Nix environment
test-nix:
	nix develop .#testing --command bash -c "make test && make test-tmux"

# Run linting
lint:
	go vet ./...
	go fmt ./...

# Clean build artifacts
clean:
	rm -rf $(BUILD_DIR) coverage.out coverage.html

# Tidy dependencies
tidy:
	go mod tidy

# Development target - build and run
dev: build
	$(BUILD_DIR)/$(APP_NAME)

# Update vendorHash in flake.nix with the correct hash
update-vendor-hash:
	@./scripts/update-vendor-hash.sh

# Release target - calls release.sh
release:
	@echo "Use: ./scripts/release.sh <version>"
	@echo "Example: ./scripts/release.sh v1.2.3"

# Tmux integration testing targets  
test-tmux-unit: build build-tmux
	@echo "Running tmux unit tests..."
	@if command -v bats >/dev/null 2>&1; then \
		bats tests/unit/; \
	else \
		echo "BATS not found. Use: nix develop .#testing"; \
	fi

# Nix-based testing targets
test-nix-tmux:
	@echo "Running tmux tests in Nix environment..."
	nix build .#checks.$$(nix eval --impure --expr builtins.currentSystem).tmux-unit-tests -L

# Help target for tmux testing
help-tmux:
	@echo "Tmux Integration Testing Targets:"
	@echo "  build-tmux             Build proj-tmux binary"
	@echo "  build-all              Build both proj and proj-tmux binaries"
	@echo "  install-tmux           Install proj-tmux binary to GOBIN"
	@echo "  install-all            Install both binaries to GOBIN"
	@echo "  test-tmux-unit         Run BATS unit tests for tmux integration"
	@echo "  test-nix-tmux          Run tmux tests in Nix environment"
	@echo ""
	@echo "Development Environment:"
	@echo "  nix develop .#testing  Enter testing environment with all tools"
	@echo ""
	@echo "Manual Testing:"
	@echo "  bats tests/unit/       Run BATS tests manually"

.PHONY: build-tmux build-all install-tmux install-all test-tmux-unit test-nix-tmux help-tmux