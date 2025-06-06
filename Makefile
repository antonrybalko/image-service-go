# Makefile for image-service-go

# Variables
APP_NAME := image-service
MAIN_PATH := ./cmd/server
BIN_DIR := ./bin
COVERAGE_FILE := coverage.out
COVERAGE_HTML := coverage.html
DOCKER_IMAGE := image-service-go
DOCKER_TAG := latest

# Go commands
GO := go
GOTEST := $(GO) test
GOBUILD := $(GO) build
GORUN := $(GO) run
GOCLEAN := $(GO) clean
GOMOD := $(GO) mod
GOGET := $(GO) get
GOLINT := golangci-lint

# Build flags
BUILD_FLAGS := -v
LDFLAGS := -ldflags="-s -w"

.PHONY: all build clean test coverage lint deps run docker-build docker-run help

# Default target
all: clean deps lint test build

# Build the application
build:
	@echo "Building $(APP_NAME)..."
	@mkdir -p $(BIN_DIR)
	$(GOBUILD) $(BUILD_FLAGS) $(LDFLAGS) -o $(BIN_DIR)/$(APP_NAME) $(MAIN_PATH)
	@echo "Build complete: $(BIN_DIR)/$(APP_NAME)"

# Clean build artifacts
clean:
	@echo "Cleaning..."
	$(GOCLEAN)
	rm -rf $(BIN_DIR)
	rm -f $(COVERAGE_FILE) $(COVERAGE_HTML)

# Run tests
test:
	@echo "Running tests..."
	$(GOTEST) -v ./...

# Generate test coverage
coverage:
	@echo "Generating test coverage..."
	$(GOTEST) -coverprofile=$(COVERAGE_FILE) -covermode=atomic ./...
	$(GO) tool cover -html=$(COVERAGE_FILE) -o $(COVERAGE_HTML)
	@echo "Coverage report generated: $(COVERAGE_HTML)"
	@echo "Coverage summary:"
	@$(GO) tool cover -func=$(COVERAGE_FILE)

# Run linter
lint:
	@echo "Running linter..."
	@if ! command -v $(GOLINT) &> /dev/null; then \
		echo "golangci-lint not found, installing..."; \
		curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(shell go env GOPATH)/bin v1.54.2; \
	fi
	$(GOLINT) run ./...

# Download dependencies
deps:
	@echo "Downloading dependencies..."
	$(GOMOD) download
	$(GOMOD) tidy

# Run the application locally
run:
	@echo "Running $(APP_NAME) locally..."
	$(GORUN) $(MAIN_PATH)

# Run with hot reload using air (if installed)
dev:
	@echo "Starting development server with hot reload..."
	@if ! command -v air &> /dev/null; then \
		echo "air not found, installing..."; \
		$(GOGET) -u github.com/cosmtrek/air; \
	fi
	air -c .air.toml

# Build Docker image
docker-build:
	@echo "Building Docker image $(DOCKER_IMAGE):$(DOCKER_TAG)..."
	docker build -t $(DOCKER_IMAGE):$(DOCKER_TAG) .

# Run Docker container
docker-run:
	@echo "Running Docker container..."
	docker run -p 8080:8080 --env-file .env $(DOCKER_IMAGE):$(DOCKER_TAG)

# Mock generation
mocks:
	@echo "Generating mocks..."
	@if ! command -v mockgen &> /dev/null; then \
		echo "mockgen not found, installing..."; \
		$(GOGET) -u github.com/golang/mock/mockgen; \
	fi
	@echo "Add your mockgen commands here"

# Help target
help:
	@echo "Available targets:"
	@echo "  all          - Clean, download dependencies, lint, test, and build"
	@echo "  build        - Build the application"
	@echo "  clean        - Clean build artifacts"
	@echo "  test         - Run tests"
	@echo "  coverage     - Generate test coverage report"
	@echo "  lint         - Run linter"
	@echo "  deps         - Download dependencies"
	@echo "  run          - Run the application locally"
	@echo "  dev          - Run with hot reload (requires air)"
	@echo "  docker-build - Build Docker image"
	@echo "  docker-run   - Run Docker container"
	@echo "  mocks        - Generate mocks for testing"
	@echo "  help         - Show this help message"
