.PHONY: help build run test clean docker-build docker-run deps lint

# Variables
APP_NAME=energy-metering-worker
DOCKER_IMAGE=$(APP_NAME):latest
GO_FILES=$(shell find . -name '*.go' -type f)

help: ## Show this help message
	@echo "Usage: make [target]"
	@echo ""
	@echo "Targets:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  %-20s %s\n", $$1, $$2}'

deps: ## Download Go dependencies
	go mod download
	go mod verify

build: deps ## Build the worker binary
	CGO_ENABLED=0 go build -o bin/worker ./cmd/worker

run: ## Run the worker locally
	go run ./cmd/worker

test: ## Run tests
	go test -v -race -coverprofile=coverage.out ./...

test-coverage: test ## Run tests with coverage report
	go tool cover -html=coverage.out

lint: ## Run linter
	golangci-lint run ./...

clean: ## Clean build artifacts
	rm -rf bin/
	rm -f coverage.out

docker-build: ## Build Docker image
	docker build -t $(DOCKER_IMAGE) .

docker-run: ## Run Docker container
	docker run --env-file .env $(DOCKER_IMAGE)

docker-compose-up: ## Start services with docker-compose
	docker-compose up -d

docker-compose-down: ## Stop services
	docker-compose down

docker-compose-logs: ## Show logs
	docker-compose logs -f worker

format: ## Format Go code
	go fmt ./...
	goimports -w .

vet: ## Run go vet
	go vet ./...

mod-tidy: ## Tidy go.mod
	go mod tidy

dev: ## Run in development mode with live reload (requires air)
	air

.DEFAULT_GOAL := help
