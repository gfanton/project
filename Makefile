# ---- Defensive Preamble
.DELETE_ON_ERROR:
MAKEFLAGS += --warn-undefined-variables
SHELL := /bin/sh
.SUFFIXES:
.DEFAULT_GOAL := all

# ---- Phony Targets
.PHONY: all build build-tmux build-all install install-tmux install-all \
	test test-coverage test-shell test-integration test-tmux test-nix test-plugin \
	lint clean tidy dev dev-tmux update-vendor-hash release test-nix-tmux help \
	test-completion tmux-sandbox

# ---- Variables
APP_NAME := proj
TMUX_APP_NAME := proj-tmux
BUILD_DIR := ./build
CMD_DIR := ./cmd/proj
TMUX_CMD_DIR := ./plugins/proj-tmux

# ---- Build Variables for Version Information
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "none")
DATE ?= $(shell date -u '+%Y-%m-%d %H:%M:%S UTC')
BUILT_BY ?= $(shell whoami)

# Build flags for version injection
BUILD_FLAGS := -ldflags "\
	-X 'main.version=$(VERSION)' \
	-X 'main.commit=$(COMMIT)' \
	-X 'main.date=$(DATE)' \
	-X 'main.builtBy=$(BUILT_BY)'"

# ---- Default Target

all: build build-tmux  ## Build both proj and proj-tmux binaries

# ---- Build Targets

build:  ## Build the main proj application
	@mkdir -p $(BUILD_DIR)
	go build $(BUILD_FLAGS) -o $(BUILD_DIR)/$(APP_NAME) $(CMD_DIR)

build-tmux:  ## Build the tmux integration binary
	@mkdir -p $(BUILD_DIR)
	go build $(BUILD_FLAGS) -o $(BUILD_DIR)/$(TMUX_APP_NAME) $(TMUX_CMD_DIR)

build-all: build build-tmux  ## Build both binaries (alias for all)

# ---- Install Targets

install:  ## Install proj to GOBIN
	go install $(BUILD_FLAGS) $(CMD_DIR)

install-tmux:  ## Install proj-tmux to GOBIN
	go install $(BUILD_FLAGS) $(TMUX_CMD_DIR)

install-all: install install-tmux  ## Install both binaries

# ---- Test Targets

test:  ## Run all Go unit tests
	go test -v ./...

test-coverage:  ## Run tests with coverage report
	go test -cover ./...

test-shell:  ## Run shell integration tests
	go test -v ./internal/shell/

test-integration:  ## Run integration tests (BATS + Expect)
	@echo "Running integration tests..."
	./tests/run_tests.sh

test-tmux: build build-tmux  ## Run tmux unit tests with BATS
	@echo "Running tmux unit tests..."
	@if command -v bats >/dev/null 2>&1; then \
		if [ -d tests/unit ] && [ "$$(find tests/unit -name '*.bats' | wc -l)" -gt 0 ]; then \
			bats tests/unit/; \
		else \
			echo "No BATS unit tests found in tests/unit/"; \
		fi \
	else \
		echo "BATS not found. Install with: nix develop .#testing"; \
		exit 1; \
	fi

test-nix:  ## Run all tests in Nix environment
	nix develop .#testing --command bash -c "make test && make test-tmux"

test-nix-tmux:  ## Run tmux tests via Nix checks
	@echo "Running tmux tests in Nix environment..."
	nix build .#checks.$$(nix eval --impure --expr builtins.currentSystem).tmux-unit-tests -L

test-plugin: build-all  ## Test tmux plugin structure and installation
	@echo "Testing tmux plugin installation..."
	@if ! command -v tmux >/dev/null 2>&1; then \
		echo "Error: tmux not found"; \
		exit 1; \
	fi
	@echo "Plugin structure:"
	@ls -la plugins/proj-tmux/plugin/
	@echo "Plugin scripts:"
	@ls -la plugins/proj-tmux/plugin/scripts/
	@echo "Plugin executable permissions:"
	@ls -la plugins/proj-tmux/plugin/proj-tmux.tmux plugins/proj-tmux/plugin/scripts/*.sh
	@echo ""
	@echo "To test manually:"
	@echo "  1. Add to ~/.tmux.conf: run-shell '$(PWD)/plugins/proj-tmux/plugin/proj-tmux.tmux'"
	@echo "  2. Or use TPM: set -g @plugin '$$(basename $(PWD))'"
	@echo "  3. Reload tmux: tmux source-file ~/.tmux.conf"

# ---- Development Targets

dev-tmux: build-all  ## Build and load tmux plugin into current session
	@if [ -z "$$TMUX" ]; then \
		echo "Error: Not inside a tmux session"; \
		exit 1; \
	fi
	@echo "Loading plugin with binaries from $(BUILD_DIR)..."
	tmux set-environment -g PROJ_BIN "$(abspath $(BUILD_DIR))/proj"
	tmux set-environment -g PROJ_TMUX_BIN "$(abspath $(BUILD_DIR))/proj-tmux"
	tmux run-shell "$(abspath plugins/proj-tmux/plugin/proj-tmux.tmux)"
	@echo "Plugin loaded. Environment variables:"
	@tmux show-environment -g | grep PROJ || true
	@echo ""
	@echo "Test with: Prefix+Ctrl+P (sessions), Prefix+Ctrl+W (windows), Prefix+W (workspace menu)"

lint:  ## Run go vet and go fmt
	go vet ./...
	go fmt ./...

clean:  ## Remove build artifacts
	rm -rf $(BUILD_DIR) coverage.out coverage.html

tidy:  ## Clean up Go dependencies
	go mod tidy

dev: build  ## Build and run proj (use ARGS= to pass arguments)
	$(BUILD_DIR)/$(APP_NAME) $(ARGS)

# ---- Release Targets

update-vendor-hash:  ## Update vendorHash in flake.nix
	@./scripts/update-vendor-hash.sh

release:  ## Show release instructions
	@echo "Use: ./scripts/release/release.sh [version]"
	@echo "Example: ./scripts/release/release.sh v1.2.3"
	@echo "Example: ./scripts/release/release.sh  (interactive)"

# ---- Interactive Testing

test-completion:  ## Start isolated zsh for shell completion testing
	@./scripts/test-completion.sh

tmux-sandbox:  ## Start isolated tmux session for plugin testing
	@./scripts/test-tmux.sh

# ---- Help Target

help:  ## Show this help message
	@echo "Usage: make [target]"
	@echo ""
	@echo "Targets:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "  %-20s %s\n", $$1, $$2}'
