.DEFAULT_GOAL := help

.PHONY: help up down db backend-run backend-test backend-vet frontend-dev frontend-test frontend-build test

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
