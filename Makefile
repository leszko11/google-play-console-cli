BINARY_NAME := gpc
VERSION ?= dev
COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo unknown)
DATE ?= $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
LDFLAGS := -X 'github.com/leszko11/google-play-console-cli/cmd.Version=$(VERSION)' -X 'github.com/leszko11/google-play-console-cli/cmd.Commit=$(COMMIT)' -X 'github.com/leszko11/google-play-console-cli/cmd.Date=$(DATE)'
GO_BUILD := go build -ldflags "$(LDFLAGS)"

.PHONY: build test lint format generate-command-docs check-command-docs generate-openapi-paths generate-openapi-coverage check-openapi-coverage dev

build:
	$(GO_BUILD) -o build/$(BINARY_NAME) .

test:
	go test ./...

lint:
	go vet ./...

format:
	gofmt -w .

generate-command-docs:
	python3 scripts/generate-command-docs.py

check-command-docs:
	python3 scripts/check-command-docs.py

generate-openapi-paths:
	python3 scripts/update-openapi-paths.py --fetch

generate-openapi-coverage:
	python3 scripts/generate-openapi-coverage.py

check-openapi-coverage:
	python3 scripts/check-openapi-coverage.py

dev: format check-command-docs check-openapi-coverage lint test build
