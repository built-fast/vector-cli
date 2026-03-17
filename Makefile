VERSION ?= dev
COMMIT  := $(shell git rev-parse --short HEAD)
DATE    := $(shell date -u +%Y-%m-%dT%H:%M:%SZ)

VERSION_PKG := github.com/built-fast/vector-cli/internal/version
LDFLAGS := -s -w \
           -X $(VERSION_PKG).Version=$(VERSION) \
           -X $(VERSION_PKG).Commit=$(COMMIT) \
           -X $(VERSION_PKG).Date=$(DATE)

.PHONY: build test lint clean check test-e2e surface check-surface check-skill-drift \
        fmt fmt-check vet tidy tidy-check race-test vuln replace-check release-check

build:
	CGO_ENABLED=0 go build -trimpath -ldflags "$(LDFLAGS)" -o bin/vector ./cmd/vector

test:
	go test ./...

lint:
	golangci-lint run

vet:
	go vet ./...

fmt:
	gofmt -s -w .

fmt-check:
	@test -z "$$(gofmt -s -l . | tee /dev/stderr)" || (echo "Code is not formatted. Run 'make fmt'" && exit 1)

race-test:
	go test -race -count=1 ./...

tidy:
	go mod tidy

tidy-check:
	@set -e; cp go.mod go.mod.tidycheck; cp go.sum go.sum.tidycheck; \
	restore() { mv go.mod.tidycheck go.mod; mv go.sum.tidycheck go.sum; }; \
	if ! go mod tidy; then \
		restore; \
		echo "'go mod tidy' failed. Restored original go.mod/go.sum."; \
		exit 1; \
	fi; \
	if ! git diff --quiet -- go.mod go.sum; then \
		restore; \
		echo "go.mod/go.sum are not tidy. Run 'make tidy' and commit the result."; \
		exit 1; \
	fi; \
	rm -f go.mod.tidycheck go.sum.tidycheck

vuln:
	@echo "Running govulncheck..."
	govulncheck ./...

replace-check:
	@if grep -q '^[[:space:]]*replace[[:space:]]' go.mod; then \
		echo "ERROR: go.mod contains replace directives"; \
		grep '^[[:space:]]*replace[[:space:]]' go.mod; \
		echo ""; \
		echo "Remove replace directives before releasing."; \
		exit 1; \
	fi
	@echo "Replace check passed (no local replace directives)"

clean:
	rm -rf bin/

test-e2e:
	./e2e/run.sh

surface:
	go test ./internal/cli/ -run TestSurface -update

check-surface:
	go test ./internal/cli/ -run TestSurface -v

check-skill-drift:
	./scripts/check-skill-drift.sh

check: fmt-check vet lint test test-e2e check-surface check-skill-drift tidy-check

release-check: check replace-check vuln race-test
