-include .env
.DEFAULT_GOAL := help

.PHONY: help up down db backend-run backend-test backend-vet frontend-dev frontend-test frontend-build test test-integration e2e

help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## ' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  %-16s %s\n", $$1, $$2}'

up: ## Build and start the full stack (db, backend, web)
	docker compose up --build

down: ## Stop the stack
	docker compose down

db: ## Start only the database
	docker compose up -d db

backend-run: ## Run the backend locally (needs DATABASE_URL, see .env.example)
	cd backend && go run ./cmd/server

backend-test: ## Run backend tests
	cd backend && go test ./...

backend-vet: ## Run go vet on the backend
	cd backend && go vet ./...

frontend-dev: ## Start the Vite dev server on :3000
	cd frontend && npm run dev

frontend-test: ## Run frontend tests
	cd frontend && npm test

frontend-build: ## Typecheck and build the frontend
	cd frontend && npm run build

test: backend-test frontend-test ## Run all tests

test-integration: ## Run all backend tests incl. database tests, in containers (needs Docker only)
	docker compose -f docker-compose.test.yml run --rm backend-tests; status=$$?; \
		docker compose -f docker-compose.test.yml down; exit $$status

e2e: ## Run end-to-end checks against the running stack (make up first)
	BASE=http://localhost:$(or $(BACKEND_PORT),8080)/api/v1 MAIL=http://localhost:$(or $(MAILPIT_PORT),8025)/api/v1 ORIGIN=http://localhost:$(or $(WEB_PORT),3000) scripts/e2e/auth.sh
	BASE=http://localhost:$(or $(BACKEND_PORT),8080)/api/v1 MAIL=http://localhost:$(or $(MAILPIT_PORT),8025)/api/v1 ORIGIN=http://localhost:$(or $(WEB_PORT),3000) scripts/e2e/tracking.sh
