BINARY     = bin/server
MODULE     = github.com/99minutos/shipping-system
BUILD_FLAGS = -ldflags="-s -w"
DOCKER_COMPOSE = docker compose

.PHONY: help build swagger run test test-race test-coverage lint fmt deps clean \
        docker-up docker-down docker-logs docker-build

help: ## Show available commands
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2}'

build: ## Build binary → ./bin/server
	go build $(BUILD_FLAGS) -o $(BINARY) ./cmd/server

swagger: ## Generate Swagger docs (requires swag CLI: go install github.com/swaggo/swag/cmd/swag@latest)
	swag init -g cmd/server/main.go --parseInternal -o docs

run: ## Run locally (requires MongoDB + Redis). Loads configs/.env.local
	set -a && source configs/.env.local && set +a && go run ./cmd/server

test: ## Run all tests
	go test ./...

test-race: ## Run tests with race detector (required before commit)
	go test -race ./...

test-k6: ## Run K6 integration tests (requires k6 CLI and a running API at BASE_URL)
	k6 run test/k6/auth.test.js
	k6 run test/k6/shipments.test.js
	k6 run test/k6/events.test.js
	k6 run test/k6/e2e.test.js

test-k6-e2e: ## Run only the end-to-end K6 scenario
	k6 run test/k6/e2e.test.js

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
	$(DOCKER_COMPOSE) build api

docker-up: ## Start MongoDB + Redis + API
	@cp -n configs/.env.example configs/.env 2>/dev/null || true
	$(DOCKER_COMPOSE) up -d
	@echo "→ API:   http://localhost:8080/health"
	@echo "→ Mongo: localhost:27017"
	@echo "→ Redis: localhost:6378"

docker-down: ## Stop all services
	$(DOCKER_COMPOSE) down

docker-logs: ## Follow API logs
	$(DOCKER_COMPOSE) logs -f api

docker-clean: ## Stop services and remove volumes
	$(DOCKER_COMPOSE) down -v
