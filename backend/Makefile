.PHONY: build test run lint

build:
	go build -o bin/gateway ./cmd/gateway

test:
	go test ./... -v -race

run:
	go run ./cmd/gateway

lint:
	golangci-lint run ./...
