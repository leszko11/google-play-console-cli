BINARY_NAME := gpc
VERSION ?= dev
COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo unknown)
DATE ?= $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
LDFLAGS := -X 'github.com/leszko11/google-play-console-cli/cmd.Version=$(VERSION)' -X 'github.com/leszko11/google-play-console-cli/cmd.Commit=$(COMMIT)' -X 'github.com/leszko11/google-play-console-cli/cmd.Date=$(DATE)'
GO_BUILD := go build -ldflags "$(LDFLAGS)"

.PHONY: build test lint format dev

build:
	$(GO_BUILD) -o build/$(BINARY_NAME) .

test:
	go test ./...

lint:
	go vet ./...

format:
	gofmt -w .

dev: format lint test build
