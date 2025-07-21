.PHONY: build install test lint clean tidy dev

# Variables
APP_NAME := proj
BUILD_DIR := ./build
CMD_DIR := ./cmd/proj

# Default target
all: build

# Build the application
build:
	@mkdir -p $(BUILD_DIR)
	go build -o $(BUILD_DIR)/$(APP_NAME) $(CMD_DIR)

# Install the application to GOBIN
install:
	go install $(CMD_DIR)

# Run tests
test:
	go test -v ./...

# Run tests with coverage
test-coverage:
	go test -cover ./...

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