BINARY_NAME := gpc

.PHONY: build test lint format dev

build:
	go build -o build/$(BINARY_NAME) .

test:
	go test ./...

lint:
	go vet ./...

format:
	gofmt -w .

dev: format lint test build
