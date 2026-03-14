VERSION ?= dev
COMMIT  := $(shell git rev-parse --short HEAD)
DATE    := $(shell date -u +%Y-%m-%dT%H:%M:%SZ)

LDFLAGS := -X github.com/built-fast/vector-cli/internal/version.Version=$(VERSION) \
           -X github.com/built-fast/vector-cli/internal/version.Commit=$(COMMIT) \
           -X github.com/built-fast/vector-cli/internal/version.Date=$(DATE)

.PHONY: build test lint clean check test-e2e

build:
	CGO_ENABLED=0 go build -ldflags "$(LDFLAGS)" -o bin/vector ./cmd/vector

test:
	go test ./...

lint:
	golangci-lint run

clean:
	rm -rf bin/

test-e2e:
	./e2e/run.sh

check: lint test test-e2e
