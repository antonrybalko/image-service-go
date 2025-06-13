# Makefile for image-service-go

# Variables
APP_NAME := image-service
MAIN_PATH := ./cmd/server
BIN_DIR := ./bin
BIN_PATH := $(BIN_DIR)/$(APP_NAME)
DOCKER_IMAGE := antonrybalko/image-service-go
DOCKER_TAG := latest
COVERAGE_FILE := coverage.out
COVERAGE_HTML := coverage.html

# Go commands
GO := go
GOTEST := $(GO) test
GOBUILD := $(GO) build
GORUN := $(GO) run
GOMOD := $(GO) mod
GOLINT := $(shell go env GOPATH)/bin/golangci-lint

# Default target
.PHONY: help
help:
	@echo "Available commands:"
	@echo "  build         - Build the application"
	@echo "  run           - Run the service locally"
	@echo "  test          - Run tests with coverage"
	@echo "  docker-build  - Build Docker image"
	@echo "  docker-run    - Run the Docker container"
	@echo "  lint          - Run linters"
	@echo "  clean         - Remove build artifacts"
	@echo "  deps          - Install dependencies"
	@echo "  mock          - Generate mocks for testing"
	@echo "  help          - Show this help message"

# Build the application
.PHONY: build
build:
	@echo "Building $(APP_NAME)..."
	@mkdir -p $(BIN_DIR)
	$(GOBUILD) -o $(BIN_PATH) $(MAIN_PATH)
	@echo "Build complete: $(BIN_PATH)"

# Run the service locally
.PHONY: run
run:
	@echo "Running $(APP_NAME)..."
	$(GORUN) $(MAIN_PATH)

# Run tests with coverage
.PHONY: test
test:
	@echo "Running tests with coverage..."
	$(GOTEST) -coverprofile=$(COVERAGE_FILE) -covermode=atomic ./...
	$(GO) tool cover -html=$(COVERAGE_FILE) -o $(COVERAGE_HTML)
	@echo "Coverage report generated: $(COVERAGE_HTML)"

# Build Docker image
.PHONY: docker-build
docker-build:
	@echo "Building Docker image $(DOCKER_IMAGE):$(DOCKER_TAG)..."
	docker build -t $(DOCKER_IMAGE):$(DOCKER_TAG) .

# Run Docker container
.PHONY: docker-run
docker-run:
	@echo "Running Docker container $(DOCKER_IMAGE):$(DOCKER_TAG)..."
	docker run -p 8080:8080 --env-file .env $(DOCKER_IMAGE):$(DOCKER_TAG)

# Run linters
.PHONY: lint
lint:
	@echo "Running linters..."
	$(GOLINT) run

# Clean build artifacts
.PHONY: clean
clean:
	@echo "Cleaning build artifacts..."
	rm -rf $(BIN_DIR)
	rm -f $(COVERAGE_FILE) $(COVERAGE_HTML)

# Install dependencies
.PHONY: deps
deps:
	@echo "Installing dependencies..."
	$(GOMOD) download
	$(GOMOD) tidy
	@if ! command -v $(GOLINT) &> /dev/null; then \
		echo "Installing golangci-lint..."; \
		curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $$(go env GOPATH)/bin; \
	fi
	@if ! command -v mockgen &> /dev/null; then \
		echo "Installing mockgen..."; \
		go install github.com/golang/mock/mockgen@latest; \
	fi

# Generate mocks for testing
.PHONY: mock
mock:
	@echo "Generating mocks..."
	mkdir -p internal/mocks
	mockgen -destination=internal/mocks/storage_mock.go -package=mocks github.com/antonrybalko/image-service-go/internal/storage S3Interface
