BINARY_NAME := gpc
VERSION ?= dev
COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo unknown)
DATE ?= $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
GO ?= go
LDFLAGS := -X 'github.com/leszko11/google-play-console-cli/cmd.Version=$(VERSION)' -X 'github.com/leszko11/google-play-console-cli/cmd.Commit=$(COMMIT)' -X 'github.com/leszko11/google-play-console-cli/cmd.Date=$(DATE)'
GO_BUILD := $(GO) build -ldflags "$(LDFLAGS)"

.PHONY: build test lint format coverage benchmark generate-command-docs check-command-docs generate-llms-txt check-llms-txt generate-openapi-paths generate-openapi-coverage check-openapi-coverage check-openapi-drift dev

build:
	$(GO_BUILD) -o build/$(BINARY_NAME) .

test:
	GPC_BYPASS_KEYCHAIN=1 $(GO) test ./...

coverage:
	GPC_BYPASS_KEYCHAIN=1 $(GO) test -coverprofile=coverage.out ./...
	$(GO) tool cover -func=coverage.out

benchmark:
	GPC_BYPASS_KEYCHAIN=1 $(GO) test -run=^$$ -bench=. -benchmem ./internal/cli/shared ./internal/cli/listing ./internal/cli/release

lint:
	go vet ./...

format:
	gofmt -w .

generate-command-docs:
	python3 scripts/generate-command-docs.py

check-command-docs:
	python3 scripts/check-command-docs.py

generate-llms-txt:
	python3 scripts/generate-llms-txt.py

check-llms-txt:
	python3 scripts/check-llms-txt.py

generate-openapi-paths:
	python3 scripts/update-openapi-paths.py --fetch

generate-openapi-coverage:
	python3 scripts/generate-openapi-coverage.py

check-openapi-coverage:
	python3 scripts/check-openapi-coverage.py

check-openapi-drift:
	python3 scripts/check-openapi-drift.py --fetch

dev: format check-command-docs check-llms-txt check-openapi-coverage lint test build
