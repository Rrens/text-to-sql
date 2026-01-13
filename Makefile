# Text-to-SQL Platform

.PHONY: all build run test clean lint docker-build docker-up docker-down setup

# Version
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_TIME := $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
LDFLAGS := -s -w -X main.Version=$(VERSION) -X main.BuildTime=$(BUILD_TIME)

# Go parameters
GOCMD := go
GOBUILD := $(GOCMD) build
GOTEST := $(GOCMD) test
GOMOD := $(GOCMD) mod
BINARY_NAME := server
BINARY_PATH := bin/$(BINARY_NAME)

all: clean lint test build

## Build commands
build:
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p bin
	$(GOBUILD) -ldflags "$(LDFLAGS)" -o $(BINARY_PATH) ./cmd/server
	@echo "Built $(BINARY_PATH)"

build-linux:
	@echo "Building for Linux AMD64..."
	@mkdir -p bin
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 $(GOBUILD) -ldflags "$(LDFLAGS)" -o bin/$(BINARY_NAME)-linux-amd64 ./cmd/server

build-all:
	@bash scripts/build.sh

## Run commands
run: build
	@echo "Starting server..."
	./$(BINARY_PATH)

run-dev:
	@echo "Starting server in development mode..."
	CONFIG_PATH=configs/config.local.yaml $(GOCMD) run ./cmd/server

## Test commands
test:
	@echo "Running tests..."
	$(GOTEST) -v -race -count=1 ./...

test-short:
	$(GOTEST) -short -count=1 ./...

test-coverage:
	@echo "Running tests with coverage..."
	$(GOTEST) -v -race -coverprofile=coverage.out -covermode=atomic ./...
	$(GOCMD) tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"

bench:
	$(GOTEST) -bench=. -benchmem ./...

## Lint commands
lint:
	@echo "Running linters..."
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run ./...; \
	else \
		echo "golangci-lint not installed, running go vet..."; \
		$(GOCMD) vet ./...; \
	fi

fmt:
	$(GOCMD) fmt ./...
	@if command -v goimports >/dev/null 2>&1; then \
		goimports -w .; \
	fi

## Dependency commands
deps:
	$(GOMOD) download
	$(GOMOD) tidy

deps-update:
	$(GOMOD) get -u ./...
	$(GOMOD) tidy

## Clean commands
clean:
	@echo "Cleaning..."
	rm -rf bin/
	rm -f coverage.out coverage.html
	$(GOCMD) clean -cache -testcache

## Docker commands
docker-build:
	@echo "Building Docker image..."
	docker build -t text-to-sql:$(VERSION) -t text-to-sql:latest -f deployments/docker/Dockerfile .

docker-up:
	@echo "Starting Docker services..."
	docker-compose -f deployments/docker/docker-compose.yaml up -d

docker-down:
	@echo "Stopping Docker services..."
	docker-compose -f deployments/docker/docker-compose.yaml down

docker-logs:
	docker-compose -f deployments/docker/docker-compose.yaml logs -f

docker-ps:
	docker-compose -f deployments/docker/docker-compose.yaml ps

## Database commands
migrate-up:
	@echo "Running migrations..."
	CONFIG_PATH=configs/config.local.yaml go run cmd/migrate/main.go

migrate-down:
	@echo "Rolling back migrations..."
	docker exec -i postgres_db psql -U texttosql -d texttosql < migrations/001_initial.down.sql

db-shell:
	docker exec -it postgres_db psql -U texttosql -d texttosql

## Setup commands
setup:
	@bash scripts/setup.sh

## Documentation
docs:
	@echo "API docs available at: docs/openapi.yaml"
	@if command -v redocly >/dev/null 2>&1; then \
		redocly preview-docs docs/openapi.yaml; \
	else \
		echo "Install redocly-cli for live preview: npm install -g @redocly/cli"; \
	fi

## Integration testing
test-api:
	@bash scripts/test-api.sh

## Help
help:
	@echo "Text-to-SQL Platform Makefile"
	@echo ""
	@echo "Usage:"
	@echo "  make <target>"
	@echo ""
	@echo "Targets:"
	@echo "  build        Build the binary"
	@echo "  run          Build and run the server"
	@echo "  run-dev      Run with local config"
	@echo "  test         Run all tests"
	@echo "  test-coverage Run tests with coverage"
	@echo "  lint         Run linters"
	@echo "  fmt          Format code"
	@echo "  clean        Clean build artifacts"
	@echo "  docker-build Build Docker image"
	@echo "  docker-up    Start Docker services"
	@echo "  docker-down  Stop Docker services"
	@echo "  setup        Run initial setup"
	@echo "  test-api     Run API integration tests"
	@echo "  help         Show this help"
