.PHONY: dev server worker tidy

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