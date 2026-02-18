BINARY     = bin/server
MODULE     = github.com/99minutos/shipping-system
BUILD_FLAGS = -ldflags="-s -w"

.PHONY: help build run test test-race test-coverage lint fmt deps clean \
        docker-up docker-down docker-logs docker-build

help: ## Show available commands
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2}'

build: ## Build binary → ./bin/server
	go build $(BUILD_FLAGS) -o $(BINARY) ./cmd/server

run: ## Run locally (requires MongoDB + Redis)
	go run ./cmd/server

test: ## Run all tests
	go test ./...

test-race: ## Run tests with race detector (required before commit)
	go test -race ./...

test-coverage: ## Generate HTML coverage report
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "→ open coverage.html"

lint: ## Run golangci-lint
	golangci-lint run ./...

fmt: ## Format all Go files
	go fmt ./...

deps: ## Download and tidy dependencies
	go mod download && go mod tidy

clean: ## Remove build artifacts
	rm -rf bin/ coverage.out coverage.html

# ── Docker ───────────────────────────────────────────────────────────────────

docker-build: ## Build the API Docker image
	docker compose build api

docker-up: ## Start MongoDB + Redis + API
	@cp -n .env.example .env 2>/dev/null || true
	docker compose up -d
	@echo "→ API:   http://localhost:8080/health"
	@echo "→ Mongo: localhost:27017"
	@echo "→ Redis: localhost:6379"

docker-down: ## Stop all services
	docker compose down

docker-logs: ## Follow API logs
	docker compose logs -f api

docker-clean: ## Stop services and remove volumes
	docker compose down -v
