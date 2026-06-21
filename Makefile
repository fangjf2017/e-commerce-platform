.PHONY: build migrate test lint docker-up docker-down run generate

build:
	go build -o bin/server ./cmd/server

migrate:
	go run ./cmd/migrate up

test:
	go test ./...

lint:
	golangci-lint run

docker-up:
	docker-compose up -d

docker-down:
	docker-compose down

run:
	go run ./cmd/server

generate:
	go generate ./...
