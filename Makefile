.PHONY: build run test clean install fmt lint help

BINARY_NAME=codedoc
GO_FILES=$(shell find . -name '*.go' -type f)
MAIN_PATH=cmd/codedoc/main.go
BUILD_DIR=build
PATH_ARG ?= .

help:
	@echo "Available targets:"
	@echo "  build       - Build the codedoc binary"
	@echo "  run         - Run codedoc on a directory (use PATH=dir)"
	@echo "  test        - Run all tests"
	@echo "  install     - Install codedoc to GOPATH/bin"
	@echo "  fmt         - Format Go code"
	@echo "  lint        - Run linters"
	@echo "  clean       - Remove build artifacts"
	@echo "  help        - Show this help message"

build:
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p $(BUILD_DIR)
	@go build -o $(BUILD_DIR)/$(BINARY_NAME) $(MAIN_PATH)
	@echo "Binary built: $(BUILD_DIR)/$(BINARY_NAME)"

run: build
	@echo "Running codedoc on $(PATH_ARG)..."
	@$(BUILD_DIR)/$(BINARY_NAME) generate --path $(PATH_ARG)

test:
	@echo "Running tests..."
	@go test -v -race ./...

test-coverage:
	@echo "Running tests with coverage..."
	@go test -v -race -coverprofile=coverage.out ./...
	@go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

install:
	@echo "Installing $(BINARY_NAME)..."
	@go install $(MAIN_PATH)
	@echo "$(BINARY_NAME) installed to $$GOPATH/bin"

fmt:
	@echo "Formatting code..."
	@go fmt ./...
	@echo "Code formatted"

lint:
	@echo "Running linters..."
	@if command -v golangci-lint &> /dev/null; then \
		golangci-lint run ./...; \
	else \
		echo "golangci-lint not installed. Install with:"; \
		echo "  go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest"; \
		go vet ./...; \
	fi

clean:
	@echo "Cleaning build artifacts..."
	@rm -rf $(BUILD_DIR)
	@rm -f coverage.out coverage.html
	@rm -rf .codedoc-cache
	@echo "Clean complete"

deps:
	@echo "Downloading dependencies..."
	@go mod download
	@go mod tidy
	@echo "Dependencies updated"

dev: fmt lint test build
	@echo "Development build complete"

demo: build
	@echo "Running demo on fixtures/tiny-repo..."
	@mkdir -p fixtures/tiny-repo
	@$(BUILD_DIR)/$(BINARY_NAME) generate --path fixtures/tiny-repo --dry-run

demo-with-llm: build
	@echo "Running demo with LLM on fixtures/tiny-repo..."
	@mkdir -p fixtures/tiny-repo
	@$(BUILD_DIR)/$(BINARY_NAME) generate --path fixtures/tiny-repo

all: clean deps fmt lint test build
	@echo "Full build complete"