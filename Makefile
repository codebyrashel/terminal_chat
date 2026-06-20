.PHONY: db-start db-stop db-reset run-server test

db-start:
	docker-compose up -d postgres
	@echo "Waiting for PostgreSQL..."
	@sleep 3
	@docker exec tchat-db pg_isready -U chatapp -d terminal_chat

db-stop:
	docker-compose down

db-reset:
	docker-compose down -v
	docker-compose up -d postgres
	@sleep 3
	@docker exec tchat-db pg_isready -U chatapp -d terminal_chat

run-server:
	DB_HOST=localhost DB_PORT=5432 DB_USER=chatapp DB_PASSWORD=chatapp123 DB_NAME=terminal_chat JWT_SECRET=dev-secret SERVER_PORT=8080 go run cmd/server/main.go

test:
	curl -X POST http://localhost:8080/api/register -H "Content-Type: application/json" -d '{"username":"test","password":"test123"}'


run-client:
	SERVER_URL=http://localhost:8080 go run cmd/client/main.go

build:
	go build -o bin/server cmd/server/main.go
	go build -o bin/client cmd/client/main.go