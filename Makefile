.PHONY: dev server worker tidy migrate-up migrate-down

dev:
	air

server:
	go run ./cmd/server

worker:
	go run ./cmd/worker
	
swagger:
	swag init -g cmd/server/main.go 

tidy:
	go mod tidy

migrate-up:
	go run ./cmd/migrate up

migrate-down:
	go run ./cmd/migrate down $(STEPS)
