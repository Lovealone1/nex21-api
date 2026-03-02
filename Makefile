.PHONY: dev server worker tidy

dev:
	air

server:
	go run ./cmd/server

worker:
	go run ./cmd/worker

tidy:
	go mod tidy