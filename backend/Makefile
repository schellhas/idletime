.PHONY: tidy build run db-up db-down

tidy:
	go mod tidy

build:
	go build ./...

run:
	go run ./cmd/server

db-up:
	docker compose up -d postgres

db-down:
	docker compose down
